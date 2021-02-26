package gutils

import (
	"bytes"
	"encoding/gob"
	"errors"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/seagiv/foreign/decimal"
)

//ArgsDST args for rpc calls
type ArgsDST struct {
	CoinTag     string
	AddressFrom string
	AddressTo   string
	PrivateKey  string
	Amount      decimal.Decimal
}

//ReplyVersion reply for Utils.Version
type ReplyVersion struct {
	Project string
	Version string
}

//CoinConfig config structure for coins
type CoinConfig struct {
	Signer          string
	URL             string
	TestTransaction string
}

//IndexOf -TODO-
func IndexOf(element string, data []string) int {
	for i, v := range data {
		if element == v {
			return i
		}
	}
	return -1 //not found.
}

//IsIn -TODO-
func IsIn(element string, data []string) bool {
	return IndexOf(element, data) >= 0
}

//GetCallStack -TODO-
func GetCallStack() string {
	var chain string

	skipLeft := regexp.MustCompile("^(runtime.call|main.main|runtime.goexit)")
	skipAny := regexp.MustCompile("(RemoteLogger|gutils|redcon)")
	skipMain := regexp.MustCompile("^(main.)")

	fpcs := make([]uintptr, 10)

	n := runtime.Callers(3, fpcs)
	if n == 0 {
		return ""
	}

	var lastObject string

	for i := 0; i < n; i++ {
		fun := runtime.FuncForPC(fpcs[i] - 1)
		if fun == nil {
			return "-"
		}

		if skipLeft.MatchString(fun.Name()) {
			break
		}

		if skipAny.MatchString(fun.Name()) {
			continue
		}

		//fmt.Printf("CS: %s\n", fun.Name())

		s := strings.Split(fun.Name(), ".")

		if len(chain) == 0 {
			chain = s[len(s)-1]
		} else {
			chain = s[len(s)-1] + "." + chain
		}

		if len(s) >= 1 {
			lastObject = s[len(s)-2]
		}
	}

	if len(lastObject) != 0 {
		chain = lastObject + "." + chain
	}

	if skipMain.MatchString(chain) {
		chainS := strings.Split(chain, ".")
		chain = strings.Join(chainS[1:], " ")
	}

	if len(chain) == 0 {
		chain = "main"
	}

	return chain
}

var errStructValue = errors.New("value must be non-nil pointer to a struct")

//GetFieldCount -TODO-
func GetFieldCount(v interface{}) (int, error) {
	d := reflect.ValueOf(v)

	if d.Kind() != reflect.Ptr || d.IsNil() {
		return 0, errStructValue
	}
	d = d.Elem()
	if d.Kind() != reflect.Struct {
		return 0, errStructValue
	}

	return d.NumField(), nil
}

//EncodeGob serialize any structured data to gob format
func EncodeGob(v interface{}) ([]byte, error) {
	var err error

	b := new(bytes.Buffer)

	encoder := gob.NewEncoder(b)
	err = encoder.Encode(v)
	if err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}

//DecodeGob deserialize gob to structure
func DecodeGob(d []byte, v interface{}) error {
	var err error

	b := bytes.NewBuffer(d)

	decoder := gob.NewDecoder(b)
	err = decoder.Decode(v)
	if err != nil {
		return err
	}

	return nil
}

//OmmitError strip err from result,
func OmmitError(d interface{}, err error) interface{} {
	return d
}
