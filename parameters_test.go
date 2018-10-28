package main

import "testing"

func TestMaxOGMPacketSize(t *testing.T) {
	if batOGMSize*batMaxBundleSize > batSafePacketSize {
		t.Error("parameters error: too many OGMs in a bundle")
	}
}

func TestSizeOfOGM(t *testing.T) {
	// Relies on the fact than an OGM bundle has an overhead of one byte.

	b := make([]byte, batSafePacketSize)
	packOGMs(&b, []RawOGM{sampleOGM})
	if len(b) != batOGMSize+1 {
		t.Error("parameters error: batOGMSize does not match raw OGM byte count")
	}
}
