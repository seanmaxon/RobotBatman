package main

import "fmt"

// sqn is a simple extension of integers with modulo arithmetic and
// comparisons intended for use with BATMAN OGM packet sequence numbers.
type sqn struct {
	num    int
	limit  int
	window int
}

func newSQN(num, limit, window int) sqn {
	// Input error checks
	if (num < 0) || (num >= limit) {
		panic(fmt.Sprint("batman: newSQN: value outside address range ", num, limit))
	}
	if window > limit {
		panic(fmt.Sprint("batman: newSQN: window larger than address range ", window, limit))
	}

	return sqn{
		num:    num,
		limit:  limit,
		window: window,
	}
}

func newDefaultSQN(num int) sqn {
	return newSQN(num, batSQNAddrSize, batLocalWindowSize)
}

func (s *sqn) raw() uint32 {
	// ToDo(Sean): Add error check that num < size(uint32)

	return uint32(s.num)
}

func (s *sqn) increment() {
	s.num = pmod((s.num + 1), s.limit)
}

func (s sqn) add(n sqn) sqn {
	return sqn{
		num:    pmod((s.num + n.num), s.limit),
		limit:  s.limit,
		window: s.window,
	}
}

func (s sqn) subtract(n sqn) sqn {
	return sqn{
		num:    pmod((s.num - n.num), s.limit),
		limit:  s.limit,
		window: s.window,
	}
}

func (s sqn) equalTo(n sqn) bool {
	return s.num == n.num
}

// A.greaterThan(B)
// For SQN-A and SQN-B within one window distance of
// each other, A is defined to be greater if
//    B+x modulo limit == A, for some int x < window.
//
// SQN-A is defined as being greater than SQN-B if
// A is outside the window that's greatest SQN is B.
//
// Note this allows both A > B and B > A to be true
// simultaneously for some cases.
func (s sqn) greaterThan(n sqn) bool {
	dif := min(s.num-n.num, s.num-n.num-s.limit)
	if pmod(abs(dif), s.limit) < s.window { // If within one window distance
		return (0 < pmod(dif, s.limit)) && (pmod(dif, s.limit) < s.window)
	}
	return true // Otherwise we default to assuming true
}

func (s sqn) lessThan(n sqn) bool {
	return !(s.greaterThan(n)) && !(s.equalTo(n))
}

func (s sqn) String() string {
	if (s.limit != 0) && (s.window != 0) {
		return fmt.Sprintf("%4d", s.num)
	}
	return "none"
}
