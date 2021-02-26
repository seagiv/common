package coinapi

import (
	"errors"
	"sync"

	"github.com/seagiv/common/gutils"

	"github.com/seagiv/foreign/decimal"
)

//CoinETH -
const CoinETH = "ETH"

//CoinETC -
const CoinETC = "ETC"

//CoinZEC -
const CoinZEC = "ZEC"

//CoinBTC -
const CoinBTC = "BTC"

//CoinBCH -
const CoinBCH = "BCH"

//CoinRVN -
const CoinRVN = "RVN"

//CoinDASH -
const CoinDASH = "DASH"

//CoinMONA -
const CoinMONA = "MONA"

//APITypeBitcoin -
const APITypeBitcoin = "BitcoinAPI"

//APITypeEthereum -
const APITypeEthereum = "EthereumAPI"

type coinE struct {
	sync.Mutex

	ChainID int64

	Nonce     uint64
	NonceFile string

	C2C decimal.Decimal
}

type coinB struct {
	Signer string

	Confirmations int64

	PubKeyID  byte
	PrivKeyID byte
	ScriptID  byte

	Fee string // per 100 byte (1x vout)
}

type coinInfo struct {
	Address string `json:"address"`
	Key     string `json:"key"`

	URL string

	TestMode  bool
	TestTrans string

	OutLimit int64

	E coinE
	B coinB
}

var initialized = []string{}

//Transfer args for SendMany function
type Transfer struct {
	ID int64

	Address string

	Amount decimal.Decimal // amount - fee

	TxFee decimal.Decimal // output parameter
}

//Transfers slice for withdrawals
type Transfers []Transfer

//Account private key and address
type Account struct {
	Address    string
	PrivateKey string
}

//Income income transfer from blockhain
type Income struct {
	Block  int64
	TxHash string
	Amount decimal.Decimal
}

var (
	errCoinNotSupported      = errors.New("coin not supported")
	errCoinNotInitialized    = errors.New("coin not initialized")
	errOperationNotSupported = errors.New("operation not supported")
)

//Incoms -
type Incoms []Income

//CoinAPI general template for coin API
type CoinAPI interface {

	//checks if address is valid
	IsValidAddress(address string) error

	//returns address service sends coins from
	GetServiceAddress() string

	//returns balance on certain address
	GetBalance(address string) (decimal.Decimal, error)

	//sends amount of coins from service address to specified address in one transaction, returns: txHash, isRetry, fee, error
	//amount + fee will be spent
	Send(amount decimal.Decimal, addressTo string) (*string, bool, decimal.Decimal, error)

	//spend specified utxo (determine fee and substruct it from amount)
	//to addressTo using private key in one transaction, returns: txHash, isRetry, amount, fee, error
	Spend(addressFrom, addressTo string, inputUTXOs []UTXO, privateKey string, nonce uint64) (*string, bool, decimal.Decimal, decimal.Decimal, error)

	//sends coins from service address to number of different addresses in one transaction, returns txHash, isRetry, error
	SendMany(w *Transfers) (*string, bool, error)

	//checks specified transaction blockchain status, returns isSuccess, fee, error
	Check(txHash string, fee decimal.Decimal) (bool, decimal.Decimal, error)

	//creates priv/pub key pair and address privateKey can be "", in that case it will be generated
	CreateAccount(privateKey string) (*Account, error)

	//override service account, used for test puproses
	SetServiceAccount(address, privKey string)

	//generate redeemScript for given privKey (BitcoinAPI only for now)
	GetRedeemScript(privKey string) (string, error)

	//return API type either BitcoinAPI or EthereumAPI
	GetAPIType() string
}

//GetAvailable list of initialized coins
func GetAvailable() []string {
	return initialized
}

//GetCoinAPI gets API for certain coin if any, error if none
func GetCoinAPI(logID int64, tag string) (CoinAPI, error) {
	var api CoinAPI

	if Coins[tag] == nil {
		return nil, errCoinNotSupported
	}

	if !gutils.IsIn(tag, initialized) {
		return nil, errCoinNotInitialized
	}

	switch tag {
	case CoinETH, CoinETC:
		api = NewEthereumAPI(logID, tag, Coins[tag])
	case CoinBTC, CoinBCH, CoinZEC, CoinRVN, CoinDASH, CoinMONA:
		api = NewBitcoinAPI(logID, tag, Coins[tag])
	default:
		return api, errCoinNotSupported
	}

	return api, nil
}

//Coins coin constants, to disable coin support comment certain coin
var Coins = map[string]*coinInfo{
	CoinETH:  &coinInfo{OutLimit: 001, E: coinE{ChainID: 01, C2C: decimal.New(1, 18)}},
	CoinETC:  &coinInfo{OutLimit: 001, E: coinE{ChainID: 61, C2C: decimal.New(1, 18)}},
	CoinZEC:  &coinInfo{OutLimit: 500, B: coinB{Fee: "0.00001", Confirmations: 06, PubKeyID: 0xB8, PrivKeyID: 0x80, ScriptID: 0xBD}},
	CoinBTC:  &coinInfo{OutLimit: 500, B: coinB{Fee: "0.00001", Confirmations: 06, PubKeyID: 0x00, PrivKeyID: 0x80, ScriptID: 0x05}},
	CoinBCH:  &coinInfo{OutLimit: 500, B: coinB{Fee: "0.00001", Confirmations: 06, PubKeyID: 0x00, PrivKeyID: 0x80, ScriptID: 0x05}},
	CoinRVN:  &coinInfo{OutLimit: 500, B: coinB{Fee: "0.00050", Confirmations: 06, PubKeyID: 0x3C, PrivKeyID: 0x80, ScriptID: 0x7A}},
	CoinDASH: &coinInfo{OutLimit: 500, B: coinB{Fee: "0.00010", Confirmations: 06, PubKeyID: 0x4C, PrivKeyID: 0xCC, ScriptID: 0x10}},
	CoinMONA: &coinInfo{OutLimit: 500, B: coinB{Fee: "0.00030", Confirmations: 05, PubKeyID: 0x32, PrivKeyID: 0xB0, ScriptID: 0x37}},
}
