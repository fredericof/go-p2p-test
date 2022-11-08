package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"net"

	"github.com/flynn/noise"
	"github.com/frederico/concurrency/p2p"
)

func main() {
	// Run Server
	cfg := p2p.ServerConfig{
		ListenAddr: ":3000",
	}
	server := p2p.NewServer(cfg)
	server.Start()

	// Dial
	conn, err := net.Dial("tcp", "localhost:3000")
	log.Printf("dialing to %s", "localhost:3000")

	if err != nil {
		panic(err)
	}

	// creating keypair and new handshake
	kp, err := noise.DH25519.GenerateKeypair(rand.Reader)
	if err != nil {
		panic(err)
	}

	noiseConfig := noise.Config{
		CipherSuite:   noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s),
		Pattern:       noise.HandshakeXX,
		Initiator:     true,
		StaticKeypair: kp,
	}
	hs, err := noise.NewHandshakeState(noiseConfig)
	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 1024)
	var msg []byte
	msg, _, _, err = hs.WriteMessage(buffer[:0], nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(">>")
	fmt.Printf("%v\n", msg)

	// 2 bytes of header size
	binary.Write(conn, binary.BigEndian, uint16(len(msg)))
	if _, err = conn.Write(msg); err != nil {
		return
	}

	// Server Acceptio new peers connections
	server.AcceptLoop()
}
