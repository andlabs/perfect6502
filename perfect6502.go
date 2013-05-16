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
	"time"
	// ...
)

func init() {
	DEBUG = true		// declared in runtime.go
}

// pin channels
var (
	rdy_chan		= make(chan bool)
	clk1_chan	= make(chan bool)
	irq_chan		= make(chan bool)
	nmi_chan		= make(chan bool)
	sync_chan	= make(chan bool)
	ab_chan		= make(chan uint16)
	db_chan		= make(chan byte)
	rw_chan		= make(chan bool)
	clk0_chan	= make(chan bool)
	so_chan		= make(chan bool)
	clk2_chan	= make(chan bool)
	res_chan		= make(chan bool)
)

var nodenames = map[uint16]string{
	rdy:		"rdy",
	clk1out:	"clk1",
	irq:		"irq",
	nmi:		"nmi",
	sync_:	"sync",
	ab0:		"ab0",
	ab1:		"ab1",
	ab2:		"ab2",
	ab3:		"ab3",
	ab4:		"ab4",
	ab5:		"ab5",
	ab6:		"ab6",
	ab7:		"ab7",
	ab8:		"ab8",
	ab9:		"ab9",
	ab10:	"ab10",
	ab11:	"ab11",
	ab12:	"ab12",
	ab13:	"ab13",
	ab14:	"ab14",
	ab15:	"ab15",
	db0:		"db0",
	db1:		"db1",
	db2:		"db2",
	db3:		"db3",
	db4:		"db4",
	db5:		"db5",
	db6:		"db6",
	db7:		"db7",
	rw:		"rw",
	clk0:		"clk0",
	so:		"so",
	clk2out:	"clk2",
	res:		"res",
}

// for debugging and monitor hook
type regs_monitor struct {
	A	byte
	X	byte
	Y	byte
	S	byte
	P	byte
}
var regs_chan = make(chan regs_monitor)

/************************************************************
 *
 * Libc Functions and Basic Data Types
 *
 ************************************************************/

const (
	// node states/values
	high = true
	low = false
	unknown = false
	// TODO(andlabs) - split states for pullup/pulldown to true/false?

	// transistor states
	on = true
	off = false
)

// TODO(andlabs) - maybe make some boolean expressions more explicit?

/************************************************************
 *
 * 6502 Description: Nodes, Transistors and Probes
 *
 ************************************************************/

/* the 6502 consists of this many nodes and transistors */
const (
	NODES = uint64(len(segdefs))
	TRANSISTORS = uint64(len(transdefs))
)

/************************************************************
 *
 * Bitmap Data Structures and Algorithms
 *
 ************************************************************/

type bitmap_t uint64
const (
	sizeof_bitmap_t = 8
	BITMAP_SHIFT = 6
	BITMAP_MASK = 63
)

func WORDS_FOR_BITS(a uint64) uint64 {
	return (a / (sizeof_bitmap_t * 8)) + 1
}

func DECLARE_BITMAP(count uint64) []bitmap_t {
	return make([]bitmap_t, WORDS_FOR_BITS(count))
}

func bitmap_clear(bitmap []bitmap_t, count uint64) {
	for i := uint64(0); i < WORDS_FOR_BITS(count); i++ {
		bitmap[i] = 0
	}
}

func set_bitmap(bitmap []bitmap_t, index uint64, state bool) {
	if state {
		bitmap[index >> BITMAP_SHIFT] |= (1 << (index & BITMAP_MASK))
	} else {
		bitmap[index >> BITMAP_SHIFT] &^= (1 << (index & BITMAP_MASK))
	}
}

func get_bitmap(bitmap []bitmap_t, index uint64) bool {
	return ((bitmap[index >> BITMAP_SHIFT] >> (index & BITMAP_MASK)) & 1) == 1
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
	nodes_gates			[NODES][NODES]uint64
	nodes_c1c2s			[NODES][2*NODES]uint64
	nodes_gatecount		[NODES]uint64
	nodes_c1c2count		[NODES]uint64
	nodes_nDependants		[NODES]uint64
	nodes_dependant		[NODES][NODES]uint64
)

/*
 * The "value" propertiy of VCC and GND is never evaluated in the code,
 * so we don't bother initializing it properly or special-casing writes.
 */

func set_nodes_pullup(node uint64, state bool) {
	set_bitmap(nodes_pullup, node, state)
}

func get_nodes_pullup(node uint64) bool {
	return get_bitmap(nodes_pullup, node)
}

func set_nodes_pulldown(node uint64, state bool) {
	set_bitmap(nodes_pulldown, node, state)
}

func get_nodes_pulldown(node uint64) bool {
	return get_bitmap(nodes_pulldown, node)
}

func set_nodes_value(node uint64, state bool) {
	set_bitmap(nodes_value, node, state)
}

func get_nodes_value(node uint64) bool {
	return get_bitmap(nodes_value, node)
}

/************************************************************
 *
 * Data Structures and Algorithms for Transistors
 *
 ************************************************************/

/* everything that describes a transistor */
var (
	transistors_gate	[TRANSISTORS]uint64
	transistors_c1		[TRANSISTORS]uint64
	transistors_c2		[TRANSISTORS]uint64
	transistors_on = DECLARE_BITMAP(TRANSISTORS)
)

//#ifdef BROKEN_TRANSISTORS
var broken_transistor = ^uint64(0)		// TODO const?
//#endif

func set_transistors_on(t uint64, state bool) {
//#ifdef BROKEN_TRANSISTORS
	if t == broken_transistor {
		return
	}
//#endif
	set_bitmap(transistors_on, t, state)
}

func transistor_state(t uint64) bool {
	return get_bitmap(transistors_on, t)
}

/************************************************************
 *
 * Data Structures and Algorithms for Lists
 *
 ************************************************************/

// TODO(andlabs) - can this whole thing be simplified to just slice logic?

/* list of nodes that need to be recalculated */
type list_t struct {
	list		[]uint64
	count	uint64
}

/* the nodes we are working with */
var (
	list1		[NODES]uint64
	listin = list_t{
		list:	 list1[:],
	}
)

/* the indirect nodes we are collecting for the next run */
var (
	list2		[NODES]uint64
	listout = list_t{
		list:	list2[:],
	}
)

func listin_get(i uint64) uint64 {
	return listin.list[i]
}

func listin_count() uint64 {
	return listin.count
}

func lists_switch() {
	listin, listout = listout, listin
}

func listout_clear() {
	listout.count = 0
}

func listout_add(node uint64) {
	listout.list[listout.count] = node
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
	group		[NODES]uint64
	groupcount	uint64
	groupbitmap = DECLARE_BITMAP(NODES)
)

// TODO(andlabs) - again, drop groupcount in favor of just len()? or will we wind up in a situation in the future where we have too many nodes...

func group_clear() {
	groupcount = 0
	bitmap_clear(groupbitmap, NODES)
}

func group_add(node uint64) {
	group[groupcount] = node
	groupcount++
	set_bitmap(groupbitmap, node, true)
}

func group_get(n uint64) uint64 {
	return group[n]
}

func group_contains(node uint64) bool {
	return get_bitmap(groupbitmap, node)
}

func group_count() uint64 {
	return groupcount
}

/************************************************************
 *
 * Node and Transistor Emulation
 *
 ************************************************************/

var (
	group_contains_pullup		bool
	group_contains_pulldown	bool
	group_contains_hi			bool
)

func addNodeToGroup(node uint64) {
	if group_contains(node) {
		return
	}

	group_add(node)

	// TODO change constant names?
	if get_nodes_pullup(node) == high {
		group_contains_pullup = true
	}
	if get_nodes_pulldown(node) == high {
		group_contains_pulldown = true
	}
	if get_nodes_value(node) == high {
		group_contains_hi = true
	}

	if node == vss || node == vcc {
		return
	}

	// revisit all transistors that are controlled by this node
	for t := uint64(0); t < nodes_c1c2count[node]; t++ {
		tn := nodes_c1c2s[node][t]
		// if the transistor connects c1 and c2...
		if transistor_state(tn) == on {
			// if original node was connected to c1, continue with c2
			if transistors_c1[tn] == node {
				addNodeToGroup(transistors_c2[tn])
			} else {
				addNodeToGroup(transistors_c1[tn])
			}
		}
	}
}

func addAllNodesToGroup(node uint64) {
	group_clear()

	group_contains_pullup = false
	group_contains_pulldown = false
	group_contains_hi = false

	addNodeToGroup(node)
}

func getGroupValue() bool {
	if group_contains(vss) {		// ground is always pulled low
		return low
	}

	if group_contains(vcc) {		// Vcc is always pulled high
		return high
	}

	if group_contains_pulldown {
		return low
	}

	if group_contains_pullup {
		return high
	}

	return group_contains_hi
}

func recalcNode(node uint64) {
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
	for i := uint64(0); i < group_count(); i++ {
		nn := group_get(i)
		if get_nodes_value(nn) != newv {
			set_nodes_value(nn, newv)
			for t := uint64(0); t < nodes_gatecount[nn]; t++ {
				tn := nodes_gates[nn][t]
				set_transistors_on(tn, !transistor_state(tn))
			}
			listout_add(nn)
		}
	}
}

func recalcNodeList(source []uint64, count uint64) {
	listout_clear()

	for i := uint64(0); i < count; i++ {
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
		for i := uint64(0); i < listin_count(); i++ {
			n := listin_get(i)
			for g := uint64(0); g < nodes_nDependants[n]; g++ {
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
	var temp [NODES]uint64
	for i := uint64(0); i < NODES; i++ {
		temp[i] = i
	}
	recalcNodeList(temp[:], NODES)
}

/************************************************************
 *
 * Node State
 *
 ************************************************************/

func setNode(nn uint64, state bool) {
	set_nodes_pullup(nn, state)
	set_nodes_pulldown(nn, !state)
	recalcNodeList([]uint64{ nn }, 1)
}

func isNodeHigh(nn uint64) bool {
	return get_nodes_value(nn)
}

/************************************************************
 *
 * Interfacing and Extracting State
 *
 ************************************************************/

func nhv(node uint64) byte {
	if isNodeHigh(node) {
		return 1
	}
	return 0
}

func read8(n0,n1,n2,n3,n4,n5,n6,n7 uint64) byte {
	return (byte(nhv(n0) << 0) |
		byte(nhv(n1) << 1) |
		byte(nhv(n2) << 2) |
		byte(nhv(n3) << 3) |
		byte(nhv(n4) << 4) |
		byte(nhv(n5) << 5) |
		byte(nhv(n6) << 6) |
		byte(nhv(n7) << 7))
}

func readAddressBus() uint16 {
	return uint16(read8(ab0,ab1,ab2,ab3,ab4,ab5,ab6,ab7)) |
		(uint16(read8(ab8,ab9,ab10,ab11,ab12,ab13,ab14,ab15)) << 8)
}

func readDataBus() byte {
	return read8(db0,db1,db2,db3,db4,db5,db6,db7)
}

var (
	dbnodes = [8]uint64{ db0, db1, db2, db3, db4, db5, db6, db7 }
)

func writeDataBus(d byte) {
	for i := 0; i < 8; i++ {
		setNode(dbnodes[i], (d & 1) == 1)
		d >>= 1
	}
}

func readRW() bool {
	return isNodeHigh(rw)
}

func readA() byte {
	return read8(a0,a1,a2,a3,a4,a5,a6,a7)
}

func readX() byte {
	return read8(x0,x1,x2,x3,x4,x5,x6,x7)
}

func readY() byte {
	return read8(y0,y1,y2,y3,y4,y5,y6,y7)
}

func readP() byte {
	return read8(p0,p1,p2,p3,p4,p5,p6,p7)
}

func readIR() byte {
	return read8(notir0,notir1,notir2,notir3,notir4,notir5,notir6,notir7) ^ 0xFF
}

func readSP() byte {
	return read8(s0,s1,s2,s3,s4,s5,s6,s7)
}

func readPCL() byte {
	return read8(pcl0,pcl1,pcl2,pcl3,pcl4,pcl5,pcl6,pcl7)
}

func readPCH() byte {
	return read8(pch0,pch1,pch2,pch3,pch4,pch5,pch6,pch7)
}

func readPC() uint16 {
	return (uint16(readPCH()) << 8) | uint16(readPCL())
}

/************************************************************
 *
 * Tracing/Debugging
 *
 ************************************************************/

var cycle uint

func chipStatus() {
	clk := isNodeHigh(clk0)
	a := readAddressBus()
	d := readDataBus()
	r_w := isNodeHigh(rw)

	fmt.Printf("halfcyc:%d phi0:%d AB:%04X D:%02X RnW:%v PC:%04X A:%02X X:%02X Y:%02X SP:%02X P:%02X IR:%02X",
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

	if clk {
		if r_w {
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

func mWrite(a uint16, d byte) {
	memory[a] = d
}

func handleMemory() {
	if isNodeHigh(rw) {
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

func chiploop() {
	for {
		// TODO(andlabs) - will this properly handle timing?
		select {
		// input pins
		case <-clk0_chan:		// TODO(andlabs) set clock state from channel input?
			clk := isNodeHigh(clk0)

			// invert clock
			setNode(clk0, !clk)

			cycle++
		case d := <-rdy_chan:
			setNode(rdy, d)
		case d := <-irq_chan:
			setNode(irq, d)
		case d := <-nmi_chan:
			setNode(nmi, d)
		case d := <-db_chan:
			writeDataBus(d)
		case d := <-so_chan:		// TODO does this properly handle so's odd behavior?
			setNode(so, d)
		case d := <-res_chan:
			setNode(res, d)
			if d == high {
				cycle = 0
			}

		// output pins
		// each of these do nothing other than the send
		case clk1_chan <- isNodeHigh(clk1out):
		case sync_chan <- isNodeHigh(sync_):
		case ab_chan <- readAddressBus():
		case db_chan <- readDataBus():
		case rw_chan <- isNodeHigh(rw):
		case clk2_chan <- isNodeHigh(clk2out):

		// debugging/monitor info
		case regs_chan <- regs_monitor{
			A:	readA(),
			X:	readX(),
			Y:	readY(),
			S:	readSP(),
			P:	readP(),
		}:
		}
	}
}

/************************************************************
 *
 * Initialization
 *
 ************************************************************/

var transistors uint		// TODO(andlabs) - make uint64?

func add_nodes_dependant(a uint64, b uint64) {
	for g := uint64(0); g < nodes_nDependants[a]; g++ {
		if nodes_dependant[a][g] == b {
			return
		}
	}

	nodes_dependant[a][nodes_nDependants[a]] = b
	nodes_nDependants[a]++
}

func setupNodesAndTransistors() {
	var i uint64

	// copy nodes into r/w data structure
	for i = 0; i < NODES; i++ {
		set_nodes_pullup(i, segdefs[i])
		nodes_gatecount[i] = 0
		nodes_c1c2count[i] = 0
	}

	// copy transistors into r/w data structure
	j := uint64(0)
	for i = 0; i < TRANSISTORS; i++ {
		gate := transdefs[i].gate
		c1 := transdefs[i].c1
		c2 := transdefs[i].c2
		/* skip duplicate transistors */
		found := false

		if !found {
			transistors_gate[j] = gate
			transistors_c1[j] = c1
			transistors_c2[j] = c2
			j++
		}
	}
	transistors = uint(j)
	if DEBUG {
		fmt.Printf("transistors: %d\n", transistors)
	}

	// cross reference transistors in nodes data structures
	for i = 0; i < uint64(transistors); i++ {
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
		nodes_nDependants[i] = 0
		for g := uint64(0); g < nodes_gatecount[i]; g++ {
			t := nodes_gates[i][g]
			add_nodes_dependant(i, transistors_c1[t])
			add_nodes_dependant(i, transistors_c2[t])
		}
	}
}

func dochip(chip_clock <-chan time.Time) {
	// set up data structures for efficient emulation
	setupNodesAndTransistors()

	// all nodes are down
	for nn := uint64(0); nn < NODES; nn++ {
		set_nodes_value(nn, low)
	}

	// all transistors are off
	for tn := uint64(0); tn < TRANSISTORS; tn++ {
		set_transistors_on(tn, off)
	}

	// set initial pin state
	setNode(res, low)
	setNode(clk0, high)
	setNode(rdy, high)
	setNode(so, low)
	setNode(irq, high)
	setNode(nmi, low)

	recalcAllNodes()

	// run the chip
	go chiploop()

	// hold RESET for 8 cycles
	for i := 0; i < 16; i++ {
		clk0_chan <- true
	}

	// release RESET
	res_chan <- high

	for _ = range chip_clock {		// apparently the syntax requires a variable for a range clause
		clk0_chan <- true
	}
}
