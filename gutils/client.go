package gutils

import (
	"crypto/tls"
	"crypto/x509"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"
)

//ClientConnection -TODO-
type ClientConnection struct {
	address       string
	config        *tls.Config
	Client        *rpc.Client
	Connection    *tls.Conn
	autoReconnect bool
}

//ClientMaxErorrs -TODO-
const clientMaxErorrs = 5

//InitClientTLS TODO-
func InitClientTLS(address string, certCA *x509.Certificate, certPair tls.Certificate, serverName string, autoReconnect bool) (*ClientConnection, error) {
	var c ClientConnection
	var err error

	c.address = address
	c.config, err = InitTLS(certCA, certPair, serverName)
	if err != nil {
		return nil, err
	}

	c.autoReconnect = autoReconnect

	return &c, nil
}

//Connect -TODO-
func (c *ClientConnection) Connect() error {
	var err error

	for i := 0; i < clientMaxErorrs; i++ {
		c.Connection, err = tls.Dial("tcp", c.address, c.config)
		if err != nil {
			if !c.autoReconnect {
				return err
			}

			fmtPutLogf(TagTLS, "Dial(%s) error(%d of %d): %s", c.address, i, clientMaxErorrs, err.Error())
			time.Sleep(5 * time.Second)
		} else {
			c.Client = jsonrpc.NewClient(c.Connection)

			return nil
		}
	}

	return err
}

//Call -TODO-
func (c *ClientConnection) Call(serviceMethod string, args interface{}, reply interface{}) error {
	var err error

	if (c.Client == nil) || (c.Connection == nil) {
		err := c.Connect()
		if err != nil {
			return err
		}
	}

	err = c.Client.Call(serviceMethod, args, &reply)
	if err != nil {
		if RemoteLog != nil {
			fmtPutLogf(TagRPCERROR, "rpc call (%s, %s) failed: %s %T", c.address, serviceMethod, err.Error(), err)
		}

		if _, ok := err.(rpc.ServerError); ok {
			return err
		}

		c.Client.Close()
		c.Connection.Close()

		err := c.Connect()
		if err != nil {
			return err
		}

		return c.Client.Call(serviceMethod, args, &reply)
	}

	return nil
}

//Close -TODO-
func (c *ClientConnection) Close() {
	if c.Client != nil {
		c.Client.Close()
	}

	if c.Connection != nil {
		c.Connection.Close()
	}
}
