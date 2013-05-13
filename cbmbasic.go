package main

//import (
//	"fmt"
//)

func main() {
	monitor_hook = handle_monitor

	initAndResetChip()

	// set up memory for user program
	init_monitor()

	// emulate the 6502!
	for {
		step()

//		chipStatus()
		//if (cycle % 1000) == 0 { fmt.Printf("%d\n", cycle) }
	}
}
