perfect6502 ported to Go
===========
Pietro Gagliardi (andlabs)<br>2013

I intend to use this to create a general-purpose perfect emulator core. Things planned:
- unifying all the types in perfect6502.go to just use uint64 (there is a lot of confusion and type mismatches in the original anyway)
- changing all external pins to use channels so I can hook things together
	- this means the simulator will run in a goroutine and the clock will be automated

The fake kernal doesn't quite line up properly with Go so there are still problems (for instance, there's that issue of plugins and argc/argv...? and read errors that are not EOF are not handled at all (in fact, the fake kernal/host OS interface functions have not been tested by me *at all*; you have been warned)

Also host-OS-dependent things (console access, SETTIM, system()) are not implemented (yet?).

or just search for `TODO(andlabs)`

also TODO - clean up the inconsistent licensing information inclusion
