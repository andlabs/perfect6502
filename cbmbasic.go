package main

//import (
//	"fmt"
//)

func main() {
	clk := false

	initAndResetChip()

	// set up memory for user program
	init_monitor()

	// emulate the 6502!
	for {
		step()
		clk = !clk
		if clk {
			handle_monitor()
		}

//		chipStatus()
		//if (cycle % 1000) == 0 { fmt.Printf("%d\n", cycle) }
	}
}
