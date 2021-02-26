package main

import (
	"encoding/hex"
	"fmt"

	"github.com/seagiv/common/gutils"
)

var masterKey []byte

//Manager -
type Manager struct {
}

func (m *Manager) onConnect(commonName string) bool {
	return true
}

func (m *Manager) onDisconnect() {
}

//GetKey -
func (m *Manager) GetKey(args int, reply *string) error {
	*reply = hex.EncodeToString(masterKey)

	fmt.Println(gutils.FormatInfoS("", "storage key sent"))

	return nil
}
