package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

// The Batman struct holds this node instance's state information.
type Batman struct {
	// Node identity information
	id  nodeID
	sqn sqn

	// Network interfaces
	localAddrs     map[ipAddr]bool
	broadcastAddrs map[ipAddr]net.IP
	udpAddrs       map[ipAddr]*net.UDPAddr
	udpConns       map[ipAddr]*net.UDPConn

	// Internal queues and channels
	stop        chan bool
	outboundOGM chan OGM
	inboundOGM  chan OGM

	// Primary data structures
	nodes     map[nodeID]*routeTracker // replace with globalNodesMap
	neighbors map[nodeID]nodeLinksMap

	// Computed data structures
	routingTable routingTableMap

	// ToDo(Sean): Handle system routing table updater through dependancy injection
}

// New initializes a new Batman node. You only need one.
func New() Batman {
	return Batman{
		id:  "L1",
		sqn: newDefaultSQN(0),

		outboundOGM: make(chan OGM),
		inboundOGM:  make(chan OGM),

		nodes:     make(map[nodeID]*routeTracker),
		neighbors: make(map[nodeID]nodeLinksMap),
	}
	// ToDo(Sean): Flesh out Batman New() function.
}

func (b *Batman) advanceSQN() {
	b.sqn.increment()

	// ToDo(Sean): Update all metrics that use own SQN
}

// advertiseOGM should be called periodically.
// It is where this node's own OGMs are created and queued for broadcast.
func (b *Batman) advertiseOGM() {
	// Advance sequence number
	b.advanceSQN()

	// Create and send OGM
	ogm := OGM{
		Origin:     b.id,
		Sender:     b.id,
		TxAddr:     "", // gets populated on broadcast; ToDo(Sean): Remove TxAddr from raw OGM definition and get from network interface on read
		PrevSender: "",
		PrevAddr:   "",
		SQN:        b.sqn,
		TTL:        batTTL,
		Quality:    batTQMaxValue,
	}

	// Queue for broadcast
	b.outboundOGM <- ogm

	// Update all link quality estimates using new SQN
	b.updateLinkEstimates()

	b.rebuildRoutingTable()
}

func (b *Batman) updateLinkEstimates() {
	// ToDo(Sean): Flesh out updateLinkEstimates() function.
}

func (b *Batman) rebuildRoutingTable() {
	// ToDo(Sean): Flesh out rebuildRoutingTable() function.
}

// startOGMBundler starts a go-routine for grabbing OGMs off the outbound ogm queue,
// bundling them up, and passing them off onto the outbound bundle queue.
func (b *Batman) startOGMBundler() <-chan []RawOGM {
	outboundBundle := make(chan []RawOGM)
	go func(outboundBundle chan<- []RawOGM) {
		defer close(outboundBundle)

		// initialize and stop a timer so it's ready for later use
		timeout := time.NewTimer(1 * time.Second)
		if !timeout.Stop() {
			<-timeout.C
		}
		timerRunning := false

		// sit listening for outbound OGMs and send them out in bundles
		bundle := make([]RawOGM, 0, batMaxBundleSize)
		for {
			select {
			// ToDo(Sean): Check this for possible race condition on timer... what if it triggers between the if and the stop?
			case ogm := <-b.outboundOGM:
				bundle = append(bundle, ogm.Pack())
				if len(bundle) >= batMaxBundleSize {
					timerRunning = false
					if !timeout.Stop() {
						<-timeout.C
					}
					// flush bundle to output
					outboundBundle <- bundle
					bundle = bundle[:0]
				} else if !timerRunning {
					timerRunning = true
					timeout.Reset(batMaxBundleDelay * time.Millisecond)
				}
			case <-timeout.C:
				timerRunning = false
				// flush bundle to output
				outboundBundle <- bundle
				bundle = bundle[:0]
			case <-b.stop:
				return
			}
		}
	}(outboundBundle)

	return outboundBundle
}

// startNetworkListeners starts one goroutine for each network interface.
// Each goroutine loops making blocking network read calls, receiving
// incoming UDP packets of OGM bundles, parsing OGMs, and feeding them
// one at a time to the Batman node's inboundOGM channel.
func (b *Batman) startNetworkListeners() error {
	for _, conn := range b.udpConns {
		receive := ogmReaderFactory(conn, b.localAddrs)
		go func(receive func() ([]OGM, error)) {
			for {
				select {
				case <-b.stop:
					return
				default:
					if ogms, err := receive(); err == nil {
						for _, ogm := range ogms {
							// ToDo(Sean): Populate TxAddr field with sender's IP (requires modifying receive)
							b.inboundOGM <- ogm
						}
					}
				}
			}
		}(receive)
	}
	return nil
}

// startNetworkBroadcasters is responsible for putting OGM bunles on the wire.
// It spawns one goroutine per network interface, and one more to replicate the
// outbound OGM bundles for each network interface goroutine.
func (b *Batman) startNetworkBroadcasters(outboundBundle <-chan []RawOGM) error {
	if len(b.udpConns) < 1 {
		return errors.New("startNetworkBroadcasters: cannot start: no UDP connections")
	}

	// ToDo(Sean): Eventually, add mechanism for adding and and removing interfaces at runtime?

	// spin up a broadcaster for each network interface
	bcastChans := make([](chan []RawOGM), 0, len(b.udpConns))
	for ip, conn := range b.udpConns {
		perConChan := make(chan []RawOGM)
		bcastChans = append(bcastChans, perConChan)

		broadcast := broadcasterFactory(conn, ip, b.broadcastAddrs[ip])

		go func(perConChan chan []RawOGM, txAddr [4]byte, broadcast func([]byte) error) {
			msg := make([]byte, batSafePacketSize)
			customBundle := make([]RawOGM, 0, batMaxBundleSize)

			for bundle := range perConChan {
				customBundle = customBundle[:0]
				msg = msg[:0]
				// customize each OGM in bundle with correct txAddr
				for i := range bundle {
					// customBundle[i] = bundle[i]
					customBundle = append(customBundle, bundle[i])
					customBundle[i].TxAddr = txAddr
				}
				packOGMs(&msg, customBundle)
				_ = broadcast(msg) // ToDo(Sean): Maybe log err message?
			}
		}(perConChan, ip.raw(), broadcast)
	}

	// replicate an outbound OGM bundle for all interfaces
	go func() {
		for bundle := range outboundBundle {
			for _, c := range bcastChans {
				c <- bundle
			}
		}
		// trigger shutdown of all broadcast gorutines
		for _, c := range bcastChans {
			close(c)
		}
		return
	}()
	return nil
}

// Run starts the Batman instance.
func (b *Batman) Run() (err error) {
	b.stop = make(chan bool)

	// Network services //

	// Find network interfaces and create sockets
	b.localAddrs, b.broadcastAddrs = localAndBroadcastAddresses()
	if b.udpAddrs, err = resolveUDPAddresses(b.localAddrs, batUDPPortStr); err != nil {
		fmt.Println(err)
		return
	}
	if b.udpConns, err = openSockets(b.udpAddrs); err != nil {
		fmt.Println(err)
		return
	}
	defer closeSockets(b.udpConns)

	// Start Services: Bundle, Listen, Broadcast
	outboundBundle := b.startOGMBundler()
	if err = b.startNetworkBroadcasters(outboundBundle); err != nil {
		fmt.Println(err)
		return
	}
	if err = b.startNetworkListeners(); err != nil {
		fmt.Println(err)
		return
	}

	// BATMAN services //

	// Start self ogm advertiser
	go func() {
		advertTimer := time.NewTimer(batOGMInterval * time.Second)
		for {
			select {
			case <-b.stop:
				return
			case <-advertTimer.C:
				b.advertiseOGM()
				advertTimer.Reset(batOGMInterval*time.Second + time.Duration(rand.Int63n(batOGMJitter))*time.Millisecond)
			}
		}
	}()

	// Start OGM handler
	for {
		select {
		case <-b.stop:
			return
		case ogm := <-b.inboundOGM:
			b.processAndForward(ogm) // apply forwarding rules and update metrics
		}
	}
}

func (b *Batman) rebroadcast(ogm OGM) {
	if ogm.TTL < 1 {
		log.Println("rebroadcast() called on OGM with <1 TTL")
		return
	}
	var lossFactor byte = 1 // TODO(Sean): Replace this with correct hop penalty!

	ogm.PrevSender = ogm.Sender
	ogm.PrevAddr = ogm.TxAddr
	ogm.Sender = b.id
	ogm.TTL -= 1
	ogm.Quality = ogm.Quality * lossFactor
	b.outboundOGM <- ogm
}
