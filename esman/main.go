package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/dixonwille/wmenu"
	"github.com/seagiv/common/gutils"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/seagiv/foreign/sss"
)

const minPasswordLength = 1

const loopback = "127.0.0.1"

var (
	flagInit = flag.Bool("init", false, "initialize master key")

	flagCreate = flag.String("create", "", "create storage")
	//	flagAdd    = flag.String("add", "", "add item to storage")
	flagDelete = flag.String("delete", "", "delete item from storage")

	flagFileName  = flag.String("file", "", "(param) file name")
	flagDirectory = flag.String("dir", "", "(param) directory")
	flagStorage   = flag.String("storage", "", "(param) storage file name")

	flagGenKey = flag.Bool("genkey", false, "(test) generate new key")

	flagNumber    = flag.Int("n", 0, "(param) number param")
	flagThreshold = flag.Int("t", 0, "(param) threshold param")

	//menu
	flagMenu    = flag.Bool("menu", false, "(menu) menu mode")
	flagKeyFile = flag.String("keyfile", "", "(param) (test) master key file")
)

func keyInit(n, threshold byte, directory string) error {
	masterKey, err := gutils.GetRandomBuffer(32)
	if err != nil {
		return gutils.FormatErrorS("GetRandomBuffer", "%v", err)
	}

	//masterKey, err := hex.Decode(storageKey)

	//fmt.Printf("!!WARNING!! master key: %s\n", hex.EncodeString(masterKey)) // !!TODO!! must be removed in release !!

	shares, err := sss.Split(n, threshold, []byte(masterKey))
	if err != nil {
		return err
	}

	for i := byte(1); i <= n; i++ {

		for {
			fmt.Printf("Enter password for share number %d (%d symbols or more): ", i, minPasswordLength)
			pass1, err := terminal.ReadPassword(int(syscall.Stdin))

			fmt.Println()

			if err != nil {
				return gutils.FormatErrorS("ReadPassword", "%v", err)
			}

			if len(pass1) < minPasswordLength {
				fmt.Println(gutils.FormatErrorS("ReadPassword", "password too short"))

				continue
			}

			fmt.Printf("Retype password for share number %d: ", i)
			pass2, err := terminal.ReadPassword(int(syscall.Stdin))

			fmt.Println()

			if err != nil {
				return gutils.FormatErrorS("ReadPassword", "%v", err)
			}

			if !bytes.Equal(pass1, pass2) {
				fmt.Println(gutils.FormatErrorS("ReadPassword", "passwords mismatch"))

				continue
			}

			//fmt.Printf("!!WARNING!! decShare %d: %s\n", i, hex.EncodeToString(shares[i])) // !!TODO!! must be removed in release !!

			key := sha256.Sum256(pass1)

			//fmt.Printf("!!WARNING!! key %d: %s\n", i, hex.EncodeToString(key)) // !!TODO!! must be removed in release !!

			encShare, err := gutils.EncryptGCM(shares[i], key[:])
			if err != nil {
				return gutils.FormatErrorS("EncryptGCM", "%v", err)
			}

			//fmt.Printf("!!WARNING!! encShare %d: %s\n", i, hex.EncodeToString(encShare)) // !!TODO!! must be removed in release !!

			fileName := fmt.Sprintf("%s/%d.ss", directory, i)

			err = ioutil.WriteFile(fileName, encShare, 0644)
			if err != nil {
				return gutils.FormatErrorS("WriteFile", "%v", err)
			}

			fmt.Printf("Share %d out of %d saved to %s\n", i, n, fileName)

			break
		}
	}

	return err
}

func keyGet(directory string, n byte) ([]byte, error) {
	//masterKey, _ := hex.DecodeString(storageKey)
	keyS, _ := ioutil.ReadFile(*flagKeyFile)

	keyB, _ := hex.DecodeString(string(keyS))

	return keyB, nil

	/*subset := make(map[byte][]byte, n)

	var reply string

	for i := byte(1); i <= n; i++ {
		for {
			fmt.Printf("Enter share number: ")
			fmt.Scanln(&reply)

			n, err := strconv.Atoi(reply)
			if err != nil {
				fmt.Println("ERROR: Invalid number (must be 0 .. 255)")

				continue
			}

			if n > 255 {
				fmt.Println("ERROR: Invalid number (must be 0 .. 255)")

				continue
			}

			b := byte(n)

			fileName := fmt.Sprintf("%s/%d.ss", directory, b)

			encShare, err := ioutil.ReadFile(fileName)
			if err != nil {
				fmt.Printf("ERROR: can't read file %s: %v\n", fileName, err)

				continue
			}

			fmt.Printf("Password for share number %d: ", b)
			pass, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return nil, gutils.FormatError("ReadPassword", "%v", err)
			}

			fmt.Println()

			key := sha256.Sum256(pass)

			decShare, err := gutils.DecryptGCM(encShare, key[:])
			if err != nil {
				fmt.Println(gutils.FormatError("DecryptGCM", "%v", err))

				continue
			}

			subset[b] = decShare

			break
		}
	}

	masterKey := sss.Combine(subset)

	fmt.Printf("!!WARNING!! master key: %s\n", hex.EncodeToString(masterKey)) // !!TODO!! must be removed in release !!

	return masterKey, nil*/
}

func storageCreate(itemsDir, ssDir, storageFile string, n byte) error {
	var err error

	masterKey, err := keyGet(ssDir, n)
	if err != nil {
		return gutils.FormatErrorS("keyGet", "%v", err)
	}

	s, err := gutils.CreateStorage(storageFile, masterKey)
	if err != nil {
		return gutils.FormatErrorS("CreateStorage", "%v", err)
	}

	files, err := ioutil.ReadDir(itemsDir)
	if err != nil {
		return gutils.FormatErrorS("ReadDir", "%v", err)
	}

	for _, f := range files {
		fileName := itemsDir + "/" + f.Name()

		buffer, err := ioutil.ReadFile(fileName)
		if err != nil {
			return gutils.FormatErrorS("ReadFile", "%s %v", fileName, err)
		}

		err = s.Set(f.Name(), buffer)
		if err != nil {
			return gutils.FormatErrorS("Set", "%v", err)
		}

		fmt.Printf("Set(%s)\n", f.Name())
	}

	return nil
}

/*func storageAdd(name, fileName string) error {
	var err error

	key, err := keyGet()
	if err != nil {
		return err
	}

	s, err := gutils.OpenStorage(*flagStorage, key, gutils.FlagPreserveKey)
	if err != nil {
		return gutils.FormatErrorS("OpenStorage", "%v", err)
	}

	buffer, err := ioutil.ReadFile(fileName)
	if err != nil {
		return gutils.FormatErrorS("ReadFile", "%v", err)
	}

	s.Set(name, buffer)
	if err != nil {
		return gutils.FormatErrorS("Set", "%v", err)
	}

	return nil
}*/

func testGenKey() error {
	var err error

	key, err := gutils.GetRandomBuffer(32)
	if err != nil {
		return err
	}

	fmt.Printf("[%s]\n", hex.EncodeToString(key))

	return nil
}

func menuUnlockKey(keyFile string) ([]byte, error) {
	keyS, err := ioutil.ReadFile(*flagKeyFile)
	if err != nil {
		return nil, err
	}

	keyB, err := hex.DecodeString(string(keyS))
	if err != nil {
		return nil, err
	}

	return keyB, nil
}

func menuStorageList(storageFile string, masterKey []byte) error {
	var err error

	s, err := gutils.OpenStorage(storageFile, masterKey, 0)
	if err != nil {
		return gutils.FormatErrorS("OpenStorage", "%v", err)
	}

	s.List()

	return nil
}

func menuStorageDelete(storageFile string, masterKey []byte) error {
	var err error

	s, err := gutils.OpenStorage(storageFile, masterKey, gutils.FlagPreserveKey)
	if err != nil {
		return gutils.FormatErrorS("OpenStorage", "%v", err)
	}

	var reply string

	fmt.Printf("Enter name for deletion: ")
	fmt.Scanln(&reply)

	err = s.Delete(reply)
	if err != nil {
		return gutils.FormatErrorS("Delete", "%v", err)
	}

	fmt.Printf("item [%s] deleted\n", reply)

	return nil
}

func managerRun(masterKey []byte) error {
	var err error

	l, err := net.Listen("tcp", ":8225")
	if err != nil {
		return gutils.FormatErrorS("Listen", "%v", err)
	}

	conn, err := l.Accept()
	if err != nil {
		return gutils.FormatErrorS("Accept", "%v", err)
	}

	if strings.Split(conn.RemoteAddr().String(), ":")[0] != loopback {
		return gutils.FormatErrorS("connectCheck", "non local connections rejected.")
	}

	rpcServer := &Server{}

	s := rpcServer.OnConnect("", conn.RemoteAddr())

	if len(s) == 0 {
		return fmt.Errorf("no rpc servers")
	}

	localServer := rpc.NewServer()

	for i := 0; i < len(s); i++ {
		localServer.Register(s[i])
	}

	localServer.ServeCodec(jsonrpc.NewServerCodec(conn))

	rpcServer.OnDisconnect()

	return nil
}

func menuExecPayServ(masterKey []byte) error {
	var err error

	cmd := exec.Command("systemctl", "start", "payserv")

	err = cmd.Run()

	if err != nil {
		return err
	}

	fmt.Println("PayServ started, waiting for connect ...")

	err = managerRun(masterKey)

	return err
}

func doMenu(keyFile, storageFile string) error {
	var err error

	for {
		menu := wmenu.NewMenu("Select action:")

		if len(masterKey) == 0 {
			menu.Option("Unlock masterKey", nil, false,
				func(opt wmenu.Opt) error {
					masterKey, err = menuUnlockKey(keyFile)

					if err == nil {
						fmt.Printf("key unlocked.\n")
					}

					return err
				},
			)
		} else {
			menu.Option("Storage List", nil, false,
				func(opt wmenu.Opt) error {
					return menuStorageList(storageFile, masterKey)
				},
			)

			menu.Option("Storage Delete", nil, false,
				func(opt wmenu.Opt) error {
					return menuStorageDelete(storageFile, masterKey)
				},
			)

			menu.Option("Execute and authenticate PayServ", nil, false,
				func(opt wmenu.Opt) error {
					err = menuExecPayServ(masterKey)

					return err
				},
			)
		}

		menu.Option("Exit", nil, false,
			func(opt wmenu.Opt) error {
				os.Exit(0)

				return err
			},
		)

		err = menu.Run()
		if err != nil {
			fmt.Println(gutils.FormatErrorN("%v", err).Error())
		}

		fmt.Println()
	}

	//return nil
}

func main() {
	var err error

	flag.Parse()

	if *flagMenu {
		err = doMenu(*flagKeyFile, *flagStorage)
		if err != nil {
			fmt.Println("ERROR: " + err.Error())
		}
	}

	if *flagInit {
		if (*flagNumber <= 0) || (*flagNumber > 255) || (*flagThreshold <= 0) || (*flagThreshold > 255) || (*flagThreshold > *flagNumber) {
			fmt.Println("ERROR: -n and -t are mandatory, both must be greater then zero, less then 255 and n >= t")
			os.Exit(1)
		}

		if len(*flagDirectory) == 0 {
			fmt.Println("ERROR: -dir is mandatory")
			os.Exit(1)
		}

		err = keyInit(byte(*flagNumber), byte(*flagThreshold), *flagDirectory)
		if err != nil {
			fmt.Println("ERROR: " + err.Error())
		}
	}

	if len(*flagCreate) != 0 {
		if len(*flagStorage) == 0 {
			fmt.Println("ERROR: -storage is mandatory")
			os.Exit(1)
		}

		if len(*flagDirectory) == 0 {
			fmt.Println("ERROR: -dir is mandatory")
			os.Exit(1)
		}

		if (*flagNumber <= 0) || (*flagNumber > 255) {
			fmt.Println("ERROR: -n is mandatory, must be greater then zero, less then 255")
			os.Exit(1)
		}

		err = storageCreate(*flagCreate, *flagDirectory, *flagStorage, byte(*flagNumber))
		if err != nil {
			fmt.Println("ERROR: " + err.Error())
		}
	}

	if *flagGenKey {
		err = testGenKey()
		if err != nil {
			fmt.Println("ERROR: " + err.Error())
		}
	}
}
