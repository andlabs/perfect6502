/*
 * Copyright (c) 2009 Michael Steil
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

/*
 * This plugin interface makes use of the standard plugin facility built into
 * Commodore BASIC that is used by BASIC extensions such as Simons' BASIC.
 * There are several vectors at 0x0300 in RAM for functions like error printing,
 * tokenizing, de-tokenizing and the interpreter loop. We hook this from C.
 * Since this adds code to the interpreter loop, it is disabled by default,
 * and can be enabled like this:
 *
 * SYS 1
 *
 * It can be disabled with:
 *
 * SYS 0
 *
 * Please note that the current implementation does not tokenize new keywords,
 * but stores them verbatim and compares strings when during execution, which
 * is very bad for performance. Also, there is currently no demo code for
 * added functions.
 */

import (
	"fmt"
	"os"
//	"os/exec"
)

func get_chrptr() uint16 {
	return uint16(RAM[0x7A]) | (uint16(RAM[0x7B]) << 8)
}

func set_chrptr(a uint16) {
	RAM[0x7A] = byte(a & 0xFF)
	RAM[0x7B] = byte(a >> 8)
}

func compare(s string) bool {
	var chrptr uint16 = get_chrptr()

	for i := 0; i < len(s); i++ {
		CHRGET()
		if A != s[i] {
			set_chrptr(chrptr);
			return false
		}
	}
	CHRGET()
	return true
}

/*
 * Continuation
 *
 * This will put a magic value onto the stack and run the main
 * function again with another PC value as a start address.
 * When the code returns, it will find the magic value, and
 * the main function will quit, so we end up here again.
 */
func call(pc uint16) {
	PC = pc
	PUSH_WORD(MAGIC_CONTINUATION - 1)
	main()		// TODO(andlabs) - handled?
}

func check_comma() {
	call(0xAEFD)
}

func get_word() uint16 {
	call(0xAD8A)
	call(0xB7F7)
	return uint16(RAM[0x14]) | (uint16(RAM[0x15]) << 8)
}

func get_byte() byte {
	call(0xB79E)
	return X;
}

func get_string() string {
	call(0xAD9E)
	call(0xB6A3)
	base := uint16(X) | (uint16(Y) << 8)
	return string(RAM[base:base + uint16(A)])
}

const (
	ERROR_TOO_MANY_FILES = 0x01
	ERROR_FILE_OPEN = 0x02
	ERROR_FILE_NOT_OPEN = 0x03
	ERROR_FILE_NOT_FOUND = 0x04
	ERROR_DEVICE_NOT_PRESENT = 0x05
	ERROR_NOT_INPUT_FILE = 0x06
	ERROR_NOT_OUTPUT_FILE = 0x07
	ERROR_MISSING_FILE_NAME = 0x08
	ERROR_ILLEGAL_DEVICE_NUMBER = 0x09
	ERROR_NEXT_WITHOUT_FOR = 0x0A
	ERROR_SYNTAX = 0x0B
	ERROR_RETURN_WITHOUT_GOSUB = 0x0C
	ERROR_OUT_OF_DATA = 0x0D
	ERROR_ILLEGAL_QUANTITY = 0x0E
	ERROR_OVERFLOW = 0x0F
	ERROR_OUT_OF_MEMORY = 0x10
	ERROR_UNDEFD_STATMENT = 0x11
	ERROR_BAD_SUBSCRIPT = 0x12
	ERROR_REDIMD_ARRAY = 0x13
	ERROR_DEVISION_BY_ZERO = 0x14
	ERROR_ILLEGAL_DIRECT = 0x15
	ERROR_TYPE_MISMATCH = 0x16
	ERROR_STRING_TOO_LONG = 0x17
	ERROR_FILE_DATA = 0x18
	ERROR_FORMULA_TOO_COMPLEX = 0x19
	ERROR_CANT_CONTINUE = 0x1A
	ERROR_UNDEFD_FUNCTION = 0x1B
	ERROR_VERIFY = 0x1C
	ERROR_LOAD = 0x1D
	ERROR_BREAK = 0x1E
)

// originally error(), renamed to avoid conflict with Go error
func error_x(index byte) uint16 {
	X = index;
	return 0xA437		// error handler
}

/*
 * Print BASIC Error Message
 *
 * We could add handling of extra error codes here, or
 * print friendlier strings, or implement "ON ERROR GOTO".
 */
func plugin_error() uint16 {
	return 0
}

/*
 * BASIC Warm Start
 *
 * This gets called whenever we are in direct mode.
 */
func plugin_main() uint16 {
	return 0
}

/*
 * Tokenize BASIC Text
 */
func plugin_crnch() uint16 {
	return 0
}

/*
 * BASIC Text LIST
 */
func plugin_qplop() uint16 {
	return 0
}

/*
 * BASIC Char. Dispatch
 *
 * This is used for interpreting statements.
 */
func plugin_gone() uint16 {
	set_chrptr(get_chrptr() + 1)
	for {
		var chrptr uint16

		set_chrptr(get_chrptr() - 1)
		chrptr = get_chrptr()

		/*
		 * this example shows:
		 * - how to get a 16 bit integer
		 * - how to get an 8 bit integer
		 * - how to check for a comma delimiter
		 * - how to do error handling
		 */
		if compare("LOCATE") {
			var x, y byte

			y = get_byte()		// 'line' first
			check_comma()
			x = get_byte()		// then 'column'
			/* XXX ignores terminal size */
			if x > 80 || y > 25 || x == 0 || y == 0 {
				return error_x(ERROR_ILLEGAL_QUANTITY)
			}
			move_cursor(x, y)

			continue
		}

		/*
		 * this example shows:
		 * - how to override existing keywords
		 * - how to hand the instruction to the
		 *   original interpreter if we don't want
		 *   to handle it
		 */
		if compare("\222") {		// 0x92 - WAIT
			var a uint16

			a = get_word()
			check_comma()
			get_byte()
			if a == 6502 {
				fmt.Printf("MICROSOFT!")
				continue
			} else {
				set_chrptr(chrptr)
				return 0
			}
		}

		/*
		 * this example shows:
		 * - how to deal with new keywords that contain
		 *   existing keywords
		 * - how to parse a string
		 */
//		if compare("\236TEM") {
//			s = get_string()
//			// TODO(andlabs) - UNIX ONLY
//			exec.Command("sh", "-c", s).Run()
//
//			continue
//		}

		if compare("QUIT") {
			os.Exit(0)
		}
		break
	}
	return 0
}

/*
 * BASIC Token Evaluation
 *
 * This is used for expression evaluation.
 * New functions and operators go here.
 */
func plugin_eval() uint16 {
	return 0
}
