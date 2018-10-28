package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
)

// A RawOGM is BATMAN's routing overhead packet
type RawOGM struct {
	Origin     [4]byte // nodeID of OGM creator
	Sender     [4]byte // nodeID of node that transmitted OGM
	TxAddr     [4]byte // sender interface identifier (ipAddr)
	PrevSender [4]byte // nodeID of previous sender; \x00 if none
	PrevAddr   [4]byte // previous sender interface identifier (ipAddr); \x00 if none
	SQN        uint32  // Sequence Number
	TTL        byte    // Time To Live
	Quality    byte    // TQ metric
}

// Unpack converts a RawOGM to an OGM.
func (ogm *RawOGM) Unpack() OGM {
	return OGM{
		Origin:     nodeIDFromBytes(ogm.Origin),
		Sender:     nodeIDFromBytes(ogm.Sender),
		TxAddr:     ipAddrFromBytes(ogm.TxAddr),
		PrevSender: nodeIDFromBytes(ogm.PrevSender),
		PrevAddr:   ipAddrFromBytes(ogm.PrevAddr),
		SQN:        newDefaultSQN(int(ogm.SQN)),
		TTL:        ogm.TTL,
		Quality:    ogm.Quality,
	}
}

func (ogm *RawOGM) String() string {
	return "{Origin:" + string(ogm.Origin[:]) + ", " +
		"Sender:" + string(ogm.Sender[:]) + ", " +
		"TxAddr:" + net.IP(ogm.TxAddr[:]).String() + ", " +
		"PrevSender:" + string(ogm.PrevSender[:]) + ", " +
		"PrevAddr:" + net.IP(ogm.PrevAddr[:]).String() + ", " +
		"SQN:" + strconv.FormatUint(uint64(ogm.SQN), 10) + ", " +
		"TTL:" + strconv.FormatUint(uint64(ogm.TTL), 10) + ", " +
		"TQ:" + strconv.FormatUint(uint64(ogm.Quality), 10) + "}"
}

func parseOGMs(ogmBundle []byte, addr ipAddr) ([]OGM, error) {
	if len(ogmBundle) < 27 || (len(ogmBundle)-1)%batOGMSize != 0 {
		//panic("malformed ogmBundle")
		return nil, fmt.Errorf("parseOGMs: malformed ogmBundle, ogmBundle=%#v", ogmBundle)
	}
	var output []OGM
	b := bytes.NewBuffer(ogmBundle)
	b.Next(1) // skip count/length byte
	count := int(ogmBundle[0])
	if count*batOGMSize != (len(ogmBundle) - 1) {
		return nil, fmt.Errorf("parseOGMs: invalid count value: count*OGM_SIZE = %v, len(ogmBundle)-1 = %v", count*batOGMSize, len(ogmBundle)-1)
	}
	for n := 0; n < count; n++ {
		ogmRaw := RawOGM{}
		// binary.Read(b, binary.BigEndian, &ogmRaw)
		binary.Read(b, binary.LittleEndian, &ogmRaw)
		ogm := ogmRaw.Unpack()
		ogm.RxAddr = addr
		// ToDo(Sean): Consider adding setting of field for TxAddr??
		output = append(output, ogm)
	}
	return output, nil
}

func packOGMs(buf *[]byte, ogms []RawOGM) error {
	if cap(*buf) < len(ogms)*batOGMSize+1 {
		return errors.New("packOGMs: byte slice too small for OGM bundle size")
	}

	buffer := bytes.NewBuffer((*buf)[:0])

	lengthByte := byte(len(ogms))
	if int(lengthByte) != len(ogms) {
		return errors.New("packOGMs: OGM slice length overflowed one byte")
	}

	buffer.WriteByte(lengthByte)

	for _, ogm := range ogms {
		err := binary.Write(buffer, binary.LittleEndian, ogm)
		if err != nil {
			return fmt.Errorf("packOGMs: %v", err)
		}
	}

	*buf = buffer.Bytes()

	return nil
}

// OGM is an equivalent representation to RawOGM using internal package types.
type OGM struct {
	Origin     nodeID //[4]byte // nodeID of OGM creator
	Sender     nodeID //[4]byte // nodeID of node that transmitted OGM
	TxAddr     ipAddr //[4]byte // sender interface identifier (ipAddr)
	PrevSender nodeID //[4]byte // nodeID of previous sender; \x00 if none
	PrevAddr   ipAddr //[4]byte // previous sender interface identifier (ipAddr); \x00 if none
	SQN        sqn    //uint32
	TTL        byte   //byte
	Quality    byte   //TQ byte

	RxAddr ipAddr // Extra info on Rx interface

	//ToDo(Sean): Rename "TxAddr" to SenderAddr
}

// Pack creates a RawOGM suitable for sending over the wire.
func (s OGM) Pack() RawOGM {
	return RawOGM{
		Origin:     s.Origin.raw(),
		Sender:     s.Sender.raw(),
		TxAddr:     s.TxAddr.raw(),
		PrevSender: s.PrevSender.raw(),
		PrevAddr:   s.PrevAddr.raw(),
		SQN:        s.SQN.raw(),
		TTL:        s.TTL,
		Quality:    s.Quality,
	}
}

// ToDo(Sean): Resolve apparent duplicate of feature/responsibility between startOGMBundler and ogmQueue!
// Probably convert the goroutine-based OGMBundler to an approach based on ogmQueue, for better transparency.
// Currently, the OGMBundler does NOT use the correct format of a bundle, having a leading OGM count byte!

// Internal queue for bundling outbound OGMs
type ogmQueue struct {
	count     int // Number of OGMs in queue
	queueSize int // Number of OGMs that can fit in the queue
	highwater int // Number of OGMs that should trigger send
	ogms      []RawOGM
}

func newOGMQueue(highwater int) *ogmQueue {
	buf := make([]RawOGM, batMaxBundleSize)
	return &ogmQueue{0, batMaxBundleSize, highwater, buf}
}

func (q *ogmQueue) addOGM(ogm RawOGM) (ok, highwater bool) {
	if q.count >= q.queueSize {
		ok = false
		highwater = true
		return
	}
	q.ogms[q.count] = ogm
	q.count++

	if q.count+1 >= q.highwater {
		highwater = true
	}
	ok = true
	return
}

// ToDo(Sean): Finish writing methods for ogmQueue

// func (q *ogmQueue) flush() []byte {
// 	if q.count < 1 {
// 		return nil
// 	}
// 	b := packOGMs(q.ogms)
// 	q.count = 0
// 	q.ogms = q.ogms[:0]
// 	return b
// }

// func (q *ogmQueue) atHighwater() bool {
// 	return q.count >= q.highwater
// }

// A nodeID uniquely identifies a node in the network.
type nodeID string

//  nodeIDFromBytes converts 4 raw bytes (e.g., from an OGM) into ipAddr.
func nodeIDFromBytes(b [4]byte) nodeID {
	bslice := bytes.Trim(b[:], "\x00")
	return nodeID(string(bslice))
}

//  raw converts a nodeID into 4 raw bytes (e.g., for an OGM).
func (id *nodeID) raw() [4]byte {
	b := []byte(*id)
	if len(b) > 4 {
		panic(fmt.Sprint("nodeID.raw(): invalid nodeID:", *id))
	}
	pad := 4 - len(b)
	rawID := [4]byte{}
	for i := range b {
		rawID[pad+i] = b[i]
	}
	return rawID
}

// An ipAddr identifies a particular link.
// A single node may have multiple IP addresses, corresponding to different links.
type ipAddr string

//  ipAddrFromBytes converts 4 raw bytes (e.g., from an OGM) into ipAddr.
func ipAddrFromBytes(b [4]byte) ipAddr {
	return ipAddr(net.IPv4(b[0], b[1], b[2], b[3]).String())
}

// raw converts an ipAddr into 4 raw bytes (e.g., for an OGM).
func (ip *ipAddr) raw() [4]byte {
	netIP := net.ParseIP(string(*ip)).To4()
	var b []byte
	if netIP == nil {
		// panic(fmt.Sprint("ipAddr.raw(): invalid ip:", ip, *ip))
		b = []byte{0, 0, 0, 0}
	} else {
		b = []byte(netIP)
	}
	rawIP := [4]byte{}
	for i := range b {
		rawIP[i] = b[i]
	}
	return rawIP
}
