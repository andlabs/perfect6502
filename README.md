perfect6502 ported to Go
===========
Pietro Gagliardi (andlabs)<br>2013

I intend to use this to create a general-purpose perfect emulator core. Things planned:
- figure out the proper terminology: are node values high/low? pullups? pulldowns?
	- or should I rename everything to isNodeHigh/isPullup/isPulldown... or would that still have an edge case
		- carriesCurrent? (for value)
- changing all external pins to use channels so I can hook things together
	- this means the simulator will run in a goroutine and the clock will be automated

The fake kernal doesn't quite line up properly with Go so there are still problems (for instance, there's that issue of plugins and argc/argv...? and read errors that are not EOF are not handled at all (in fact, the fake kernal/host OS interface functions have not been tested by me **at all**; you have been warned)

Also host-OS-dependent things (console access, SETTIM, system()) are not implemented (yet?).

or just search for `TODO(andlabs)`

also TODO
- clean up the inconsistent licensing information inclusion
- make the comparative tester useful by seeing which chip's output format to use for the other
- replace character output printfs with direct writes to see if it makes printing characters to the console go any faster
- right now, the monitor is running on the same clock as the CPU, and it is assumed they are in sync; a better option is to use clk2, but I would need to rework everything to generate pin signals in the correct order and at the correct time, otherwise clk2 goes out of sync




6502 pinout, noting I/O direction

```
 vss | ?      I | res
 rdy | I      O | clk2
clk1 | O      I | so
 irq | I      I | clk0
 n.c | ?      ? | n.c
 nmi | I      ? | n.c
sync | O     O? | rw
 vcc | ?    I/O | db0
 ab0 | O    I/O | db1
 ab1 | O    I/O | db2
 ab2 | O    I/O | db3
 ab3 | O    I/O | db4
 ab4 | O    I/O | db5
 ab5 | O    I/O | db6
 ab6 | O    I/O | db7
 ab7 | O      O | ab15
 ab8 | O      O | ab14
 ab9 | O      O | ab13
ab10 | O      O | ab12
ab11 | O      ? | vss
```

Info taken from the May 1976 datasheet.

My guess that RW is output is based on both the block diagram and from http://members.casema.nl/hhaydn/howel/parts/6502_CPU.htm; the pinout section of the May 1976 datasheet does not say...
