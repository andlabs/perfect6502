package main

import (
	"fmt"
	"os"
)

type step_t byte

var (
	t		step_t	// step inside the instruction */
	ir		byte		// instruction register
//static uint8_t operand;
	emuPC	uint16
	emuA	byte
	emuX	byte
	emuY	byte
	emuS	byte
	emuP	byte
	temp_lo	byte
	temp_hi	byte
)

func TEMP16() uint16 {
	return uint16(temp_lo) | (uint16(temp_hi) << 8)
}

var (
	AB	uint16
	DB	uint8
	RW	int
)

const (
	RW_WRITE = 0
	RW_READ = 1
)

func emuLOAD(a uint16) byte {
	return memory[a]
}

const (
	T1 = (1<<0)
	T2 = (1<<1)
	T3 = (1<<2)
	T4 = (1<<3)
	T5 = (1<<4)
	T6 = (1<<5)
	T7 = (1<<6)
)

func IS_T1() bool { return (t & T1) != 0 }
func IS_T2() bool { return (t & T2) != 0 }
func IS_T3() bool { return (t & T3) != 0 }
func IS_T4() bool { return (t & T4) != 0 }
func IS_T5() bool { return (t & T5) != 0 }
func IS_T6() bool { return (t & T6) != 0 }
func IS_T7() bool { return (t & T7) != 0 }

func IFETCH() {
	AB = emuPC
	RW = RW_READ
	t = T1
}

func emu_init() {
	t = T1
	ir = 0
	emuPC = 0
	emuA = 0
	emuX = 0
	emuY = 0
	emuS = 0
	emuP = 0
	IFETCH()
}

func EOI_INCPC_READPC() {
	emuPC++
	t <<= 1
	AB = emuPC
	RW = RW_READ
}

func DB_TO_ADDRLO() {
	temp_lo = DB
}

func DB_TO_ADDRHI() {
	temp_hi = DB
}

func EOI() {
	t <<= 1
}

func EOI_INCPC() {
	emuPC++
	EOI()
}

func EOI_INCPC_READADDR() {
	EOI_INCPC()
	AB = TEMP16()
	RW = RW_READ
}

func EOI_INCPC_WRITEADDR() {
	EOI_INCPC()
	AB = TEMP16()
	RW = RW_WRITE
}

func pha() {
//	printf("%s",__func__);
	if IS_T2() {
		emuS--
		EOI()
	} else if IS_T3() {
		AB = 0x0100 + uint16(emuS)
		DB = emuA
		RW = RW_WRITE
		emuS++
		EOI()
	} else if IS_T4() {
		IFETCH()
	}
}

func plp() {
//	printf("%s",__func__);
	if IS_T2() {
		EOI()
	} else if IS_T3() {
		temp_lo = emuS
		temp_hi = 0x01
		AB = TEMP16()
		RW = RW_READ
		emuS++
		EOI()
	} else if IS_T4() {
		temp_lo = emuS
		AB = TEMP16()
		RW = RW_READ
		EOI()
	} else if IS_T5() {
		emuP = DB
		IFETCH()
	}
}

func txs() {
//	printf("%s",__func__);
	/* T2 */
	emuS = emuX
	IFETCH()
}

func lda_imm() {
//	printf("%s",__func__);
	/* T2 */
	emuA = DB
	emuPC++
	IFETCH()
}

func ldx_imm() {
//	printf("%s",__func__);
	/* T2 */
	emuX = DB
	emuPC++
	IFETCH()
}

func ldy_imm() {
//	printf("%s",__func__);
	/* T2 */
	emuY = DB
	emuPC++
	IFETCH()
}

func lda_abs() {
//	printf("%s",__func__);
	if IS_T2() {
		DB_TO_ADDRLO()
		EOI_INCPC_READPC()
	} else if IS_T3() {
		DB_TO_ADDRHI()
		EOI_INCPC_READADDR()
	} else if IS_T4() {
		emuA = DB
		IFETCH()
	}
}

func sta_abs() {
//	printf("%s",__func__);
	if IS_T2() {
		DB_TO_ADDRLO()
		EOI_INCPC_READPC()
	} else if IS_T3() {
		DB_TO_ADDRHI()
		DB = emuA
		EOI_INCPC_WRITEADDR()
	} else if IS_T4() {
		IFETCH()
	}
}

var emucycle int = 0

func emulate_step() {
	// memory
	if RW == RW_READ {
		fmt.Printf("PEEK(%04X)=%02X ", AB, memory[AB])
		DB = memory[AB]
	} else {
		fmt.Printf("POKE %04X, %02X ", AB, DB)
		memory[AB] = DB
	}

	//printf("T%d PC=%04X ", t, emuPC);
	if IS_T1() {	// T0: get result of IFETCH
		fmt.Printf("fetch")
		ir = DB
		EOI_INCPC_READPC()
	} else {
		//printf ("IR: %02X ", ir);
		switch ir {
		case 0x28:
			plp()
		case 0x48:
			pha()
		case 0x8D:
			sta_abs()
		case 0x9A:
			txs()
		case 0xA0:
			ldy_imm()
		case 0xA2:
			ldx_imm()
		case 0xA9:
			lda_imm()
		case 0xAD:
			lda_abs()
		default:
			fmt.Printf("unimplemented opcode: %02X\n", ir)
			os.Exit(0)
		}
	}

	fmt.Printf("\ncycle:%d phi0:1 AB:%04X D:%02X RnW:%d PC:%04X A:%02X X:%02X Y:%02X SP:%02X P:%02X IR:%02X",
			emucycle,
			AB,
	        DB,
	        RW,
			emuPC,
			emuA,
			emuX,
			emuY,
			emuS,
			emuP,
			ir)

}

func setup_emu() {
	emu_init()
}

func reset_emu() {
	emu_init()
	emuPC = uint16(memory[0xFFFC]) | (uint16(memory[0xFFFD]) << 8)
	fmt.Printf("PC %x\n", emuPC)
	IFETCH()
}

func emu_measure_instruction() int {
	for {
		fmt.Printf("cycle %d: ", emucycle)
		emulate_step()
		fmt.Printf("\n")
		emucycle++
	}
	return 0
}
