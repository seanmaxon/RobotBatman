package main

const (
	batUDPPortStr = "30703"
	batUDPPortInt = 30703

	batSQNAddrSize = 2048 // Value at which sequence numbers roll over

	batLocalWindowSize = 64 // Window size for link quality estimating
	batCutoffRQSamples = 10
	batCutoffEQSamples = 10
	batCutoffTQ        = 10

	batTQMaxValue   = 255
	batTQHopPenalty = 10

	batTTL            = 16 // OGM packet Time To Live (number of forwarding hops)
	batOGMSize        = 26
	batSafePacketSize = 512 // ToDo(Sean): Make this a per-link (or link type) thing
	batMaxBundleSize  = 19  // Max OGMs bundled together;  batOGMSize * batMaxBundleSize < batSafePacketSize
	batMaxBundleDelay = 200 // Milliseconds to delay transmission waiting for more OGMs

	batOGMInterval = 1   // Seconds between sending own OGM
	batOGMJitter   = 100 // (Milliseconds) Max additive variation for randomized OGM interval
)
