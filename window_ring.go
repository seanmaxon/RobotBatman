package main

import "fmt"

// windowRing is a small sliding window buffer indexed by a large looping
// address space.
//
// Used by BATMAN to track link measurements (RQ, EQ) and path metrics (TQ).
//
// A value can be registered at any location within the address space.
// If the given location is outside the current window, the head will shift
// forward until the location is reached, padding with the default value as
// needed. Any data that falls outside the current window is lost.
//
// Conceptual view: (Implementation differs in that the window is implemented
//                   as a circular buffer with it's own moving windowhead.)
//
//          |<-winSize->|
//          |           |
//          v           v
//   |-----|0|X|0|X|0|0|0|----------------------------------|
//                      ^                                  ^
//                      |                                  |
//                      +---- addressHead                  +--- addressSize-1
type windowRing struct {
	ring        []byte
	addressSize int
	windowSize  int
	defaultVal  byte
	windowHead  int // Note that the windowHead and addressHead have no fixed
	addressHead int // relationship; they are always manipulated relativity.
}

func newWindowRing(addressSize, windowSize int, defaultVal byte) *windowRing {
	// Error check input
	if addressSize < windowSize {
		panic(fmt.Sprint("batman: newWindowRing: addressSize < windowSize: ", addressSize, "<", windowSize))
	}

	w := &windowRing{
		ring:        make([]byte, windowSize),
		addressSize: addressSize,
		windowSize:  windowSize,
		defaultVal:  defaultVal,
	}

	if defaultVal != 0 {
		for i := range w.ring {
			w.ring[i] = defaultVal
		}
	}

	return w
}

// inWindow tests if the given location, loc, from the large address
// space is within the current sliding window.
//
// Distance is measured counting back from head until reaching loc; this
// counting may loop around the address space.
func (w windowRing) inWindow(loc int) bool {
	distance := pmod((w.addressHead - loc), w.addressSize)
	return distance < w.windowSize
}

// Write the given value into the sliding window for the address
// given by loc. If the sliding window does not currently cover the
// given address, the window is shifted so that the addressHead
// points to the given address.
//
// Usage Notes:
//     calling write without a val does nothing if the given address is
//     within the window, but shifts the window head to loc and writes
//     the default value if it is not.
//
// Developer note: In practice, it seems the only value ever written is
// the TQ_MAX_VALUE with a value of 255
func (w *windowRing) write(loc int, val ...byte) {
	// Error checking
	if loc < 0 {
		panic(fmt.Sprintf("windowRing: write: negative loc"))
	}
	if loc > w.addressSize {
		panic(fmt.Sprintf("windowRing: write: loc exceeds addressSize"))
	}
	if len(val) > 1 {
		panic(fmt.Sprintf("windowRing: write: too many values"))
	}

	var windowWriteIndex int

	if w.inWindow(loc) {
		// The head need not be advanced.
		distFromHead := pmod((w.addressHead - loc), w.addressSize)
		windowWriteIndex = pmod((w.windowHead - distFromHead), w.windowSize)
	} else {
		// Advance head.
		moveBy := pmod((loc - w.addressHead), w.addressSize) // how far to advance head
		w.addressHead = pmod((moveBy + w.addressHead), w.addressSize)

		// Assert to catch bugs?
		if w.addressHead != loc {
			panic(fmt.Sprintf("windowRing: write: addressHead/loc mismatch! loc:%d, addressHead:%d", loc, w.addressHead))
		}

		// Clear windowRing elements newly exposed by advancing head.
		for i := 0; i < moveBy && i < w.windowSize; i++ { // at most, clear all local elements
			w.windowHead = pmod((1 + w.windowHead), w.windowSize)
			w.ring[w.windowHead] = w.defaultVal
			windowWriteIndex = w.windowHead
		}
	}

	if len(val) > 0 {
		w.ring[windowWriteIndex] = val[0]
	}
}

// Read returns the value stored at the given address, if that address is
// within the active window. Otherwise it returns val as 0 and ok as false
func (w windowRing) read(loc int) (val byte, ok bool) {
	distance := pmod((w.addressHead - loc), w.addressSize)
	if distance < w.windowSize { // w.inWindow(loc) == true
		index := pmod(w.windowHead-distance, w.windowSize)

		val = w.ring[index]
		ok = true
	}
	return
}

// countHits returns the number of exact matches of val in the ring.
func (w windowRing) countHits(val byte) int {
	count := 0
	for _, v := range w.ring {
		if val == v {
			count++
		}
	}
	return count
}

// countHitsFunc returns the number of successes of the given test function
// applied to each element of the ring.
func (w windowRing) countHitsFunc(f func(byte) bool) int {
	count := 0
	for _, v := range w.ring {
		if f(v) {
			count++
		}
	}
	return count
}

// String constructs a visual text-based representation of the state of the WindowRing.
func (w *windowRing) String() string {
	width := 20
	factor := (float64(len(w.ring)) - 0.5) / float64(width)
	pic := make([]byte, width)
	for i := 0; i < width; i++ {
		empty := true
		full := true
		for n := int(float64(i) * factor); n < int(float64(i+1)*factor); n++ {
			if w.ring[n] > 0 {
				empty = false
			}
			if w.ring[n] == 0 {
				full = false
			}
		}
		switch {
		case empty && full:
			pic[i] = ' '
		case empty:
			pic[i] = '.'
		case full:
			pic[i] = '|'
		default:
			pic[i] = '!'
		}
	}
	return fmt.Sprintf("<windowRing{%d}: head:%04d (%s)>", w.windowSize, w.addressHead, string(pic))
}
