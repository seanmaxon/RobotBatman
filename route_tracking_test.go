package main

import (
	"testing"
	"time"
)

// func TestRouteTracker(t *testing.T) {
// 	var rm *routingMetric

// 	rt = newRouteTracker()
// 	rt.nextHops['10.0.0.3'] =
// }

func TestIpAddrRaw(t *testing.T) {
	var ip ipAddr = "10.1.6.3"

	r := ip.raw()

	s := ipAddrFromBytes(r)

	if ip != s {
		t.Error("ipAddr error: raw conversion failure:", ip, r, s)
	}
}

func TestRouteTracker(t *testing.T) {
	rt := newRouteTracker()

	rt.update(ipAddr("192.168.1.1"), newDefaultSQN(5), 200, time.Now())
	rt.update(ipAddr("192.168.1.1"), newDefaultSQN(6), 200, time.Now())

	if !rt.latestSQN.equalTo(newDefaultSQN(6)) {
		t.Error("RouteTracker update error: Latest SQN:", rt.String())
	}

	// ToDo(Sean): Add test cases for remaining update RouteTracker tasks.
}
