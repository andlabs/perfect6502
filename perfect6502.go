/*
 Copyright (c) 2010 Michael Steil, Brian Silverman, Barry Silverman
 Copyright (c) 2013 Pietro Gagliardi

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package main

import (
	"fmt"
	// ...
)

func init() {
	DEBUG = true		// declared in runtime.go
}

/************************************************************
 *
 * Libc Functions and Basic Data Types
 *
 ************************************************************/

#include "perfect6502.h"

type BOOL uint

const (
	YES = 1
	NO = 0
)

/************************************************************
 *
 * 6502 Description: Nodes, Transistors and Probes
 *
 ************************************************************/

/* the 6502 consists of this many nodes and transistors */
var (
	NODES = len(segdefs)
	TRANSISTORS = len(transdefs)
)

/************************************************************
 *
 * Global Data Types
 *
 ************************************************************/

/* the smallest types to fit the numbers */
type nodenum_t uint16
type transnum_t uint16
type count_t uint16
// TODO(andlabs) - these will all need to be made a single name and type to simplify everything later (preferaly to just uint64 for future compatibility)

/************************************************************
 *
 * Bitmap Data Structures and Algorithms
 *
 ************************************************************/

type bitmap_t uint64
const (
	BITMAP_SHIFT = 6
	BITMAP_MASK = 63
)

func WORDS_FOR_BITS(a uint64) uint64 {
	return (a / (4 * 8)) + 1
}

func DECLARE_BITMAP(count uint64) []bitmap_t {
	return make([]bitmap_t, WORDS_FOR_BITS(count))
}

func bitmap_clear(bitmap []bitmap_t, count count_t) {
	for i := uint64(0); i < WORDS_FOR_BITS(uint64(count)); i++ {
		bitmap[i] = 0
	}
}

func set_bitmap(bitmap []bitmap_t, index uint64, state BOOL) {
	if state != 0 {
		bitmap[index >> BITMAP_SHIFT] |= (1 << (index & BITMAP_MASK))
	} else {
		bitmap[index >> BITMAP_SHIFT] &^= (1 << (index & BITMAP_MASK))
	}
}

func get_bitmap(bitmap []bitmap_t, index uint64) BOOL {
	return BOOL((bitmap[index >> BITMAP_SHIFT] >> (index & BITMAP_MASK)) & 1)
}

/************************************************************
 *
 * Data Structures for Nodes
 *
 ************************************************************/

/* everything that describes a node */
var (
	nodes_pullup = DECLARE_BITMAP(NODES)
	nodes_pulldown = DECLARE_BITMAP(NODES)
	nodes_value = DECLARE_BITMAP(NODES)
	nodes_gates		[NODES][NODES]nodenum_t
	nodes_c1c2s		[NODES][2*NODES]nodenum_t
	nodes_gatecount	[NODES]count_t
	nodes_c1c2count	[NODES]count_t
	nodes_dependants	[NODES]nodenum_t
	nodes_dependant	[NODES][NODES]nodenum_t
)

/*
 * The "value" propertiy of VCC and GND is never evaluated in the code,
 * so we don't bother initializing it properly or special-casing writes.
 */

func set_nodes_pullup(t transnum_t, state BOOL) {
	set_bitmap(nodes_pullup, uint64(t), state)
}

func get_nodes_pullup(t transnum_t) BOOL {
	return get_bitmap(nodes_pullup, uint64(t))
}

func set_nodes_pulldown(t transnum_t, state BOOL) {
	set_bitmap(nodes_pulldown, uint64(t), state)
}

func get_nodes_pulldown(t transnum_t) BOOL {
	return get_bitmap(nodes_pulldown, uint64(t))
}

func set_nodes_value(t transnum_t, state BOOL) {
	set_bitmap(nodes_value, uint64(t), state)
}

func get_nodes_value(t transnum_t) BOOL {
	return get_bitmap(nodes_value, uint64(t))
}

/************************************************************
 *
 * Data Structures and Algorithms for Transistors
 *
 ************************************************************/

/* everything that describes a transistor */
var (
	transistors_gate	[TRANSISTORS]nodenum_t
	transistors_c1		[TRANSISTORS]nodenum_t
	transistors_c2		[TRANSISTORS]nodenum_t
	transistors_on = DECLARE_BITMAP(TRANSISTORS)
)

//#ifdef BROKEN_TRANSISTORS
var broken_transistor = ^transnum_t(0)		// TODO const?
//#endif

func set_transistors_on(t transnum_t, state BOOL) {
//#ifdef BROKEN_TRANSISTORS
	if t == broken_transistor {
		return
	}
//#endif
	set_bitmap(transistors_on, uint64(t), state)
}

func get_transistors_on(t transnum_t) BOOL {
	return get_bitmap(transistors_on, uint64(t))
}

/************************************************************
 *
 * Data Structures and Algorithms for Lists
 *
 ************************************************************/

// TODO(andlabs) - can this whole thing be simplified to just slice logic?

/* list of nodes that need to be recalculated */
type list_t struct {
	list		[]nodenum_t
	count	count_t
}

/* the nodes we are working with */
var (
	list1		[NODES]nodenum_t
	listin = list_t{
		list:	 list1[:],
	}
)

/* the indirect nodes we are collecting for the next run */
var (
	list2		[NODES]nodenum_t
	listout = list_t{
		list:	list2[:],
	}
)

func listin_get(i count_t) nodenum_t {
	return listin.list[i]
}

func listin_count() count_t {
	return listin.count
}

func lists_switch() {
	listin, listout = listout, listin
}

func listout_clear() {
	listout.count = 0;
}

func listout_add(i nodenum_t) {
	listout.list[listout.count] = i
	listout.count++
}

/************************************************************
 *
 * Data Structures and Algorithms for Groups of Nodes
 *
 ************************************************************/

/*
 * a group is a set of connected nodes, which consequently
 * share the same potential
 *
 * we use an array and a count for O(1) insert and
 * iteration, and a redundant bitmap for O(1) lookup
 */
var (
	group		[NODES]nodenum_t
	groupcount	count_t
	groupbitmap = DECLARE_BITMAP(NODES)
)

// TODO(andlabs) - again, drop groupcount in favor of just len()? or will we wind up in a situation in the future where we have too many nodes...

func group_clear() {
	groupcount = 0
	bitmap_clear(groupbitmap, NODES)
}

func group_add(i nodenum_t) {
	group[groupcount] = i
	groupcount++
	set_bitmap(groupbitmap, uint64(i), 1)
}

func group_get(count_t n) nodenum_t {
	return group[n]
}

func group_contains(el nodenum_t) BOOL {
	return get_bitmap(groupbitmap, uint64(el))
}

func group_count() count_t {
	return groupcount
}

/************************************************************
 *
 * Node and Transistor Emulation
 *
 ************************************************************/

// TODO(andlabs) - make these into actual booleans (for BOOL, need to handle case where some of the segdefs are not either 0 or 1... 1701, for instance)... same for group_contains and some other stuff...
var (
	group_contains_pullup		BOOL
	group_contains_pulldown	BOOL
	group_contains_hi			BOOL
)

func addNodeToGroup(n nodenum_t) {
	if group_contains(n) == YES {
		return
	}

	group_add(n)

	if get_nodes_pullup(n) != 0 {
		group_contains_pullup = YES
	}
	if get_nodes_pulldown(n) != 0 {
		group_contains_pulldown = YES
	}
	if get_nodes_value(n) != 0 {
		group_contains_hi = YES
	}

	if n == vss || n == vcc {
		return
	}

	// revisit all transistors that are controlled by this node
	for t = count_t(0); t < nodes_c1c2count[n]; t++ {
		tn := nodes_c1c2s[n][t]
		// if the transistor connects c1 and c2...
		if get_transistors_on(tn) != 0 {
			// if original node was connected to c1, continue with c2
			if transistors_c1[tn] == n {
				addNodeToGroup(transistors_c2[tn])
			} else {
				addNodeToGroup(transistors_c1[tn])
			}
		}
	}
}

func addAllNodesToGroup(node nodenum_t) {
	group_clear()

	group_contains_pullup = NO
	group_contains_pulldown = NO
	group_contains_hi = NO

	addNodeToGroup(node)
}

func getGroupValue() BOOL {
	if group_contains(vss) == YES {
		return NO
	}

	if group_contains(vcc) == YES {
		return YES
	}

	if group_contains_pulldown == YES {
		return NO
	}

	if group_contains_pullup == YES {
		return YES
	}

	return group_contains_hi
}

func BOOL_not(b BOOL) BOOL {
	if b == NO {
		return YES
	}
	return NO
}

func recalcNode(node nodenum_t) {
	/*
	 * get all nodes that are connected through
	 * transistors, starting with this one
	 */
	addAllNodesToGroup(node)

	/* get the state of the group */
	newv := getGroupValue()

	/*
	 * - set all nodes to the group state
	 * - check all transistors switched by nodes of the group
	 * - collect all nodes behind toggled transistors
	 *   for the next run
	 */
	for i := count_t(0); i < group_count(); i++ {
		nn := group_get(i)
		if get_nodes_value(nn) != newv {
			set_nodes_value(nn, newv)
			for t := count_t(0); t < nodes_gatecount[nn]; t++ {
				tn := nodes_gates[nn][t]
				set_transistors_on(tn, BOOL_not(get_transistors_on(tn)))
			}
			listout_add(nn)
		}
	}
}

func recalcNodeList(source []nodenum_t, count count_t) {
	listout_clear()

	for i := count_t(0); i < count; i++ {
		recalcNode(source[i])
	}

	lists_switch()

	for j := 0; j < 100; j++ {		// loop limiter (TODO(andlabs) - is this really best?)
		if listin_count() == 0 {
			break
		}

		listout_clear()

		/*
		 * for all nodes, follow their paths through
		 * turned-on transistors, find the state of the
		 * path and assign it to all nodes, and re-evaluate
		 * all transistors controlled by this path, collecting
		 * all nodes that changed because of it for the next run
		 */
		for i := count_t(0); i < listin_count(); i++ {
			n := listin_get(i)
			for g = count_t(0); g < nodes_dependants[n]; g++ {
				recalcNode(nodes_dependant[n][g])
			}
		}
		/*
		 * make the secondary list our primary list, use
		 * the data storage of the primary list as the
		 * secondary list
		 */
		lists_switch()
	}
}

func recalcAllNodes() {
	var temp [NODES]nodenum_t
	for i = count_t(0); i < NODES; i++ {
		temp[i] = i
	}
	recalcNodeList(temp[:], NODES)
}

/************************************************************
 *
 * Node State
 *
 ************************************************************/

func setNode(nn nodenum_t, state BOOL) {
	set_nodes_pullup(nn, state)
	set_nodes_pulldown(nn, !state)
	recalcNodeList([]nodenum_t{ nn }, 1)
}

func isNodeHigh(nn nodenum_t) BOOL {
	return get_nodes_value(nn)
}

/************************************************************
 *
 * Interfacing and Extracting State
 *
 ************************************************************/

func read8(n0,n1,n2,n3,n4,n5,n6,n7 nodenum_t) byte {
	return (byte(isNodeHigh(n0) << 0) |
		byte(isNodeHigh(n1) << 1) |
		byte(isNodeHigh(n2) << 2) |
		byte(isNodeHigh(n3) << 3) |
		byte(isNodeHigh(n4) << 4) |
		byte(isNodeHigh(n5) << 5) |
		byte(isNodeHigh(n6) << 6) |
		byte(isNodeHigh(n7) << 7))
}

func readAddressBus() uint16 {
	return uint16(read8(ab0,ab1,ab2,ab3,ab4,ab5,ab6,ab7)) |
		(uint16(read8(ab8,ab9,ab10,ab11,ab12,ab13,ab14,ab15)) << 8)
}

func readDataBus() byte {
	return read8(db0,db1,db2,db3,db4,db5,db6,db7)
}

var (
	dbnodes = [8]nodenum_t{ db0, db1, db2, db3, db4, db5, db6, db7 }
)

func writeDataBus(d byte) {
	for i := 0; i < 8; i++ {
		setNode(dbnodes[i], BOOL(d & 1))
		d >>= 1
	}
}

func readRW() BOOL {
	return isNodeHigh(rw)
}

func readA() byte {
	return read8(a0,a1,a2,a3,a4,a5,a6,a7)
}

func readX() byte {
	return read8(x0,x1,x2,x3,x4,x5,x6,x7);
}

func readY() byte {
	return read8(y0,y1,y2,y3,y4,y5,y6,y7);
}

func readP() byte {
	return read8(p0,p1,p2,p3,p4,p5,p6,p7);
}

func readIR() byte {
	return read8(notir0,notir1,notir2,notir3,notir4,notir5,notir6,notir7) ^ 0xFF
}

func readSP() byte {
	return read8(s0,s1,s2,s3,s4,s5,s6,s7);
}

func readPCL() byte {
	return read8(pcl0,pcl1,pcl2,pcl3,pcl4,pcl5,pcl6,pcl7);
}

func readPCH() {
	return read8(pch0,pch1,pch2,pch3,pch4,pch5,pch6,pch7);
}

func readPC() uint16 {
	return (uint16(readPCH()) << 8) | uint16(readPCL())
}

/************************************************************
 *
 * Tracing/Debugging
 *
 ************************************************************/

unsigned int cycle;

void
chipStatus()
{
	clk := isNodeHigh(clk0)
	a := readAddressBus()
	d := readDataBus()
	r_w := isNodeHigh(rw)

	fmt.Printf("halfcyc:%d phi0:%d AB:%04X D:%02X RnW:%d PC:%04X A:%02X X:%02X Y:%02X SP:%02X P:%02X IR:%02X",
			cycle,
			clk,
			a,
	        d,
	        r_w,
			readPC(),
			readA(),
			readX(),
			readY(),
			readSP(),
			readP(),
			readIR())

	if clk != 0 {
		if r_w != 0 {
			fmt.Printf(" R$%04X=$%02X", a, memory[a])
		} else {
			fmt.Printf(" W$%04X=$%02X", a, d)
		}
	}

	fmt.Printf("\n")
}

/************************************************************
 *
 * Address Bus and Data Bus Interface
 *
 ************************************************************/

var memory [65536]byte

func mRead(a uint16) byte {
	return memory[a]
}

func mWrite(a uint16_t, d byte) {
	memory[a] = d
}

func handleMemory() {
	if isNodeHigh(rw) == YES {
		writeDataBus(mRead(readAddressBus()))
	} else {
		mWrite(readAddressBus(), readDataBus())
	}
}

/************************************************************
 *
 * Main Clock Loop
 *
 ************************************************************/

func step() {
	clk := isNodeHigh(clk0)

	// invert clock
	setNode(clk0, BOOL_not(clk))

	/* handle memory reads and writes */
	if clk == 0 {
		handleMemory()
	}

	cycle++
}

/************************************************************
 *
 * Initialization
 *
 ************************************************************/

var transistors uint

func add_nodes_dependant(a nodenum_t, b nodenum_t) {
	for g := count_t(0); g < nodes_dependants[a]; g++ {
		if nodes_dependant[a][g] == b {
			return
		}
	}

	nodes_dependant[a][nodes_dependants[a]] = b
	nodes_dependants[a]++
}

func setupNodesAndTransistors() {
	var i count_t

	// copy nodes into r/w data structure
	for i = 0; i < NODES; i++ {
		b := BOOL(NO)
		if segdefs[i] == 1 {
			b = YES
		}
		set_nodes_pullup(i, b)
		nodes_gatecount[i] = 0
		nodes_c1c2count[i] = 0
	}

	// copy transistors into r/w data structure
	j := count_t(0)
	for i = 0; i < TRANSISTORS; i++ {
		gate := nodenum_t(transdefs[i].gate)
		c1 := nodenum_t(transdefs[i].c1)
		c2 := nodenum_t(transdefs[i].c2)
		/* skip duplicate transistors */
		found := BOOL(NO)

		if found == NO {
			transistors_gate[j] = gate
			transistors_c1[j] = c1
			transistors_c2[j] = c2
			j++
		}
	}
	transistors = j
	if DEBUG {
		fmt.Printf("transistors: %d\n", transistors)
	}

	/* cross reference transistors in nodes data structures */
	for i = 0; i < transistors; i++ {
		gate := transistors_gate[i]
		c1 := transistors_c1[i]
		c2 := transistors_c2[i]
		nodes_gates[gate][nodes_gatecount[gate]] = i
		nodes_gatecount[gate]++
		nodes_c1c2s[c1][nodes_c1c2count[c1]] = i
		nodes_c1c2count[c1]++
		nodes_c1c2s[c2][nodes_c1c2count[c2]] = i
		nodes_c1c2count[c2]++
	}

	for i = 0; i < NODES; i++ {
		nodes_dependants[i] = 0
		for g := count_t(0); g < nodes_gatecount[i]; g++ {
			t := nodes_gates[i][g]
			add_nodes_dependant(i, transistors_c1[t])
			add_nodes_dependant(i, transistors_c2[t])
		}
	}
}

func resetChip() {
	// all nodes are down
	for nn = nodenum_t(0); nn < NODES; nn++ {
		set_nodes_value(nn, 0)
	}

	// all transistors are off
	for tn = transnum_t(0); tn < TRANSISTORS; tn++ {
		set_transistors_on(tn, NO)
	}

	setNode(res, 0)
	setNode(clk0, 1)
	setNode(rdy, 1)
	setNode(so, 0)
	setNode(irq, 1)
	setNode(nmi, 1)

	recalcAllNodes()

	// hold RESET for 8 cycles
	for i = 0; i < 16; i++ {
		step()
	}

	// release RESET
	setNode(res, 1)

	cycle = 0
}

func initAndResetChip() {
	// set up data structures for efficient emulation
	setupNodesAndTransistors()

	// set initial state of nodes, transistors, inputs; RESET chip
	resetChip()
}
