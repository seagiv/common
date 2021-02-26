package gutils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
)

const maxConnections = 128

//Receiver -TODO-
type Receiver interface {
	OnConnect(commonName string, remoteAddress net.Addr) []interface{}
	OnDisconnect()
}

//ServerConnection -TODO-
type ServerConnection struct {
	ID int

	Server  *rpc.Server
	Handler Receiver
}

//Listener -TODO-
type Listener struct {
	sync.Mutex

	address string

	currentID int

	config   *tls.Config
	Listener net.Listener

	clients map[int]*ServerConnection
}

//InitServerTLS -TODO-
func InitServerTLS(address string, certCA *x509.Certificate, certTLS tls.Certificate) (*Listener, error) {
	var l Listener

	l.address = address

	l.clients = make(map[int]*ServerConnection)

	var err error

	l.config, err = InitTLS(certCA, certTLS, "")
	if err != nil {
		return nil, err
	}

	l.Listener, err = tls.Listen("tcp", l.address, l.config)
	if err != nil {
		return nil, err
	}

	return &l, nil
}

//InitServerPlain -TODO-
func InitServerPlain(address string) (*Listener, error) {
	var l Listener

	l.address = address

	l.clients = make(map[int]*ServerConnection)

	var err error

	l.Listener, err = net.Listen("tcp", l.address)
	if err != nil {
		return nil, err
	}

	return &l, nil
}

//GetCommonNameTLS -TODO-
func GetCommonNameTLS(conn *tls.Conn) string {
	state := conn.ConnectionState()

	for _, cert := range state.PeerCertificates {
		return cert.Subject.CommonName
	}

	return "-"
}

//GetUniqID -TODO-
func (l *Listener) GetUniqID() int {
	var id int

	l.Lock()

	id = l.currentID

	l.currentID++

	l.Unlock()

	return id
}

//NewConnection -TODO-
func (l *Listener) NewConnection(r Receiver) *ServerConnection {
	l.Lock()

	var sc ServerConnection

	sc.ID = l.currentID

	l.clients[sc.ID] = &sc

	sc.Handler = r

	l.currentID++

	l.Unlock()

	return &sc
}

//Delete -TODO-
func (l *Listener) Delete(id int) {
	l.Lock()

	delete(l.clients, id)

	l.Unlock()
}

//AcceptConnect -TODO-
func (l *Listener) AcceptConnect(conn net.Conn, r Receiver) error {
	c := l.NewConnection(r)

	if len(l.clients) > maxConnections {
		conn.Close()
		l.Delete(c.ID)

		return FormatErrorS("AcceptConnect", "(%d) too many concurrent connections - %d", c.ID, len(l.clients))
	}

	fmtPutLogf(TagTLS, "(%d): connect from %s, concurrent connections - %d", c.ID, conn.RemoteAddr(), len(l.clients))

	if len(l.clients) > 4 {
		fmt.Printf("(%d): !WARNING! connect from %s, concurrent connections - %d\n", c.ID, conn.RemoteAddr(), len(l.clients))
	}

	tlscon, ok := conn.(*tls.Conn)
	defer tlscon.Close()
	defer conn.Close()
	defer l.Delete(c.ID)

	var commonName string

	if ok {
		state := tlscon.ConnectionState()

		if !state.HandshakeComplete {
			//fmtPutLogf(tagRPCTLS, "(%d): handshaking ...", c.ID)

			err := tlscon.Handshake()
			if err != nil {
				return fmt.Errorf("tls(%d, %s): handshake %s", c.ID, conn.RemoteAddr(), err.Error())
			}
		}

		commonName = GetCommonNameTLS(tlscon)
	}

	fmtPutLogf(TagTLS, "(%d): handshake complete.", c.ID)

	s := c.Handler.OnConnect(commonName, conn.RemoteAddr())

	if len(s) == 0 {
		return fmt.Errorf("tls(%d): no servers allowed for %s", c.ID, commonName)
	}

	c.Server = rpc.NewServer()

	for i := 0; i < len(s); i++ {
		c.Server.Register(s[i])
	}

	c.Server.ServeCodec(jsonrpc.NewServerCodec(conn))

	c.Handler.OnDisconnect()

	fmtPutLogf(TagTLS, "(%d): closed %s", c.ID, conn.RemoteAddr())

	return nil
}
