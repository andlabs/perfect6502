package main

import (
	"os"
)

/* XXX hook up memory[] with RAM[] in runtime.c */
 
/************************************************************
 *
 * Interface to OS Library Code / Monitor
 *
 ************************************************************/

/* imported by runtime.c */
var (
	A, X, Y, S, P	byte
	PC			uint16
	N, Z, C		bool
)

func init_monitor() {
	f, err := os.Open("cbmbasic.bin")
	if err != nil {
		fatalf("open cbmbasic.bin failed: %v", err)
	}
	_, err = f.Read(memory[0xA000:0xA000 + 17591])
	if err != nil {
		fatalf("error reading cbmbasic.bin: %v", err)
	}
	f.Close()

	/*
	 * fill the KERNAL jumptable with JMP $F800;
	 * we will put code there later that loads
	 * the CPU state and returns
	 */
	for addr := 0xFF90; addr < 0xFFF3; addr += 3 {
		memory[addr+0] = 0x4C
		memory[addr+1] = 0x00
		memory[addr+2] = 0xF8
	}

	/*
	 * cbmbasic scribbles over 0x01FE/0x1FF, so we can't start
	 * with a stackpointer of 0 (which seems to be the state
	 * after a RESET), so RESET jumps to 0xF000, which contains
	 * a JSR to the actual start of cbmbasic
	 */
	memory[0xF000] = 0x20
	memory[0xF001] = 0x94
	memory[0xF002] = 0xE3
	
	memory[0xFFFC] = 0x00
	memory[0xFFFD] = 0xF0
}

func monitor() {
	call_kernal := false

	for {
		// wait for a REF "B"
		for <-clk1_chan != low {
			// do nothing
		}

		// REF "B"; handle memory accesses
		addr := <-ab_chan
		rw := <-rw_chan
		if rw == high {		// read
			if <-sync_chan == high {		// instruction fetch
				PC = addr

				if PC >= 0xFF90 && ((PC - 0xFF90) % 3 == 0) {
					call_kernal = true
				}
			}
		}

		// wait for a REF "A"
//		for <-clk1_chan != high {
//			// do nothing
//		}

		// REF "A"; call the kernal and commit reads/writes
		// TODO which order?
		if call_kernal {
			// get register status out of 6502
			regs := <-regs_chan
			A = regs.A
			X = regs.X
			Y = regs.Y
			S = regs.S
			P = regs.P
			N = (P >> 7) == 1
			Z = ((P >> 1) & 1) == 1
			C = (P & 1) == 1

			kernal_dispatch()

			// encode processor status
			P &= 0x7C				// clear N, Z, C
			if N {
				P |= 1 << 7
			}
			if Z {
				P |= 1 << 1
			}
			if C {
				P |= 1
			}

			/*
			 * all KERNAL calls make the 6502 jump to $F800, so we
			 * put code there that loads the return state of the
			 * KERNAL function and returns to the caller
			 */
			memory[0xF800] = 0xA9		// LDA #P
			memory[0xF801] = P
			memory[0xF802] = 0x48		// PHA
			memory[0xF803] = 0xA9		// LHA #A
			memory[0xF804] = A
			memory[0xF805] = 0xA2		// LDX #X
			memory[0xF806] = X
			memory[0xF807] = 0xA0		// LDY #Y
			memory[0xF808] = Y
			memory[0xF809] = 0x28		// PLP
			memory[0xF80A] = 0x60		// RTS
			/*
			 * XXX we could do RTI instead of PLP/RTS, but RTI seems to be
			 * XXX broken in the chip dump - after the KERNAL call at 0xFF90,
			 * XXX the 6502 gets heavily confused about its program counter
			 * XXX and executes garbage instructions
			 */

			call_kernal = false
		}

		if rw == high {		// read
			// send data
			rdy_chan <- high
			db_chan <- memory[addr]
		} else {			// write
			memory[addr] = <-db_chan
		}
	}
}
