package main

// Do I need this? +build linux

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// var listenV4, _ = net.ResolveUDPAddr("udp4", "0.0.0.0:30703")

// bcastAddr returns the UDP broadcast address for the IP and subnet in the IPNet.
func bcastAddr(ipnet *net.IPNet) (net.IP, error) {
	ip := ipnet.IP.To4()
	mask := ipnet.Mask

	if ip == nil {
		return nil, fmt.Errorf("bcastAddr: IPv4 broadcast address not defined for given IPNet: %v", ipnet)
	}

	subnet := make([]byte, 4)
	for i := range mask {
		subnet[i] = ^mask[i]
	}
	bcast := make([]byte, 4)
	for i := range ip {
		bcast[i] = ip[i] | subnet[i]
	}
	return net.IP(bcast), nil
}

// localAndBroadcastAddresses returns a map of loopback addresses to ignore,
// and a mapping of local interface addresses to their UDP broadcast addresses.
// Note: All addresses are IPv4 addresses.
func localAndBroadcastAddresses() (localAddrs map[ipAddr]bool, broadcastAddrs map[ipAddr]net.IP) {
	// Compile list of local addresses
	localAddrs = make(map[ipAddr]bool) // list of own (non-loopback) IPv4 addresses
	broadcastAddrs = make(map[ipAddr]net.IP)
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
			// fmt.Println("Iface error:", err)
		}
		for _, addr := range addrs {
			// fmt.Printf("Addr: %T, %v\n", addr, addr)
			switch t := addr.(type) {
			case *net.IPNet:
				// fmt.Printf("IP: %v\n", t.IP.String())

				if (iface.Flags & net.FlagLoopback) != 0 {
					// fmt.Println("Loopback")
					continue
				} else if (iface.Flags & net.FlagUp) == 0 {
					// interface not up
					continue
				} else if (iface.Flags & net.FlagBroadcast) == 0 {
					// doesn't support broadcast
					continue
				} else if t.IP.To4() != nil {
					// is an IPv4 address

					if bcastIP, err := bcastAddr(t); err == nil {
						// fmt.Println("The UDP broadcast address for", t, "is", bcastIP)
						localAddrs[ipAddr(t.IP.String())] = true
						broadcastAddrs[ipAddr(t.IP.String())] = bcastIP
						fmt.Println("address found: ", t.IP.To4())
					}

				}
			default:
				// NOOP
			}
		}
	}
	if len(broadcastAddrs) < 1 {
		panic("No broadcast interfaces found!")
	}
	return
}

// resolveUDPAddresses takes a slice of local IPv4 addresses together with a
// port number and returns UDP addresses as needed for opening UDP sockets.
func resolveUDPAddresses(localAddrs map[ipAddr]bool, port string) (map[ipAddr]*net.UDPAddr, error) {
	var err error
	var errstrings []string
	udpAddrs := make(map[ipAddr]*net.UDPAddr)
	for addr := range localAddrs {
		udpAddr, err := net.ResolveUDPAddr("udp4", string(addr)+":"+port)
		if err != nil {
			errstrings = append(errstrings, err.Error())
		} else {
			udpAddrs[addr] = udpAddr
		}
	}
	if errstrings != nil {
		err = errors.New(strings.Join(errstrings, ",'"))
	}
	return udpAddrs, err
}

// openSockets opens network connetions for the resolved list of UDP addrs.
func openSockets(udpAddrs map[ipAddr]*net.UDPAddr) (map[ipAddr]*net.UDPConn, error) {
	var err error
	var errstrings []string
	sockets := make(map[ipAddr]*net.UDPConn)
	for k, v := range udpAddrs {
		// Create UDP socket
		conn, err := net.ListenUDP("udp4", v)
		if err != nil {
			errstrings = append(errstrings, err.Error())
		} else {
			sockets[k] = conn
		}
	}
	if errstrings != nil {
		err = errors.New(strings.Join(errstrings, ",'"))
	}
	return sockets, err
}

func closeSockets(udpConns map[ipAddr]*net.UDPConn) (err error) {
	var errstrings []string
	for _, conn := range udpConns {
		if err := conn.Close(); err != nil {
			errstrings = append(errstrings, err.Error())
		}
	}
	if errstrings != nil {
		err = errors.New(strings.Join(errstrings, ","))
	}
	return
}

// Call ogmReaderFactory to get a function that will (blocking, 30s) read OGMs
// from the UDP connection, and once available return them as an OGM slice.
//
// Example usage:
//    readOGMs := ogmReaderFactory(conn, localAddrs)
//    for {
//    	   if ogms, err = readOGMs(); err == nil {
//      	    for _, ogm := range ogms {
//    	  	    fmt.Println(ogm)
//        }
//    }
//
func ogmReaderFactory(conn *net.UDPConn, ignoreAddrs map[ipAddr]bool) func() ([]OGM, error) {
	data := make([]byte, 4096)

	return func() ([]OGM, error) {
		// Note: It is CRITCAL that conn MUST have a read deadline set.
		conn.SetReadDeadline(time.Now().Add(time.Second * 30))
		n, addr, err := conn.ReadFromUDP(data)
		if err != nil {
			return nil, err
			// we expect an error if we close the conn (i.e., Batman instance stopped)
			// or if the read times out after 30 seconds with no OGMs
		}
		// Ignore own transmissions
		if ignoreAddrs[ipAddr(addr.IP.String())] {
			return nil, nil
		}
		// ToDo(Sean): Store addr from read and use it in place of txAddr in ogm.

		rxAddress := conn.LocalAddr()
		return parseOGMs(data[:n], ipAddr(rxAddress.String()))
	}
}

// broadcasterFactory returns a function that will send a byte slice as a
// UDP broadcast packet on the given connection to the given address.
func broadcasterFactory(conn *net.UDPConn, linkIP ipAddr, broadcastAddrs net.IP) func([]byte) error {

	broadcastAddr := &net.UDPAddr{
		IP:   broadcastAddrs,
		Port: batUDPPortInt,
	}

	return func(pkt []byte) error {
		n, err := conn.WriteToUDP(pkt, broadcastAddr)
		if err != nil {
			return err
		} else if n != len(pkt) {
			return fmt.Errorf("broadcaster: WriteToUDP: wrong number of bytes sent: len(pkt)=%v, n=%v", len(pkt), n)
		}
		return nil
	}
}
