package main

import (
	"net"
)

type rpcServer interface {
	onConnect(commonName string) bool
	onDisconnect()
}

//Server interfaces for rpc servers
type Server struct {
	servers []interface{}
}

func (s *Server) initServer(server interface{}, commonName string) {
	if server.(rpcServer).onConnect(commonName) {
		s.servers = append(s.servers, server)
	}
}

//OnConnect check security properties and create rpc interfaces for allowed objects
func (s *Server) OnConnect(commonName string, remoteAddress net.Addr) []interface{} {
	s.initServer(&Manager{}, commonName)

	return s.servers
}

//OnDisconnect proceed disconnect routines
func (s *Server) OnDisconnect() {
	for _, server := range s.servers {
		server.(rpcServer).onDisconnect()
	}
}
