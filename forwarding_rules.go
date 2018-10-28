package main

import "time"

// processAndForward contains the logic governing when and how to forward an OGM packet.
// It also calls the appropriate metric updating functions.
func (b *Batman) processAndForward(ogm OGM) {

	// Facts for Deciding Case Statement //
	_, sentByNeighbor := b.neighbors[ogm.Sender] // The OGM was sent by one of our known neighbors
	_, viaKnownLink := b.neighbors[ogm.Sender]   // The OGM sent by a neighbor's known link address
	//ToDo(Sean): Eventually update viaKnownLink to track link-address spesific information to allow multi-interface

	// General Facts //
	_, knownNode := b.nodes[ogm.Origin] // An OGM from this node has been seen before.

	// Possible Routing Cases //
	switch {
	// Expired TTL Case:
	case ogm.TTL <= 0:
		// Do nothing

	// My OGM Echo Case:
	case ogm.Origin == b.id && ogm.PrevSender == b.id && sentByNeighbor && viaKnownLink:
		// My OGM has been echoed by a known neighbor via a known link, and so may be used to
		// update my estimate of link quality.
		// I shall NOT rebroadcast this OGM.

		// ToDo(Sean): Add SQN check

		// Useful Facts //
		directLink := ogm.RxAddr == ogm.PrevAddr // OGM send and received on same link; EQ and RQ are compatible for estimating TQ.
		var directIP ipAddr
		if directLink {
			directIP = ogm.TxAddr
		}

		// Update Metrics //
		b.neighbors[ogm.Sender].markEcho(directIP, ogm.SQN, time.Now()) // ToDo(Sean): Add timestamp to OGM using actual received time.

	// Neighbor OGM Case:
	case ogm.Sender == ogm.Origin && ogm.Origin != b.id:
		// The OGM is from a neighbor (1-hop link). It informs me of the node's existence and will
		// be used to update my estimate of our link quality (RQ).
		// I shall rebroadcast this OGM.

		// Useful Facts //
		_, knownNeighbor := b.neighbors[ogm.Origin] // The node is already a known neighbor.
		// ToDo(Sean): Handle checking and updating based on known/new link
		// knownLink := false
		// if knownNeighbor {
		// 	_, knownLink = b.neighbors[ogm.Origin][ogm.TxAddr] // The current link has been seen before.
		// }

		// Update Metrics //
		if !knownNode {
			b.nodes[ogm.Origin] = newRouteTracker()
		}
		if !knownNeighbor {
			b.neighbors[ogm.Origin] = newNodeLinkMap()
		}
		b.neighbors[ogm.Origin].markReceive(ogm.TxAddr, ogm.SQN, time.Now())     // Perform link metric update
		b.nodes[ogm.Origin].update(ogm.TxAddr, ogm.SQN, ogm.Quality, time.Now()) // Update next-hop node data

		// Rebroadcast //
		b.rebroadcast(ogm) // Always rebroadcast a neighbor OGM

	// Distant OGM Case:
	case ogm.Sender != ogm.Origin && ogm.Origin != b.id && ogm.Sender != b.id && sentByNeighbor && viaKnownLink:
		// The OGM has been forwarded to me via one or more intermediate nodes. It informs me of
		// the node's existence, and the sender is a possible next hop for packets destined for
		// this origin. Note that the origin may also be a a direct neighbor, but that this OGM
		// packet has arrived via a multi-hop route. We only process a distant OGM if the sender
		// is a known neighbor.
		// I might rebroadcast this OGM.

		// ToDo(Sean): Critical: Finish writing distant OGM case forwarding logic

		// Useful Facts //
		// bestHop, knownRoute := b.routingTable[ogm.Origin]
		// // from_best_route # We only forward distant OGMs
		// //                 # if they arrived to us via our
		// //                 # best next hop route back to
		// //                 # the origin.
		// if knownRoute && ogm.TxAddr == bestHop.ip {
		// 	fromBestRoute := true
		// } else {
		// 	fromBestRoute := false
		// }
		// potentialBroadcastLoop := ogm.PrevSender == b.id // We have already broadcast this OGM in the recent past.

		// Update Metrics //

		// Rebroadcast //

	// Do Nothging Case:
	default:
		// Do nothing

	}

}
