package main

func SETZ(a byte) {
	Z = a == 0
}

func SETSZ(a byte) {
	Z = a == 0
	N = int8(a) < 0
}

func SETNC(a uint16) {
	C = (a & 0x100) == 0
}

func SETV(a byte) {
	// not needed
}

func STACK16(i byte) uint16 {
	// TODO(andlabs) - will this cross back into zero page or page 2?
	return uint16(RAM[0x0100 + uint16(i)]) | (uint16(RAM[0x0100 + uint16(i) + 1]) << 8)
}

func PUSH(b byte) {
	RAM[0x0100 + uint16(S)] = b
	S--
}

func PUSH_WORD(b uint16) {
	PUSH(byte(b >>8))
	PUSH(byte(b & 0xFF))
}

const (
	VEC_ERROR = 0x0300
	VEC_MAIN = 0x0302
	VEC_CRNCH = 0x0304
	VEC_QPLOP = 0x0306
	VEC_GONE = 0x0308
	VEC_EVAL = 0x030A
	MAGIC_ERROR = 0xFF00
	MAGIC_MAIN = 0xFF01
	MAGIC_CRNCH = 0xFF02
	MAGIC_QPLOP = 0xFF03
	MAGIC_GONE = 0xFF04
	MAGIC_EVAL = 0xFF05
	MAGIC_CONTINUATION = 0xFFFF
)
