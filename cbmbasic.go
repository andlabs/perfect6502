package main

import (
	"time"
)

const C64ClockHz = 1022727		// NTSC, from http://codebase64.org/doku.php?id=base:cpu_clocking
const C64Clock = time.Second / (C64ClockHz * 2)		// thanks David Wendt

func splitClock(c <-chan time.Time) (c1 <-chan time.Time, c2 <-chan time.Time) {
	dc1 := make(chan time.Time)
	dc2 := make(chan time.Time)
	go func() {
		for x := range c {
			dc1 <- x
			dc2 <- x
		}
	}()
	return dc1, dc2
}

func main() {
	// set up 6502 environment
	chip_clock := time.Tick(C64Clock)

	// emulate the 6502!
	go dochip(chip_clock)

	// set up memory for user program
	init_monitor()
	go monitor()

	// consume clk2 so as to not deadlock
	for _ = range clk2_chan {
		// do nothing
	}
}
