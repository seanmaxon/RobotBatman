package main

// pmod implements the Python-style integer modulo
func pmod(x, y int) (int) {
	return (x % y + y) % y
}

// integer min
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// integer max
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// integer absolute value
func abs(x int) int {
	if x >= 0 {
		return x
	}
	return -x
}