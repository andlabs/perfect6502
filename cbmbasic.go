package main

import (
	"time"
)

const C64ClockHz = 1022727		// NTSC, from http://codebase64.org/doku.php?id=base:cpu_clocking
const C64Clock = time.Second / (C64ClockHz * 2)		// thanks David Wendt

func main() {
	// set up memory for user program
	init_monitor()
	go monitor()

	// set up 6502 environment
	clock_chan = time.Tick(C64Clock)

	// emulate the 6502!
	go dochip()

	select {}		// wait forever
}
