package main

import (
	"fmt"
	"testing"
)

var sampleOGM = RawOGM{
	Origin:     [4]byte{0, 0, 0, 1},
	Sender:     [4]byte{0, 0, 0, 2},
	TxAddr:     [4]byte{0, 0, 0, 3},
	PrevSender: [4]byte{0, 0, 0, 4},
	PrevAddr:   [4]byte{0, 0, 0, 5},
	SQN:        42,
	TTL:        2,
	Quality:    128,
}

func TestPackUnpackSingle(t *testing.T) {
	b := make([]byte, 0, batSafePacketSize)
	err := packOGMs(&b, []RawOGM{sampleOGM})
	if err != nil {
		t.Error("ogm error: packing and unpacking:", err)
	}
	if len(b) == 0 {
		t.Error("WHY")
	}
	ogms, err := parseOGMs(b, "")
	if err != nil {
		t.Error("ogm error: packing and unpacking:", err)
	}
	if sampleOGM != ogms[0].Pack() {
		t.Error("ogm error: packing and unpacking inconsistency:", sampleOGM, ogms[0])
	}
}

func TestPackUnpackTwo(t *testing.T) {
	b := make([]byte, 0, batSafePacketSize)
	packOGMs(&b, []RawOGM{sampleOGM, sampleOGM})
	ogms, err := parseOGMs(b, "")
	if err != nil {
		t.Error("ogm error: packing and unpacking with buldle:", err)
	}
	if sampleOGM != ogms[0].Pack() && sampleOGM != ogms[1].Pack() {
		t.Error("ogm error: packing and unpacking inconsistency with bundle")
	}
}

func TestSimpleOGMRawConversion(t *testing.T) {
	s := OGM{
		Origin:     "5",
		Sender:     "7",
		TxAddr:     "10.4.6.2",
		PrevSender: "2",
		PrevAddr:   "192.168.10.10",
		SQN:        newDefaultSQN(102),
		TTL:        200,
		Quality:    250,
	}
	o := s.Pack()
	s2 := o.Unpack()
	if s != s2 {
		t.Error("simpleOGM conversion error:", fmt.Sprintf("%T, %T, %#v, %#v", s, s2, s, s2))
	}
}

func TestOGMRawSimpleOGMConversion(t *testing.T) {
	sim := sampleOGM.Unpack()
	raw := sim.Pack()

	if raw != sampleOGM {
		t.Error("OGM raw conversion error:", sampleOGM, raw)
	}
}

// ToDo(Sean): Add tests for ogmQueue
