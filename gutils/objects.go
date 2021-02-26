package gutils

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"
)

var marshal = func(v interface{}) (io.Reader, error) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

var unmarshal = func(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

var lock sync.Mutex

//SaveObject -TODO-
func SaveObject(fileName string, v interface{}) error {
	lock.Lock()
	defer lock.Unlock()

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := marshal(v)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, r)

	return err
}

//LoadObject -TODO-
func LoadObject(fileName string, v interface{}) error {
	lock.Lock()
	defer lock.Unlock()

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	return unmarshal(f, v)
}
