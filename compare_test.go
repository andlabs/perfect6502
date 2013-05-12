package main

import (
	"fmt"
	"testing"
)

func full_step(a *uint16, d *byte, r_w *bool) {
	step()
	step()

	*a = readAddressBus()
	*d = readDataBus()
	*r_w = isNodeHigh(rw)
}

const (
	RESET = 0xF000
	A_OUT = 0xF100
	X_OUT = 0xF101
	Y_OUT = 0xF102
	S_OUT = 0xF103
	P_OUT = 0xF104
)

const TRIGGER1 = 0x5555
var trigger2 uint16
const TRIGGER3 = 0xAAAA

func setup_memory(length int, b1 byte, b2 byte, b3 byte, A byte, X byte, Y byte, S byte, P byte) {
	for i := 0; i < 65536; i++ {
		memory[i] = 0
	}

	memory[0xFFFC] = RESET & 0xFF
	memory[0xFFFD] = RESET >> 8
	memory[RESET + 0x00] = 0xA2	// LDA #S
	memory[RESET + 0x01] = S
	memory[RESET + 0x02] = 0x9A	// TXS
	memory[RESET + 0x03] = 0xA9	// LDA #P
	memory[RESET + 0x04] = P
	memory[RESET + 0x05] = 0x48	// PHA
	memory[RESET + 0x06] = 0xA9	// LHA #A
	memory[RESET + 0x07] = A
	memory[RESET + 0x08] = 0xA2	// LDX #X
	memory[RESET + 0x09] = X
	memory[RESET + 0x0A] = 0xA0	// LDY #Y
	memory[RESET + 0x0B] = Y
	memory[RESET + 0x0C] = 0x28	// PLP
	memory[RESET + 0x0D] = 0x8D	// STA TRIGGER1
	memory[RESET + 0x0E] = TRIGGER1 & 0xFF
	memory[RESET + 0x0F] = TRIGGER1 >> 8
	memory[RESET + 0x10] = b1
	addr := uint16(RESET + 0x11)
	if length >= 2 {
		memory[addr] = b2
		addr++
	}
	if length >= 3 {
		memory[addr] = b3
		addr++
	}
	trigger2 = addr
	memory[addr] = 0x08; addr++		// PHP
	memory[addr] = 0x8D; addr++		// STA A_OUT
	memory[addr] = A_OUT & 0xFF; addr++
	memory[addr] = A_OUT >> 8; addr++
	memory[addr] = 0x8E; addr++		// STX X_OUT
	memory[addr] = X_OUT & 0xFF; addr++
	memory[addr] = X_OUT >> 8; addr++
	memory[addr] = 0x8C; addr++		// STY Y_OUT
	memory[addr] = Y_OUT & 0xFF; addr++
	memory[addr] = Y_OUT >> 8; addr++
	memory[addr] = 0x68; addr++		// PLA
	memory[addr] = 0x8D; addr++		// STA P_OUT
	memory[addr] = P_OUT & 0xFF; addr++
	memory[addr] = P_OUT >> 8; addr++
	memory[addr] = 0xBA; addr++		// TSX
	memory[addr] = 0x8E; addr++		// STX S_OUT
	memory[addr] = S_OUT & 0xFF; addr++
	memory[addr] = S_OUT >> 8; addr++
	memory[addr] = 0x8D; addr++		// STA TRIGGER3
	memory[addr] = TRIGGER3 & 0xFF; addr++
	memory[addr] = TRIGGER3 >> 8; addr++
	memory[addr] = 0xA9; addr++		// LDA #$00
	memory[addr] = 0x00; addr++
	memory[addr] = 0xF0; addr++		// BEQ .
	memory[addr] = 0xFE; addr++
}

func IS_READ_CYCLE() bool { return (isNodeHigh(clk0) && isNodeHigh(rw)) }
func IS_WRITE_CYCLE() bool { return (isNodeHigh(clk0) && !isNodeHigh(rw)) }
func IS_READING(a uint16) bool { return (IS_READ_CYCLE() && readAddressBus() == (a)) }

const MAX_CYCLES = 100

const (
	STATE_BEFORE_INSTRUCTION = iota
	STATE_DURING_INSTRUCTION
	STATE_FIRST_FETCH
)

func setup_perfect() {
	setupNodesAndTransistors()
//	verbose = 0
}

var (
	instr_ab	[10]uint16
	instr_db	[10]byte
	instr_rw	[10]bool
)

func perfect_measure_instruction() int {
	state := STATE_BEFORE_INSTRUCTION
	c := 0
	for i := 0; i < MAX_CYCLES; i++ {
		var ab uint16
		var db uint8
		var r_w bool

		full_step(&ab, &db, &r_w)

		if state == STATE_DURING_INSTRUCTION && ab > trigger2 {
			/*
			 * we see the FIRST fetch of the next instruction,
			 * the test instruction MIGHT be done
			 */
			state = STATE_FIRST_FETCH
		}

		if state == STATE_DURING_INSTRUCTION {
			instr_rw[c] = r_w
			instr_ab[c] = ab
			instr_db[c] = db
			c++
		}

		if ab == TRIGGER1 {
			state = STATE_DURING_INSTRUCTION	// we're done writing the trigger value; now comes the instruction!
		}
		if ab == TRIGGER3 {
			break							// we're done dumping the CPU state
		}
	}

	return c
}

func TestCompare(t *testing.T) {
	setup_perfect()
//	setup_memory(1, 0xEA, 0x00, 0x00, 0, 0, 0, 0, 0);
//	setup_memory(2, 0xA9, 0x00, 0x00, 0, 0, 0, 0, 0);
//	setup_memory(2, 0xAD, 0x00, 0x10, 0, 0, 0, 0, 0);
//	setup_memory(3, 0xFE, 0x00, 0x10, 0, 0, 0, 0, 0);
//	setup_memory(3, 0x9D, 0xFF, 0x10, 0, 2, 0, 0, 0);
	setup_memory(1, 0x28, 0x00, 0x00, 0x55, 0, 0, 0x80, 0)
	resetChip()
	instr_cycles := perfect_measure_instruction()

	for c := 0; c < instr_cycles; c++ {
		fmt.Printf("T%d ", c + 1)
		if instr_rw[c] {
			fmt.Printf("R $%04X\n", instr_ab[c])
		} else {
			fmt.Printf("W $%04X = $%02X\n", instr_ab[c], instr_db[c])
		}
	}

	setup_emu()
	setup_memory(1, 0x48, 0x00, 0x00, 0x55, 0, 0, 0x80, 0)
	reset_emu()
	instr_cycles2 := emu_measure_instruction()
	_ = instr_cycles2
}
