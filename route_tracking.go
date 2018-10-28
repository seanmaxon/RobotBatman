package main

import (
	"bytes"
	"fmt"
	"time"
)

// The routingTableMap holds the best next hop for each reachable node.
type routingTableMap map[nodeID]bestNextHop

// func (table routingTableMap) String() string {
// 	return table.String()
// }

// ToDo(Sean): Critical: Write routingTableMap update method!

// ToDo(Sean): Write IP routing table update/sync method for routingTableMap

// bestNextHop stores address and quality information for the routing path that
// begins by following this link to some particular node.
type bestNextHop struct {
	ip      ipAddr
	quality byte
	age     time.Duration
}

// A routeTracker is used for tracking *all* possible routes (next hops)
// and their scalar metric values (distance vectors) for a single destination.
//
// Next hops are indexed by address, which in one implementation might be
// IP address. This is because nodes are allowed to have multiple
// addresses, some of which may be reachable in one hop (neighbors) and
// others which are not, for the same node.
type routeTracker struct {
	nextHops  map[ipAddr]*hop
	latestSQN sqn
}

// identical to pathData
type hop struct {
	quality  byte // A hop's self-reported quality, not considering additional local link cost
	sqn      sqn
	lastSeen time.Time
	// id       nodeID
}

func newRouteTracker() *routeTracker {
	nh := make(map[ipAddr]*hop)
	return &routeTracker{nextHops: nh, latestSQN: sqn{}}
}

func (r *routeTracker) String() string {
	var buf bytes.Buffer
	for key, v := range r.nextHops {
		fmt.Fprintf(&buf, "%s: Quality=%d, SQN=%v, Age=%d, ", key, v.quality, v.sqn, time.Since(v.lastSeen))
	}
	return fmt.Sprintf("{routeTracker: SQN=%s, %s}", r.latestSQN.String(), buf.String())
}

func (r *routeTracker) update(ip ipAddr, sqn sqn, quality byte, when time.Time) {
	if _, ok := r.nextHops[ip]; !ok {
		r.nextHops[ip] = &hop{quality, sqn, when}
	}
	if sqn.greaterThan(r.latestSQN) {
		r.latestSQN = sqn
	}
	hopPtr := r.nextHops[ip]
	if sqn.greaterThan(hopPtr.sqn) || sqn.equalTo(hopPtr.sqn) {
		hopPtr.quality = quality
		hopPtr.sqn = sqn
		hopPtr.lastSeen = when
	}
	// ToDo(Sean): Add check on lastSeen to keep newest when equal
}
