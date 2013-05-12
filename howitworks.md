<!-- 11 may 2013 -->
My attempt at understanding and explaining how perfect6502 works
======================
Pietro Gagliardi

TODO
- explain what the effect of all this is

## What's in a netlist?
A netlist contains two data structures:
- **segdefs** - a description of nodes
- **transdefs** - a description of transistors

Nodes connect transistors. They have a *value*, which is high when there is current flowing through and low otherwise, and a *pull-up* and *pull-down* states, which control one (two?) "resistor(s?)" that determine what happens to transistors connected to the node.

As in NMOS logic, transistors are field-effect transistors, and thus have gate, source, and drain terminals. However, for visual6502's needs, the difference between source and drain is unimportant and thus not considered.

### segdefs
```
type segdefs []nodeState
```
segdefs is just a linear list of nodes and their default pull-up states.

There are a handful of nodes that are not in the visual6502 segdefs; these are marked with a 2 in perfect6502. If you search for them in visual6502 you will get no results, and the original perfect6502 code will handle a 2 as off.

### transdefs
```
type transdefs []struct {
    gate    nodenum
    c1      nodenum
    c2      nodenum
}
```
transdefs defines a list of transistors by their terminals. Each value refers to an index in segdefs. I will call c1 and c2 *connectors* (not sure if this is the most accurate word).

http://visual6502.org/wiki/index.php?title=The_ChipSim_Simulator and http://forum.6502.org/viewtopic.php?f=1&t=2420 indicate that c1 is source and c2 is drain.

### Special nodes
```
var (
    clk0    nodenum
    vss     nodenum        // aka GND
    vcc     nodenum
)
```
These three special node names are required for proper simulation.

### So what information do we need to store?
We need to store the following information:

- For every node n, we need to store:
	- the pull-up state
	- the pull-down state
	- the value
	- the transistors t for which t.gate == n
		- t.c1 and t.c2, which we call the *dependants* of n
	- the transistors t for which t.c1 == n
	- the transistors t for which t.c2 == n
- For every transistor, we need to store:
	- the value

## So what happens on initialization?
On initialization,

1. For each node in segdefs,
	1. Set the pull-up state to the state in segdefs
	2. Clear the gate and connector count
2. For each transistor in transdefs,
	1. Add the transistor as a gate of its gate node
	2. Add the transistor as the appropriate connectors of its connector nodes
3. For each node in segdefs, determine the dependents

TODO does this handle pull-down?

## So what happens on power on?
On power on,

1. Set all node values to low
2. Turn off all transistors
3. Set any node states necessary on system startup; for the 6502:
	1. Set res and so to low (pull-down) - this holds the reset line
	2. Set clk0, rdy, irq, and nmi to high (pull-up)
4. Recalculate all nodes
5. Do any other initialization; for the 6502:
	1. Wait 8 cycles
	2. Set res to high (pull-up) - this releases the reset line

## So what happens on each clock edge change?
The main loop of the emulator is simple:

1. Flip the clock node and update the chain of connected nodes and transistors
2. If this is a FALLING edge, handle memory accesses, which will touch the other pins, whose chains are updated likewise

Therefore, the entry point to the chip simulation process is the routine that sets a node.

### Setting a node
When a node is set, three things happen:

1. The pull-up state is set to state
2. The pull-down state is set to the inverse of the state
3. The node's associated node list is recalculated; this is done by recalculating a list consisting of just the node

Steps 1 and 2 ensure that a node is never both pull-up and pull-down at the same time (TODO does it also ensure neither? on startup all nodes are set down IIRC...).

### Recalculating a list of nodes
There are two global lists, the input list and the output list. When we recalculate a list of one or more nodes,

1. Clear the output list
2. Recalculate each individual node in succession; this populates the output list
3. Set the output list to the input list
4. While there are still nodes in the input list,
	1. Clear the output list
	2. Recalculate every individual dependant of every node of the input list
	3. Set the output list to the input list

perfect6502 has a limiter so that step 4 may run for at most 100 iterations; I'm not sure if this is the right thing to do for other chips.

### Recalculating an individual node
When recalculating an individual node,

1. Compute the list of nodes connected to the current node; this will be called the *group*
2. Get the value of the group
3. For every node in the group, if the value of that node is not the same as the value of the group,
	1. Set the value of the node to the value of the group
	2. For every transistor whose gate is this node, flip the transistor
	3. Add the node to the output list

### Adding a node to a group
The steps to add a node to a group is:

1. Make sure the node is not already in the group; if it is, stop
2. Actually store the node in the data structure
3. If the node is NEITHER ground NOR Vcc, for every transistor for which this node is either connector, if the transistor is on, add the other connected node to the group

### Computing the group's value
Computing the value of a group is easy:

1. If the group contains ground (Vss), the value is low
2. If the group contains Vcc, the value is high
3. If the group contains a pull-down node, the value is low
4. If the group contains a pull-up node, the value is high
5. Otherwise, the group value is high if it contains a node whose value is high, and low otherwise

These "if the group contains" can be computed during the node addition process.

## Thanks
- GerbilSoft, nensondubois: terminology clear-up, confirming things I suspected
- http://en.wikipedia.org/wiki/Node_%28circuits%29 and http://uvicrec.blogspot.com/2011/07/simple-experiment-with-jssim-visual6502.html for explaining nodes and their pullup/pulldown property (in the latter, the paragraph starting with "Which probably looks pretty cryptic")
- http://uvicrec.blogspot.com/2011/07/simple-experiment-with-jssim-visual6502.html also confirms that the difference between source and drain is unimportant; I'd like to know why...
