package gutils

import (
	"bytes"
	"errors"
	"fmt"

	"encoding/gob"
	"encoding/json"

	"io/ioutil"
	"sync"
)

//FlagPreserveKey -TODO-
const FlagPreserveKey = 0x00000001

//Storage -TODO-
type Storage struct {
	sync.Mutex

	flags int64

	fileName  string
	masterKey []byte

	items map[string][]byte
}

//OpenStorage -
func OpenStorage(fileName string, masterKey []byte, flags int64) (*Storage, error) {
	s := &Storage{fileName: fileName, masterKey: masterKey, items: make(map[string][]byte)}

	if (flags & FlagPreserveKey) == 0 {
		defer s.WipeKey()
	}

	return s, s.load()
}

//CreateStorage -
func CreateStorage(fileName string, masterKey []byte) (*Storage, error) {
	s := &Storage{fileName: fileName, masterKey: masterKey, items: make(map[string][]byte)}

	return s, s.save()
}

func (s *Storage) encrypt() ([]byte, error) {
	decBuffer := new(bytes.Buffer)

	encoder := gob.NewEncoder(decBuffer)
	encoder.Encode(&s.items)

	encBuffer, err := EncryptGCM(decBuffer.Bytes(), s.masterKey)
	if err != nil {
		return nil, err
	}

	return encBuffer, nil
}

func (s *Storage) decrypt(encBuffer []byte) error {
	d, err := DecryptGCM(encBuffer, s.masterKey)
	if err != nil {
		return err
	}

	decBuffer := bytes.NewBuffer(d)

	decoder := gob.NewDecoder(decBuffer)
	decoder.Decode(&s.items)

	return nil
}

func (s *Storage) load() error {
	var err error

	if s.masterKey == nil {
		return FormatErrorS("", "masterKey not set")
	}

	encBuffer, err := ioutil.ReadFile(s.fileName)
	if err != nil {
		return err
	}

	return s.decrypt(encBuffer)
}

//Save -
func (s *Storage) save() error {
	var err error

	if s.masterKey == nil {
		return FormatErrorS("", "masterKey not set")
	}

	buffer, err := s.encrypt()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(s.fileName, buffer, 0644)
}

//WipeKey -
func (s *Storage) WipeKey() {
	s.masterKey = nil
}

//Set -
func (s *Storage) Set(name string, data []byte) error {
	s.Lock()
	defer s.Unlock()

	if s.masterKey == nil {
		return FormatErrorS("", "masterKey not set")
	}

	if len(name) > 255 {
		return FormatErrorS("NameLength", "name too long")
	}

	if len(data) > 65535 {
		return FormatErrorS("DataLength", "data too large")
	}

	if len(s.items) > 65523 {
		return FormatErrorS("ItemsLength", "too many items")
	}

	s.items[name] = data

	return s.save()
}

//Delete -
func (s *Storage) Delete(name string) error {
	s.Lock()
	defer s.Unlock()

	if s.masterKey == nil {
		return fmt.Errorf("masterKey not set")
	}

	if s.items[name] == nil {
		return fmt.Errorf("item [%s] not found", name)
	}

	delete(s.items, name)

	return s.save()
}

//Get -
func (s *Storage) Get(name string) ([]byte, bool) {
	s.Lock()
	defer s.Unlock()

	if value, ok := s.items[name]; ok {
		return value, true
	}

	return []byte{}, false
}

//List -TODO-
func (s *Storage) List() {
	s.Lock()
	defer s.Unlock()

	for k, v := range s.items {
		fmt.Printf("[%s]: %d\n", k, len(v))
	}
}

type coinInfo struct {
	Address string `json:"address"`
	Key     string `json:"key"`
}

//GetCoinInfo return address, privKey, ok
func (s *Storage) GetCoinInfo(coinTag string) (string, string, error) {
	var err error

	scid := "Coin." + coinTag + ".JSON"

	keyJSON, ok := s.Get(scid)
	if !ok {
		return "", "", errors.New("coin info not found in storage")
	}

	var ci coinInfo

	err = json.Unmarshal(keyJSON, &ci)
	if err != nil {
		return "", "", err
	}

	return ci.Address, ci.Key, nil
}
