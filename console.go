package main

const (
	COLOR_WHITE = iota
	COLOR_RED
	COLOR_GREEN
	COLOR_BLUE
	COLOR_BLACK
	COLOR_PURPLE
	COLOR_YELLOW
	COLOR_CYAN
	COLOR_ORANGE
	COLOR_BROWN
	COLOR_LTRED
	COLOR_GREY1
	COLOR_GREY2
	COLOR_LTGREEN
	COLOR_LTBLUE
	COLOR_GREY3
)

func clear_screen() {}
func up_cursor() {}
func down_cursor() {}
func left_cursor() {}
func right_cursor() {}
func move_cursor(byte, byte) {}
func get_cursor(*int, *int) {}
func set_color(int) {}
