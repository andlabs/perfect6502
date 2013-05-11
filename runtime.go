/*
 * Copyright (c) 2009 Michael Steil, James Abbatiello
 * Copyright (c) 2013 Pietro Gagliardi
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 */

package main

import (
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"math/rand"
	"time"
	// ...
)

//#define NO_CLRHOME

// the main program must set this to true if it had #define DEBUG before
var DEBUG bool = false

var (
	RAM		[65536]byte
)

func stack4(a uint16, b uint16, c uint16, d uint16) bool {
//	printf("stack4: %x,%x,%x,%x\n", a, b, c, d);
	if STACK16(S+1) + 1 != a {
		return false
	}
	if STACK16(S+3) + 1 != b {
		return false
	}
	if STACK16(S+5) + 1 != c {
		return false
	}
	if STACK16(S+7) + 1 != d {
		return false
	}
	return true
}

/*
 * CHRGET/CHRGOT
 * CBMBASIC implements CHRGET/CHRGOT as self-modifying
 * code in the zero page. This cannot be done with
 * static recompilation, so here is a reimplementation
 * of these functions in C.
0073   E6 7A      INC $7A
0075   D0 02      BNE $0079
0077   E6 7B      INC $7B
0079   AD XX XX   LDA $XXXX
007C   C9 3A      CMP #$3A   ; colon
007E   B0 0A      BCS $008A
0080   C9 20      CMP #$20   ; space
0082   F0 EF      BEQ $0073
0084   38         SEC
0085   E9 30      SBC #$30   ; 0
0087   38         SEC
0088   E9 D0      SBC #$D0
008A   60         RTS
*/
// TODO(andlabs) - was static; this makes it exported (worry?)
func CHRGET_common(inc bool) {
	var temp16 uint16

	if !inc {
		goto CHRGOT_start
	}
CHRGET_start:
	RAM[0x7A]++
	SETSZ(RAM[0x7A])
	if !Z {
		goto CHRGOT_start
	}
	RAM[0x7B]++
	SETSZ(RAM[0x7B])
CHRGOT_start:
	A = RAM[uint16(RAM[0x7A]) | (uint16(RAM[0x7B]) << 8)]
	SETSZ(A)
	temp16 = uint16(A) - 0x3A
	SETNC(temp16)
	SETSZ(byte(temp16 & 0xFF))
	if C {
		return
	}
	temp16 = uint16(A) - 0x20
	SETNC(temp16)
	SETSZ(byte(temp16 & 0xFF))
	if Z {
		goto CHRGET_start
	}
	C = true
	temp16 = uint16(A) - 0x30 - 0//(1 - uint16(C))
//TODO(andlabs)
//	SETV(byte((uint16(A) ^ temp16) & 0x80) && ((A ^ 0x30) & 0x80))
	A = byte(temp16 & 0xFF)
	SETSZ(A)
	SETNC(temp16)
	C = true
	temp16 = uint16(A) - 0xD0 - 0//(1 - uint16(C))
//TODO(andlabs)
//	SETV(byte((uint16(A) ^ temp16) & 0x80) && ((A ^ 0xD0) & 0x80))
	A = byte(temp16 & 0xFF)
	SETSZ(A)
	SETNC(temp16)
}

func CHRGET() {
	CHRGET_common(true)
}

func CHRGOT() {
	CHRGET_common(false)
}


/************************************************************/
/* KERNAL interface implementation                          */
/* http://members.tripod.com/~Frank_Kontros/kernal/addr.htm */
/************************************************************/

/* KERNAL constants */
// #if 0
// #define RAM_BOT 0x0400 /* we could just as well start at 0x0400, as there is no screen RAM */
// #else
const RAM_BOT = 0x0800
// #endif
const (
	RAM_TOP = 0xA000
	KERN_ERR_NONE = 0
	KERN_ERR_FILE_OPEN = 2
	KERN_ERR_FILE_NOT_OPEN = 3
	KERN_ERR_FILE_NOT_FOUND = 4
	KERN_ERR_DEVICE_NOT_PRESENT = 5
	KERN_ERR_NOT_INPUT_FILE = 6
	KERN_ERR_NOT_OUTPUT_FILE = 7
	KERN_ERR_MISSING_FILE_NAME = 8
	KERN_ERR_ILLEGAL_DEVICE_NUMBER = 9

	KERN_ST_TIME_OUT_READ = 0x02
	KERN_ST_EOF = 0x40
)

/* KERNAL internal state */
var (
	kernal_msgflag		byte
	kernal_status			byte = 0
	kernal_filename		uint16
	kernal_filename_len		uint16		// originally byte but changed to uint16 to avoid casting everywhere
	kernal_lfn				byte
	kernal_dev			byte
	kernal_sec			byte
	kernal_quote			int = 0
	kernal_output			byte = 0
	kernal_input			byte = 0
	kernal_files = []*os.File{ nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil }
	kernal_files_next = []int{ 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0 }
)

const EOF = 0xFFFF

/* shell script hack */
var (
	readycount	int = 0
	interactive	bool
	input_file		*os.File
)

func init_os(args []string) uint16 {
	var err error

//	printf("init_os %d\n", argc);
	if len(args) == 0 {	// continuation
		return PC
	}

	if len(args) > 1 {
		interactive = false
		input_file, err = os.Open(args[1])
		if err != nil {
			fatalf("error opening %s: %v", args[1], err)
		}

		getc := func() byte {
			c := make([]byte, 1)
			_, err = input_file.Read(c)
			if err != nil {
				fatalf("error reading %s: %v", args[1], err)
			}
			return c[0]
		}

		if getc() =='#' {
			c := getc()
			for (c != 13) && (c!=10) {
				c = getc()
			}
		} else {
			_, err = input_file.Seek(0, 0)
			if err != nil {
				fatalf("error seeking %s back to start: %v", args[1], err)
			}
		}
	} else {
		interactive = true
		input_file = nil
	}
	rand.Seed(time.Now().Unix())

	return 0xE394		// main entry point of BASIC
}

var (
	orig_error	uint16
	orig_main	uint16
	orig_crnch	uint16
	orig_qplop	uint16
	orig_gone	uint16
	orig_eval		uint16
)

var plugin bool = false

func replace_vector(address uint16, new uint16, old *uint16) {
	*old = uint16(RAM[address]) | (uint16(RAM[address+1]) << 8)
	RAM[address] = byte(new & 0xFF)
	RAM[address+1] = byte(new >> 8)
}

func plugin_on() {
	if plugin {
		return
	}

	replace_vector(VEC_ERROR, MAGIC_ERROR, &orig_error)
	replace_vector(VEC_MAIN, MAGIC_MAIN, &orig_main)
	replace_vector(VEC_CRNCH, MAGIC_CRNCH, &orig_crnch)
	replace_vector(VEC_QPLOP, MAGIC_QPLOP, &orig_qplop)
	replace_vector(VEC_GONE, MAGIC_GONE, &orig_gone)
	replace_vector(VEC_EVAL, MAGIC_EVAL, &orig_eval)
	
	plugin = true
}

func plugin_off() {
	var dummy uint16

	if !plugin {
		return
	}

	replace_vector(VEC_ERROR, orig_error, &dummy)
	replace_vector(VEC_MAIN, orig_main, &dummy)
	replace_vector(VEC_CRNCH, orig_crnch, &dummy)
	replace_vector(VEC_QPLOP, orig_qplop, &dummy)
	replace_vector(VEC_GONE, orig_gone, &dummy)
	replace_vector(VEC_EVAL, orig_eval, &dummy)

	plugin = false
}

// TODO(andlabs) - was static; this makes it exported (worry?)
func SETMSG() {
		kernal_msgflag = A
		A = kernal_status
}

// TODO(andlabs) - was static; this makes it exported (worry?)
func MEMTOP() {
	if DEBUG {		// CBMBASIC doesn't do this
		if !C {
			fatalf("UNIMPL: set top of RAM")
		}
	}
	X = byte(RAM_TOP & 0xFF)
	Y = byte(RAM_TOP >> 8)

	/*
	 * if we want to turn on the plugin
	 * automatically at start, we can do it here.
	 */
	//plugin_on();
}

/* MEMBOT */
// TODO(andlabs) - was static; this makes it exported (worry?)
func MEMBOT() {
	if DEBUG {		// CBMBASIC doesn't do this
		if !C {
			fatalf("UNIMPL: set bot of RAM")
		}
	}
	X = byte(RAM_BOT & 0xFF)
	Y = byte(RAM_BOT >> 8)
}

/* READST */
// TODO(andlabs) - was static; this makes it exported (worry?)
func READST() {
	A = kernal_status
}

/* SETLFS */
// TODO(andlabs) - was static; this makes it exported (worry?)
func SETLFS() {
	kernal_lfn = A
	kernal_dev = X
	kernal_sec = Y
}

/* SETNAM */
// TODO(andlabs) - was static; this makes it exported (worry?)
func SETNAM() {
	kernal_filename = uint16(X) | (uint16(Y) << 8)
	kernal_filename_len = uint16(A)
}

/* OPEN */
// TODO(andlabs) - was static; this makes it exported (worry?)
func OPEN() {
	kernal_status = 0
	if kernal_files[kernal_lfn] != nil {
		C = true
		A = KERN_ERR_FILE_OPEN
	} else if kernal_filename_len == 0 {
		C = true
		A = KERN_ERR_MISSING_FILE_NAME
	} else {
		var f *os.File
		var err error

		filename := string(RAM[kernal_filename:kernal_filename + kernal_filename_len])
		if kernal_sec == 0 {
			f, err = os.Open(filename)
		} else {
			f, err = os.Create(filename)
		}
		// TODO(andlabs) - report error on stderr?
		if err != nil {
			f = nil
		}
		kernal_files[kernal_lfn] = f
		if kernal_files[kernal_lfn] != nil {
			kernal_files_next[kernal_lfn] = EOF
			C = false
		} else {
			C = true
			A = KERN_ERR_FILE_NOT_FOUND
		}
	}
}

/* CLOSE */
// TODO(andlabs) - was static; this makes it exported (worry?)
func CLOSE() {
	if kernal_files[kernal_lfn] == nil {
		C = true
		A = KERN_ERR_FILE_NOT_OPEN
	} else {
		kernal_files[kernal_lfn].Close()
		kernal_files[kernal_lfn] = nil
		C = false
	}
}

/* CHKIN */
// TODO(andlabs) - was static; this makes it exported (worry?)
func CHKIN() {
	kernal_status = 0
	if kernal_files[X] == nil {
		C = true
		A = KERN_ERR_FILE_NOT_OPEN
	} else {
		// TODO Check read/write mode
		kernal_input = X
		C = false
	}
}

/* CHKOUT */
// TODO(andlabs) - was static; this makes it exported (worry?)
func CHKOUT() {
	kernal_status = 0
	if kernal_files[X] == nil {
		C = true
		A = KERN_ERR_FILE_NOT_OPEN
	} else {
		// TODO Check read/write mode
		kernal_output = X
		C = false
	}
}

/* CLRCHN */
// TODO(andlabs) - was static; this makes it exported (worry?)
func CLRCHN() {
	kernal_input = 0
	kernal_output = 0
}

var run = []byte{ 'R', 'U', 'N', 13 }

var (
	fakerun			bool = false
	fakerun_index		int = 0
)

/* CHRIN */
// TODO(andlabs) - was static; this makes it exported (worry?)
func CHRIN() {
	if (!interactive) && (readycount == 2) {
		os.Exit(0)
	}
	if kernal_input != 0 {
		if kernal_files_next[kernal_input] == EOF {
			c := make([]byte, 1)
			_, err := kernal_files[kernal_input].Read(c)
			if err == io.EOF {
				kernal_status |= KERN_ST_EOF
				kernal_status |= KERN_ST_TIME_OUT_READ
				A = 13
				goto out
			} else if err != nil {
				// TODO(andlabs)
			} else {
				kernal_files_next[kernal_input] = int(c[0])
			}
		}
		A = byte(kernal_files_next[kernal_input] & 0xFF)
		c := make([]byte, 1)
		_, err := kernal_files[kernal_input].Read(c)
		if err == io.EOF {
			kernal_status |= KERN_ST_EOF
			kernal_files_next[kernal_input] = EOF
		} else if err != nil {
			// TODO(andlabs)
		} else {
			kernal_files_next[kernal_input] = int(c[0])
		}
	out:
	} else if input_file == nil {
		c := make([]byte, 1)
		_, err := os.Stdin.Read(c)
		if err != nil {
			// TODO(andlabs)
		}
		A = c[0]
		if A=='\n' {
			A = '\r'
		}
	} else {
		if fakerun {
			A = run[fakerun_index]
			fakerun_index++
			if fakerun_index == len(run) {
				input_file = nil		// switch to stdin
			}
		} else {
			c := make([]byte, 1)
			_, err := input_file.Read(c)
			if err == io.EOF {
				A = 255		// TODO(andlabs) - is this correct?
			} else if err != nil {
				// TODO(andlabs)
			} else {
				A = c[0]
			}
			if (A == 255) && (readycount == 1) {
				fakerun = true
				fakerun_index = 0
				A = run[fakerun_index]
				fakerun_index++
			}
			if A == '\n' {
				A = '\r'
			}
		}
	}
	C = false
}

/* CHROUT */
// TODO(andlabs) - was static; this makes it exported (worry?)
func CHROUT() {
//return;
//exit(1);
/*
#if 0
int a = *(unsigned short*)(&RAM[0x0100+S+1]) + 1;
int b = *(unsigned short*)(&RAM[0x0100+S+3]) + 1;
int c = *(unsigned short*)(&RAM[0x0100+S+5]) + 1;
int d = *(unsigned short*)(&RAM[0x0100+S+7]) + 1;
printf("CHROUT: %d @ %x,%x,%x,%x\n", A, a, b, c, d);
#endif
*/
	if !interactive {
		if stack4(0xe10f, 0xab4a, 0xab30, 0xe430) {
			/* COMMODORE 64 BASIC V2 */
			C = false
			return
		}
		if stack4(0xe10f, 0xab4a, 0xab30, 0xe43d) {
			/* 38911 */
			C = false
			return
		}
		if stack4(0xe10f, 0xab4a, 0xab30, 0xe444) {
			/* BASIC BYTES FREE */
			C = false
			return
		}
	}
	if stack4(0xe10f, 0xab4a, 0xab30, 0xa47b) {
		/* READY */
		if A == 'R' {
			readycount++
		}
		if !interactive {
			C = false
			return
		}
	}
	if stack4(0xe10f, 0xab4a, 0xaadc, 0xa486) {
		/*
		 * CR after each entered numbered program line:
		 * The CBM screen editor returns CR when the user
		 * hits return, but does not print the character,
		 * therefore CBMBASIC does. On UNIX, the terminal
		 * prints all input characters, so we have to avoid
		 * printing it again
		 */
		C = false
		return
	}
	
//#if 0
//	printf("CHROUT: %c (%d)\n", A, A);
//#else
	if kernal_output != 0 {
		c := []byte{ A }
		_, err := kernal_files[kernal_output].Write(c)
		if err == io.EOF {
			C = true
			A = KERN_ERR_NOT_OUTPUT_FILE
		} else if err != nil {
			// TODO(andlabs)
		} else {
			C = false
		}
	} else {
		if kernal_quote != 0 {		// TODO make kernal_quote a bool?
			if A == '"' || A == '\n' || A == '\r' {
				kernal_quote = 0
			}
            		fmt.Printf("%c", A)
		} else {
			switch A {
			case 5:
				set_color(COLOR_WHITE);
			case 10:
				// do nothing (what is this byte? TODO(andlabs))
			case 13:
				fmt.Printf("%c%c", 13, 10)
			case 17:		// CSR DOWN
				down_cursor()
			case 19:		// CSR HOME
				move_cursor(0, 0)
			case 28:
				set_color(COLOR_RED)
			case 29:		// CSR RIGHT
				right_cursor()
			case 30:
				set_color(COLOR_GREEN)
			case 31:
				set_color(COLOR_BLUE)
			case 129:
				set_color(COLOR_ORANGE)
			case 144:
				set_color(COLOR_BLACK);
			case 145:		// CSR UP
				up_cursor()
			case 147:		// clear screen
//#ifndef NO_CLRHOME
				clear_screen()
//#endif
			case 149:
				set_color(COLOR_BROWN)
			case 150:
				set_color(COLOR_LTRED)
			case 151:
				set_color(COLOR_GREY1)
			case 152:
				set_color(COLOR_GREY2)
			case 153:
				set_color(COLOR_LTGREEN)
			case 154:
				set_color(COLOR_LTBLUE)
			case 155:
				set_color(COLOR_GREY3)
			case 156:
				set_color(COLOR_PURPLE)
			case 158:
				set_color(COLOR_YELLOW)
			case 159:
				set_color(COLOR_CYAN)
			case 157:		// CSR LEFT
				left_cursor()
			case '"':
				kernal_quote = 1
				fallthrough
			default:
				fmt.Printf("%c", A)
			}
		}
//#endif
		C = false
	}
}

/* LOAD */
// TODO(andlabs) - was static; this makes it exported (worry?)
func LOAD() {
	var start uint16
	var end uint16
	var i int

	if A != 0 {
		fatalf("UNIMPL: VERIFY");
	}
	if kernal_filename_len == 0 {
		goto missing_file_name
	}

/* on special filename $ read directory entries and load they in the basic area memory */
	if RAM[kernal_filename] == '$' {
		var file_size int64
		var old_memp uint16			// TODO hack!
		var memp uint16 = 0x0801		// TODO hack!

		old_memp = memp
		memp += 2
		RAM[memp] = 0; memp++
		RAM[memp] = 0; memp++
		RAM[memp] = 0x12; memp++		// REVERS ON
		RAM[memp] = '"'; memp++
		for i = 0; i < 16; i++ {
			RAM[memp + uint16(i)] = ' '
		}
		cwd, err := os.Getwd()
		if err != nil {
			goto device_not_present
		}
		if len(cwd) > 256 {		// TODO(andlabs) - correct for getcwd((char*)&RAM[memp], 256)) / memp += strlen((char*)&RAM[memp]) /* only 16 on COMMODORE DOS */ ?
			cwd = cwd[:256]
		}
		for i = 0; i < len(cwd); i++ {
			RAM[memp] = cwd[i]; memp++
		}
		RAM[memp] = '"'; memp++
		RAM[memp] = ' '; memp++
		RAM[memp] = '0'; memp++
		RAM[memp] = '0'; memp++
		RAM[memp] = ' '; memp++
		RAM[memp] = '2'; memp++
		RAM[memp] = 'A'; memp++
		RAM[memp] = 0; memp++

		RAM[old_memp] = byte(memp & 0xFF)
		RAM[old_memp+1] = byte(memp >> 8)

		dirp, err := os.Open(".")
		if err != nil {
			goto device_not_present
		}
		for {
			fi, err := dirp.Readdir(1)
			if err == io.EOF {
				break
			} else if err != nil {
				// TODO(andlabs)
			}
			dp := fi[0]
			name := dp.Name()
			namlen := len(name)
			file_size = (dp.Size() + 253) / 254	// convert file size from num of bytes to num of blocks(254 bytes)
			if file_size > 0xFFFF {
				file_size = 0xFFFF
			}
			old_memp = memp
			memp += 2
			RAM[memp] = byte(file_size & 0xFF); memp++
			RAM[memp] = byte((file_size >> 8) & 0xFF); memp++
			if file_size < 1000 {
				RAM[memp] = ' '; memp++
				if file_size < 100 {
					RAM[memp] = ' '; memp++
					if file_size < 10 {
						RAM[memp] = ' '; memp++
					}
				}
			}
			RAM[memp] = '"'; memp++
			if namlen > 16 {
				namlen = 16		// TODO hack
			}
			for i = 0; i < namlen; i++ {
				RAM[memp] = name[i]; memp++
			}
			RAM[memp] = '"'; memp++
			for i = namlen; i < 16; i++ {
				RAM[memp] = ' '
				memp++
			}
			RAM[memp] = ' '; memp++
			RAM[memp] = 'P'; memp++
			RAM[memp] = 'R'; memp++
			RAM[memp] = 'G'; memp++
			RAM[memp] = ' '; memp++
			RAM[memp] = ' '; memp++
			RAM[memp] = 0; memp++

			RAM[old_memp] = byte(memp & 0xFF)
			RAM[old_memp+1] = byte(memp >> 8)
		}
		RAM[memp] = 0;
		RAM[memp+1] = 0;
		dirp.Close()
		end = memp + 2
/*
for (i=0; i<255; i++) {
	if (!(i&15))
		printf("\n %04X  ", 0x0800+i);
	printf("%02X ", RAM[0x0800+i]);
}
*/
		goto load_noerr
	} // end if RAM[kernal_filename] == '$'

	filename := string(RAM[kernal_filename:kernal_filename + kernal_filename_len])

	// on directory filename chdir on it
	st, err := os.Stat(filename)
	if err != nil {
		goto file_not_found
	}
	if st.IsDir() {
		if os.Chdir(filename) != nil {
			goto device_not_present
		}

		RAM[0x0801] = 0
		RAM[0x0802] = 0
		end = 0x0803
		goto load_noerr
	}

	// on file load it read it and load in the basic area memory
	f, err := os.Open(filename)
	if err != nil {
		goto file_not_found
	}
	c := make([]byte, 2)
	_, err = f.Read(c)
	if err != nil {
		// TODO(andlabs)
	}
	start = uint16(c[0]) | (uint16(c[1]) << 8)
	if kernal_sec != 0 {
		start = uint16(X) | (uint16(Y) << 8)
	}
	b, err := ioutil.ReadAll(f)	// we cannot read directly into RAM as we cannot guarantee RAM[size of f:] is left alone
	if err != nil {
		// TODO(andlabs)
	}
	end = start
	for i = 0; ; i++ {			// TODO may overwrite ROM
		RAM[end] = b[i]
		if end == 0xFFFF {
			break
		}
		end++
	}
	fmt.Printf("LOADING FROM $%04X to $%04X\n", start, end)
	f.Close()

load_noerr:
	X = byte(end & 0xFF)
	Y = byte(end >> 8)
	C = false
	A = KERN_ERR_NONE
	return

file_not_found:
	C = true
	A = KERN_ERR_FILE_NOT_FOUND
	return

device_not_present:
	C = true
	A = KERN_ERR_DEVICE_NOT_PRESENT
	return

missing_file_name:
	C = true
	A = KERN_ERR_MISSING_FILE_NAME
	return
}

/* SAVE */
// TODO(andlabs) - was static; this makes it exported (worry?)
func SAVE() {
	var start uint16
	var end uint16

	start = uint16(RAM[A]) | (uint16(RAM[A+1]) << 8);
	end = uint16(X) | (uint16(Y) << 8)
	if end < start {
		C = true
		A = KERN_ERR_NONE
		return
	}
	if kernal_filename_len == 0 {
		C = true
		A = KERN_ERR_MISSING_FILE_NAME
		return
	}
	filename := string(RAM[kernal_filename:kernal_filename + kernal_filename_len])
	f, err := os.Create(filename)	// overwrite - these are not the COMMODORE DOS semantics!
	if err != nil {
		C = true
		A = KERN_ERR_FILE_NOT_FOUND
		return
	}
	_, err = f.Write([]byte{
		byte(start & 0xFF),
		byte(start >> 8),
	})
	if err != nil {
		// TODO(andlabs)
	}
	_, err = f.Write(RAM[start:end])
	if err != nil {
		// TODO(andlabs)
	}
	f.Close()
	C = false
	A = KERN_ERR_NONE
}

/* SETTIM */
// TODO(andlabs) - was static; this makes it exported (worry?)
func SETTIM() {
	// no portable way to do it in Go without having code in other files
	// and I don't think being able to set the system time from within cbmbasic is a good idea anyway...
	fatalf("UNIMPL: set time of day")
}

/* RDTIM */
// TODO(andlabs) - was static; this makes it exported (worry?)
func RDTIM() {
	var jiffies uint32

	now := time.Now()
	usec := now.Nanosecond() / int(time.Microsecond)
	jiffies = uint32(((now.Hour()*60 + now.Minute())*60 + now.Second())*60 + usec / (1000000/60))

	Y = byte(jiffies / 65536)
	X = byte((jiffies % 65536) / 256)
	A = byte(jiffies % 256)
}

/* STOP */
// TODO(andlabs) - was static; this makes it exported (worry?)
func STOP() {
	SETZ(0)		// TODO we don't support the STOP key
}

/* GETIN */
// TODO(andlabs) - was static; this makes it exported (worry?)
func GETIN() {
	if kernal_input != 0 {
		if kernal_files_next[kernal_input] == EOF {
			c := make([]byte, 1)
			_, err := kernal_files[kernal_input].Read(c)
			if err == io.EOF {
				kernal_status |= KERN_ST_EOF
				kernal_status |= KERN_ST_TIME_OUT_READ
				A = 199
				goto out
			} else if err != nil {
				// TODO(andlabs)
			} else {
				kernal_files_next[kernal_input] = int(c[0])
			}
		}
		A = byte(kernal_files_next[kernal_input] & 0xFF)
		c := make([]byte, 1)
		_, err := kernal_files[kernal_input].Read(c)
		if err == io.EOF {
			kernal_status |= KERN_ST_EOF
			kernal_files_next[kernal_input] = EOF
		} else if err != nil {
			// TODO(andlabs)
		} else {
			kernal_files_next[kernal_input] = int(c[0])
		}
	out:
		C = false
	} else {
		c := make([]byte, 1)
		_, err := os.Stdin.Read(c)
		if err != nil {
			// TODO(andlabs)
		}
		A = c[0]
		if A == '\n' {
			A = '\r'
		}
		C = false
	}
}

/* CLALL */
// TODO(andlabs) - was static; this makes it exported (worry?)
func CLALL() {
	for i := 0; i < len(kernal_files); i++ {
		if kernal_files[i] != nil {
			kernal_files[i].Close()
			kernal_files[i] = nil
		}
	}
}

/* PLOT */
// TODO(andlabs) - was static; this makes it exported (worry?)
func PLOT() {
	if C {
		var CX, CY int

		get_cursor(&CX, &CY)
		Y = byte(CX)
		X = byte(CY)
	} else {
		fatalf("UNIMPL: set cursor %d %d\n", Y, X);
	}
}


/* IOBASE */
// TODO(andlabs) - was static; this makes it exported (worry?)
func IOBASE() {
	const CIA = 0xDC00		// we could put this anywhere...

	/*
	 * IOBASE is just used inside RND to get a timer value.
	 * So, let's fake this here, too.
	 * Commodore BASIC reads offsets 4/5 and 6/7 to get the
	 * two timers of the CIA.
	 */
	pseudo_timer := uint(rand.Intn(65536))
	RAM[CIA+4] = byte(pseudo_timer & 0xFF)
	RAM[CIA+5] = byte((pseudo_timer >> 8) & 0xFF)
	pseudo_timer = uint(rand.Intn(65536))		// more entropy!
	RAM[CIA+8] = byte(pseudo_timer & 0xFF)
	RAM[CIA+9] = byte((pseudo_timer >> 8) & 0xFF)
	X = byte(CIA & 0xFF)
	Y = byte((CIA >> 8) & 0xFF)
}

func kernal_dispatch() int {
//{ printf("kernal_dispatch $%04X; ", PC); int i; printf("stack (%02X): ", S); for (i=S+1; i<0x100; i++) { printf("%02X ", RAM[0x0100+i]); } printf("\n"); }
	var new_pc uint16

	switch(PC) {
	case 0x0073:
		CHRGET()
	case 0x0079:
		CHRGOT()
	case 0xFF90:
		SETMSG()
	case 0xFF99:
		MEMTOP()
	case 0xFF9C:
		MEMBOT()
	case 0xFFB7:
		READST()
	case 0xFFBA:
		SETLFS()
	case 0xFFBD:
		SETNAM()
	case 0xFFC0:
		OPEN()
	case 0xFFC3:
		CLOSE()
	case 0xFFC6:
		CHKIN()
	case 0xFFC9:
		CHKOUT()
	case 0xFFCC:
		CLRCHN()
	case 0xFFCF:
		CHRIN()
	case 0xFFD2:
		CHROUT()
	case 0xFFD5:
		LOAD()
	case 0xFFD8:
		SAVE()
	case 0xFFDB:
		SETTIM()
	case 0xFFDE:
		RDTIM()
	case 0xFFE1:
		STOP()
	case 0xFFE4:
		GETIN()
	case 0xFFE7:
		CLALL()
	case 0xFFF0:
		PLOT()
	case 0xFFF3:
		IOBASE()

	case 0:
		plugin_off()
		S += 2
	case 1:
		plugin_on()
		S += 2

	case MAGIC_ERROR:
		new_pc = plugin_error()
		if new_pc != 0 {
			PUSH_WORD(new_pc - 1)
		} else {
			PUSH_WORD(orig_error - 1)
		}
	case MAGIC_MAIN:
		new_pc = plugin_main()
		if new_pc != 0 {
			PUSH_WORD(new_pc - 1)
		} else {
			PUSH_WORD(orig_main - 1)
		}
	case MAGIC_CRNCH:
		new_pc = plugin_crnch()
		if new_pc != 0 {
			PUSH_WORD(new_pc - 1)
		} else {
			PUSH_WORD(orig_crnch - 1)
		}
	case MAGIC_QPLOP:
		new_pc = plugin_qplop()
		if new_pc != 0 {
			PUSH_WORD(new_pc - 1)
		} else {
			PUSH_WORD(orig_qplop - 1)
		}
	case MAGIC_GONE:
		new_pc = plugin_gone()
		if new_pc != 0 {
			PUSH_WORD(new_pc - 1)
		} else {
			PUSH_WORD(orig_gone-1)
		}
	case MAGIC_EVAL:
		new_pc = plugin_eval()
		if new_pc != 0 {
			PUSH_WORD(new_pc - 1)
		} else {
			PUSH_WORD(orig_eval - 1)
		}
		
	case MAGIC_CONTINUATION:
		/*printf("--CONTINUATION--\n");*/
		return 0

//#if 0
//	default:
//		printf("unknown PC=$%04X S=$%02X\n", PC, S);
//		exit(1);
//#else
	default:
		return 1
//#endif
	}
	return 1
}
