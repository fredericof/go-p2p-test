package p2p

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/flynn/noise"
)

type Peer struct {
	conn net.Conn
	kp   noise.DHKey
}

func (p *Peer) send(b []byte) error {
	_, err := p.conn.Write(b)
	return err
}

type ServerConfig struct {
	ListenAddr string
}

type Message struct {
	Payload io.Reader
	From    net.Addr
}

type Server struct {
	ServerConfig

	handler  Handler
	listener net.Listener
	mu       sync.RWMutex
	peers    map[net.Addr]*Peer
	addPeer  chan *Peer
	delPeer  chan *Peer
	msgCh    chan *Message
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		handler:      &DefaultHandler{},
		ServerConfig: cfg,
		peers:        make(map[net.Addr]*Peer),
		addPeer:      make(chan *Peer),
		msgCh:        make(chan *Message),
	}
}

func (s *Server) Start() {
	go s.loop()

	if err := s.listen(); err != nil {
		panic(err)
	}

	fmt.Printf("game server running on port %s\n", s.ListenAddr)

	//s.acceptLoop()
}

func (s *Server) handleConn(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}

		s.msgCh <- &Message{
			From:    conn.RemoteAddr(),
			Payload: bytes.NewReader(buf[:n]),
		}
	}
}

func (s *Server) AcceptLoop() {
	buffer := make([]byte, 1024)

	for {
		conn, err := s.listener.Accept()
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
			Initiator:     false,
			StaticKeypair: kp,
		}
		hs, err := noise.NewHandshakeState(noiseConfig)
		if err != nil {
			panic(err)
		}

		// read bytes size from header
		var size uint16
		err = binary.Read(conn, binary.BigEndian, &size)
		if err != nil {
			return
		}

		if _, err = conn.Read(buffer); err != nil {
			panic(err)
		}

		_, _, _, err = hs.ReadMessage(nil, buffer)
		if err != nil {
			return
		}

		fmt.Println("<<")
		fmt.Printf("%v\n", buffer)

		// handshake done
		peer := &Peer{
			conn: conn,
			kp:   kp,
		}
		s.addPeer <- peer

		go s.handleConn(conn)
	}
}

func (s *Server) listen() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}

	s.listener = ln

	return nil
}

func (s *Server) loop() {
	for {
		select {
		case peer := <-s.delPeer:
			addr := peer.conn.RemoteAddr()
			delete(s.peers, addr)
			fmt.Printf("peer disconnected %s\n", peer.conn.RemoteAddr())
		case peer := <-s.addPeer:
			s.peers[peer.conn.RemoteAddr()] = peer
			fmt.Printf("new peer connected %s\n", peer.conn.RemoteAddr())
		case msg := <-s.msgCh:
			if err := s.handler.HandleMessage(msg); err != nil {
				panic(err)
			}

		}
	}
}
