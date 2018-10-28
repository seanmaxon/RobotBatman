package main

import "testing"

func TestWindowRing(t *testing.T) {
	var wr *windowRing
	var correctRing []byte

	// helper function
	checkRing := func(r1, r2 []byte) {
		for i := range r1 {
			if r1[i] != r2[i] {
				t.Error("windowRing: newWindowRing: ", r1)
				break
			}
		}
	}

	// Constructor function tests
	wr = newWindowRing(64, 8, 255)
	correctRing = []byte{255, 255, 255, 255, 255, 255, 255, 255}
	checkRing(wr.ring, correctRing)

	wr = newWindowRing(64, 8, 0)
	correctRing = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	checkRing(wr.ring, correctRing)

	// Internal windowRing circular buffer tests
	// and inWindow method tests
	wr = newWindowRing(32, 4, 0)
	correctRing = []byte{0, 0, 0, 0}
	checkRing(wr.ring, correctRing)
	wr.write(0, 255)
	correctRing = []byte{255, 0, 0, 0}
	checkRing(wr.ring, correctRing)
	wr.write(1, 1)
	correctRing = []byte{255, 1, 0, 0}
	checkRing(wr.ring, correctRing)
	wr.write(2, 2)
	correctRing = []byte{255, 1, 2, 0}
	checkRing(wr.ring, correctRing)
	wr.write(3, 3)
	correctRing = []byte{255, 1, 2, 3}
	checkRing(wr.ring, correctRing)
	if wr.inWindow(1) != true {
		t.Error("windowRing: inWindow error")
	}
	wr.write(4, 4)
	correctRing = []byte{4, 1, 2, 3}
	checkRing(wr.ring, correctRing)
	wr.write(5, 5)
	correctRing = []byte{4, 5, 2, 3}
	checkRing(wr.ring, correctRing)
	if wr.inWindow(1) != false {
		t.Error("windowRing: inWindow error") // falling out of buffer
	}
	if wr.inWindow(30) != false {
		t.Error("windowRing: inWindow error")
	}
	wr.write(30, 30)
	correctRing = []byte{0, 30, 0, 0}
	checkRing(wr.ring, correctRing)
	if wr.inWindow(30) != true {
		t.Error("windowRing: inWindow error")
	}

	wr = newWindowRing(64, 8, 0)
	wr.write(39, 39)
	wr.write(40, 40)
	correctRing = []byte{39, 40, 0, 0, 0, 0, 0, 0}
	checkRing(wr.ring, correctRing)
	wr.write(47) // valueless format test
	correctRing = []byte{0, 40, 0, 0, 0, 0, 0, 0}
	checkRing(wr.ring, correctRing)

	// String test
	wr = newWindowRing(1024, 64, 0)
	for i := 0; i < 24; i++ {
		wr.write(i, 255)
	}
	wr.write(50, 255)
	if wr.String() != "<windowRing{64}: head:0050 (|||||||!........!...)>" {
		t.Error(wr.String())
	}

	// Write and read indexing tests
	wr = newWindowRing(2048, 64, 0)
	wr.write(0, 255)
	if val, ok := wr.read(0); (val != 255) || ok != true {
		t.Error("windowRing: read error")
	}
	wr.write(1, 1)
	if val, ok := wr.read(1); (val != 1) || ok != true {
		t.Error("windowRing: read error")
	}
	wr.write(50, 50)
	if val, ok := wr.read(50); (val != 50) || ok != true {
		t.Error("windowRing: read error")
	}
	if val, ok := wr.read(0); (val != 255) || ok != true {
		t.Error("windowRing: read error")
	}
	wr.write(800, 80)
	if val, ok := wr.read(800); (val != 80) || ok != true {
		t.Error("windowRing: read error")
	}
	if val, ok := wr.read(50); (val != 0) || ok != false {
		t.Error("windowRing: read error")
	}

	// Counting value and eval func tests
	wr = newWindowRing(1024, 64, 0)
	for i := 0; i < 24; i++ {
		wr.write(i, 255)
	}
	for i := 40; i < 50; i++ {
		wr.write(i, byte(i))
	}
	if c := wr.countHits(255); c != 24 {
		t.Error("windowRing: countHits error: ", c)
	}
	if c := wr.countHits(40); c != 1 {
		t.Error("windowRing: countHits error: ", c)
	}
	if c := wr.countHitsFunc(func(b byte) bool {
		if b > 45 {
			return true
		}
		return false
	}); c != 24+4 {
		t.Error("windowRing: countHitsFunc error: ", c)
	}
}
