package main

import (
	"github.com/frederico/concurrency/p2p"
)

func main() {
	cfg := p2p.ServerConfig{
		ListenAddr: ":3000",
	}
	server := p2p.NewServer(cfg)

	server.Start()
}
