// routing metrics

package main

import (
	"fmt"
	"time"
)

// The nodeLinksMap type maps link-specific IP addresses to link-tracking
// data structures that store measurements of the quality of the links.
type nodeLinksMap map[ipAddr]*linkData

func newNodeLinkMap() nodeLinksMap {
	return make(nodeLinksMap)
}

func (nlm nodeLinksMap) markReceive(ip ipAddr, seq sqn, when time.Time) {
	if _, ok := nlm[ip]; !ok {
		nlm.addLink(ip)
	}
	for ipKey, linkPtr := range nlm {
		if ipKey == ip {
			linkPtr.markReceive(seq, batTQMaxValue, when)
		} else {
			linkPtr.markReceive(seq, 0, linkPtr.seen) // Marking 0 shifts window but does not write
		}
	}
}

func (nlm nodeLinksMap) markEcho(ip ipAddr, seq sqn, when time.Time) {
	for ipKey, linkPtr := range nlm {
		if ipKey == ip {
			linkPtr.markEcho(seq, batTQMaxValue, when)
		} else {
			linkPtr.markEcho(seq, 0, linkPtr.seen) // Marking 0 shifts window but does not write
		}
	}
}

func (nlm nodeLinksMap) addLink(ip ipAddr) {
	linkPtr := newlinkData()
	nlm[ip] = linkPtr
}

func (nlm nodeLinksMap) update() {
	// TODO(Sean): Write this function
}

// linkData is used for tracking bidirectional link quality of a single link (IP address)
type linkData struct {
	tq       byte
	rqWindow *windowRing
	eqWindow *windowRing
	seen     time.Time
}

func newlinkData() *linkData {
	rqWindow := newWindowRing(batSQNAddrSize, batLocalWindowSize, 0)
	eqWindow := newWindowRing(batSQNAddrSize, batLocalWindowSize, 0)
	return &linkData{0, rqWindow, eqWindow, time.Time{}}
}

func (link *linkData) markReceive(seq sqn, value byte, when time.Time) {
	link.rqWindow.write(seq.num, value)
	if when.After(link.seen) {
		link.seen = when
	}
	link.updateTQ()
}

func (link *linkData) markEcho(seq sqn, value byte, when time.Time) {
	link.eqWindow.write(seq.num, value)
	if when.After(link.seen) {
		link.seen = when
	}
	link.updateTQ()
}

// updateTQ calculates a new TQ metric value from the EQ and RQ windows
func (link *linkData) updateTQ() {
	// Samples of link loss/success rates are used to estimate EQ and RQ.
	// Local link receive events are assumed to be marked using non-zero values.
	countEQ := link.eqWindow.windowSize - link.eqWindow.countHits(0) // EQ = countEQ/LOCAL_WINDOW_SIZE
	countRQ := link.rqWindow.windowSize - link.rqWindow.countHits(0) // RQ = countRQ/LOCAL_WINDOW_SIZE

	// These EQ & RQ estimates are used to compute a raw TQ probability.
	// The final TQ value is obtained by applying an asymmetric adjustment
	// that nonlinearly penalizes poor RQ.
	var tq byte
	switch {
	case countRQ < batCutoffRQSamples || countEQ < batCutoffEQSamples: // Minimum threshold
		tq = 0
	case countRQ < countEQ: // Prevent situation where tq > TQ_MAX_VALUE
		tq = batTQMaxValue
	default: // Ordinary case
		rawTQ := countEQ * batTQMaxValue / countRQ
		// Asymmetric link penalization
		// The following integer calculation is equivalent to:
		//   255*[1-(1-RQ)^3] == 255-(255*(window-count_rq)^3)/(window^3)
		localWinSize := link.eqWindow.windowSize
		tqAsymPenalty := (batTQMaxValue - (batTQMaxValue*
			(localWinSize-countRQ)*
			(localWinSize-countRQ)*
			(localWinSize-countRQ))/
			(localWinSize*localWinSize*localWinSize))
		tq = byte(rawTQ * tqAsymPenalty / batTQMaxValue)

		if tq < batCutoffTQ {
			tq = 0
		}
	}
	// Store the resulting TQ value for the link
	link.tq = tq
}

func (link *linkData) String() string {
	return fmt.Sprintf("<linkData: TQ=%d, RQ=%.1f%%, EQ=%.1f%%, Age=%v>",
		link.tq,
		float64(link.rqWindow.countHits(batTQMaxValue))/batLocalWindowSize,
		float64(link.eqWindow.countHits(batTQMaxValue))/batLocalWindowSize,
		time.Since(link.seen))
}

// The globalNodesMap type maps node IDs to the data structures that store the
// measurements of the quality of each possible next hop routing path to that node.
type globalNodesMap map[nodeID]routingMetricMap

func (global globalNodesMap) updateNode(id nodeID, ip ipAddr, tq byte, num sqn, when time.Time) {
	if routingMetric, ok := global[id]; ok {
		routingMetric.updatePath(ip, tq, num, when)
	} else {
		global.addNode(id, ip, tq, num, when)
	}
}

// internal use only
func (global globalNodesMap) addNode(id nodeID, ip ipAddr, tq byte, num sqn, when time.Time) {
	routingMetric := make(routingMetricMap)
	routingMetric.updatePath(ip, tq, num, when)
	global[id] = routingMetric
}

// ToDo(Sean): Critical: Investigate / merge redundent or duplicate routingMetricMap/routingTableMap and pathData/hop
// Since it's not TQ Metric-spesific, it probably makes most sense to move move/merge implementation into route_tracking.go

// The routingMetricMap type maps possibly-multiple IP addresses of a single node
// to the pathData structures used for tracking the quality of each path.
type routingMetricMap map[ipAddr]pathData

func (routing routingMetricMap) updatePath(ip ipAddr, tq byte, num sqn, when time.Time) {
	if path, ok := routing[ip]; ok {
		if num.greaterThan(path.sqn) {
			path.tq = tq
			path.sqn = num
			path.seen = when

			routing[ip] = path
		}
	} else {
		routing.addPath(ip, tq, num, when)
	}
}

// internal use only
func (routing routingMetricMap) addPath(ip ipAddr, tq byte, num sqn, when time.Time) {
	path := pathData{tq, num, when}
	routing[ip] = path
}

// pathData is used for tracking possible routes (next hops) and their TQ metric values
// (distance vectors) for a single destination node.
type pathData struct {
	tq   byte
	sqn  sqn
	seen time.Time
}

func (path *pathData) String() string {
	return fmt.Sprintf("<TQ=%d, SQN=%v, Age=%d>", path.tq, path.sqn, time.Since(path.seen))
}

// A linkMetric is for tracking local bidirectional link quality of a single link (address)
//
// BATMAN tracks link quality in terms of two measured quantities:
//   Receive Quality (RQ) -- Conceptually, the percentage of gaps in their OGM SEQ#s
//   Echo Quality (EQ)    -- Conceptually, the percentage of our own OGMs echoed back
// From these it computes Transmission Quality (TQ), as a function of EQ and RQ.
type linkMetric struct {
	tq        byte
	rqWindow  *windowRing
	eqWindow  *windowRing
	latestSQN sqn
}

func newLinkMetric(seqNum sqn) *linkMetric {
	var tq byte
	rqWindow := newWindowRing(batSQNAddrSize, batLocalWindowSize, 0)
	eqWindow := newWindowRing(batSQNAddrSize, batLocalWindowSize, 0)
	return &linkMetric{tq, rqWindow, eqWindow, newSQN(seqNum.num, batSQNAddrSize, batLocalWindowSize)}
}

// Part of the BATMAN metric metric known as "TQ", this function
// defines how a link metric value is computed:
//
//     o Samples of link loss/success rates are used to estimate
//       EQ and RQ.
//     o These estimates are used to compute a raw TQ probability.
//     o The final TQ value is obtained by applying an asymmetric
//       adjustment that nonlinearly penalizes poor RQ.
func (m *linkMetric) updateTQ() {
	// Local link receive events are assumed to be marked by writing TQ_MAX_VALUE
	sumEQ := m.eqWindow.countHits(batTQMaxValue) // EQ = sumEQ/LOCAL_WINDOW_SIZE
	sumRQ := m.rqWindow.countHits(batTQMaxValue) // RQ = sumRQ/LOCAL_WINDOW_SIZE

	var tq byte
	switch {
	case sumRQ < batCutoffRQSamples || sumEQ < batCutoffEQSamples: // Threshold
		tq = 0
	case sumRQ < sumEQ: // Prevent situation where TQ > Max
		tq = batTQMaxValue
	default: // Ordinary case
		rawTQ := sumEQ * batTQMaxValue / sumRQ
		// Asymmetric link penalization:
		//   The following integer calculation is equivalent to,
		//    255*[1-(1-RQ)^3] == 255-(255*(window-sum_rq)^3)/(window^3)
		tqAsymPenalty := (batTQMaxValue - (batTQMaxValue*
			(batLocalWindowSize-sumRQ)*
			(batLocalWindowSize-sumRQ)*
			(batLocalWindowSize-sumRQ))/
			(batLocalWindowSize*batLocalWindowSize*batLocalWindowSize))
		tq = byte(rawTQ * tqAsymPenalty / batTQMaxValue)

		if tq < batCutoffTQ {
			tq = 0
		}
	}

	// Store the resulting TQ value for the link
	m.tq = tq
}

func (m *linkMetric) String() string {
	return fmt.Sprintf("<linkMetric: TQ=%d, SQN=%4d, RQ=%.1f%%, EQ=%.1f%%>",
		m.tq, m.latestSQN,
		float64(m.rqWindow.countHits(batTQMaxValue))/batLocalWindowSize,
		float64(m.eqWindow.countHits(batTQMaxValue))/batLocalWindowSize)
}
