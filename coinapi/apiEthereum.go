package coinapi

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"regexp"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seagiv/common/gutils"
	"github.com/seagiv/foreign/decimal"
	"github.com/seagiv/foreign/jsonrpcf"
)

const ethGasLimit = 60000
const ethGasLimitStrict = 21000

var ethGwei = big.NewInt(1000000000)
var ethMaxGasPrice = big.NewInt(1000000000) // 1 Gwei

//EthereumAPI interface for bitcoind based coins
type EthereumAPI struct {
	Tag  string
	Coin *coinInfo

	logID int64

	client *jsonrpcf.Client
}

//EthereumReceiptItem representation of ethereum reply for getTransactionReceipt
type EthereumReceiptItem struct {
	BlockHash   string
	BlockNumber string
	GasUsed     hexutil.Big
	Status      string
}

//InitEthereum initialises ethereum based coins
func InitEthereum(tag string, config gutils.CoinConfig, testMode bool) error {
	var err error

	if Coins[tag] == nil {
		return gutils.FormatErrorS(tag, "coin %s not supported", tag)
	}

	Coins[tag].URL = config.URL
	Coins[tag].E.NonceFile = "/var/lib/payserv/" + tag + ".nonce"

	Coins[tag].TestMode = testMode

	gutils.RemoteLog.PutInfoS(tag, "addressFrom [%s]\n", Coins[tag].Address)

	//jsonrpcf.EnableDebug()

	jsonrpcf.SetVersion("2.0")

	Coins[tag].E.Nonce, err = ethGetNonce(Coins[tag].Address, Coins[tag].URL)
	if err != nil {
		return gutils.FormatErrorSD("ethGetNonce", tag, "RPC test FAILED [%v]", err)
	}

	cachedNonce, err := ethLoadNonce(Coins[tag].E.NonceFile)

	if err != nil {
		gutils.RemoteLog.PutWarningS(tag, "can't read nonce from file [%s]", Coins[tag].E.NonceFile)
	} else {
		if cachedNonce != Coins[tag].E.Nonce {
			gutils.RemoteLog.PutWarningS(tag, "saved nonce out of sync with nonce from blockchain (%d != %d)", cachedNonce, Coins[tag].E.Nonce)
			gutils.RemoteLog.PutWarningS(tag, "must be geth issue, using cached value %d", cachedNonce)

			Coins[tag].E.Nonce = cachedNonce
		}
	}

	Coins[tag].TestTrans = config.TestTransaction

	initialized = append(initialized, tag)

	return nil
}

//NewEthereumAPI creates interface instance for ethereum based coins
func NewEthereumAPI(logID int64, tag string, c *coinInfo) *EthereumAPI {
	return &EthereumAPI{logID: logID, Tag: tag, Coin: c}
}

func ethLoadNonce(fileName string) (uint64, error) {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(b), nil
}

func ethWeiToGWei(w *big.Int) decimal.Decimal {
	d := decimal.NewFromBigInt(w, 0)

	return d.Div(decimal.NewFromBigInt(ethGwei, 0))
}

func ethGetNonce(address, url string) (uint64, error) {
	var err error

	client := jsonrpcf.NewHTTPClient(url)
	defer client.Close()

	var reply hexutil.Uint64

	err = client.Call("eth_getTransactionCount", []string{address, "pending"}, &reply)
	if err != nil {
		return 0, err
	}

	return uint64(reply), err
}

func (a *EthereumAPI) saveNonce(n uint64) error {
	b := make([]byte, 8)

	binary.LittleEndian.PutUint64(b, n)

	return ioutil.WriteFile(a.Coin.E.NonceFile, b, 644)
}

func (a *EthereumAPI) getTransactionReceipt(hash string) (*EthereumReceiptItem, error) {
	var reply EthereumReceiptItem

	err := a.client.Call("eth_getTransactionReceipt", []string{hash}, &reply)
	if err != nil {
		return nil, err
	}

	return &reply, err
}

func (a *EthereumAPI) getGasPrice() (*big.Int, error) {
	var reply hexutil.Big

	err := a.client.Call("eth_gasPrice", nil, &reply)
	if err != nil {
		return nil, err
	}

	return (*big.Int)(&reply), err
}

func (a *EthereumAPI) isRetryError(err error) bool {
	var rcpError gutils.ErrorRPC

	if err == nil {
		return false
	}

	json.Unmarshal([]byte(err.Error()), &rcpError)

	return regexp.MustCompile("^(insufficient funds)|(balance too low)").MatchString(rcpError.Message)
}

//IsValidAddress -
func (a *EthereumAPI) IsValidAddress(address string) error {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")

	if !re.MatchString(address) {
		return gutils.FormatErrorI(a.logID, "address [%s] is not valid for Ethereum", address)
	}

	return nil
}

//GetServiceAddress -
func (a *EthereumAPI) GetServiceAddress() string {
	return a.Coin.Address
}

//GetBalance -
func (a *EthereumAPI) GetBalance(address string) (decimal.Decimal, error) {
	var err error

	c := jsonrpcf.NewHTTPClient(a.Coin.URL)
	defer c.Close()

	var reply string

	err = c.Call("eth_getBalance", []string{address, "latest"}, &reply)
	if err != nil {
		return decimal.Zero, err
	}

	wei := new(big.Int)
	wei.SetString(reply, 0)

	result := decimal.NewFromBigInt(wei, 0)

	result = result.Div(a.Coin.E.C2C)

	return result, nil
}

//Send -
func (a *EthereumAPI) Send(amount decimal.Decimal, addressTo string) (*string, bool, decimal.Decimal, error) {
	var err error

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, false, decimal.Zero, fmt.Errorf("amount must not be zero or negative")
	}

	amountWei := amount.Mul(a.Coin.E.C2C)

	if len(a.Coin.Key) == 0 {
		return nil, false, decimal.Zero, fmt.Errorf("key not loaded")
	}

	a.Coin.E.Lock()
	defer a.Coin.E.Unlock()

	a.client = jsonrpcf.NewHTTPClient(a.Coin.URL)
	defer a.client.Close()

	gasPrice, err := a.getGasPrice()
	if err != nil {
		return nil, false, decimal.Zero, fmt.Errorf("getGasPrice: %v", err)
	}

	if gasPrice.Cmp(ethMaxGasPrice) > 0 {
		gutils.RemoteLog.PutDebugI(a.logID, "GasPrice too high limiting it %s -> %s", ethWeiToGWei(gasPrice).String(), ethWeiToGWei(ethMaxGasPrice).String())

		gasPrice.Set(ethMaxGasPrice)
	}

	var amountI big.Int

	amountI.SetString(amountWei.String(), 10)

	tx := types.NewTransaction(
		a.Coin.E.Nonce,                 //nonce
		common.HexToAddress(addressTo), //Address send to
		&amountI,                       //amount
		ethGasLimit,                    //gas limit
		gasPrice,                       //gas price
		nil,                            //contract
	)

	gutils.RemoteLog.PutDebugI(a.logID, "Address: %s -> %s",
		a.Coin.Address,
		addressTo,
	)

	gutils.RemoteLog.PutDebugI(a.logID, "amount(%s): %s, gasLimit: %d, gasPrice(Gwei): %s, Nonce: %d",
		a.Tag, amount.String(),
		ethGasLimit,
		ethWeiToGWei(gasPrice).String(),
		a.Coin.E.Nonce,
	)

	privKey, err := crypto.HexToECDSA(a.Coin.Key)
	if err != nil {
		return nil, false, decimal.Zero, fmt.Errorf("HexToECDSA: %v", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(a.Coin.E.ChainID)), privKey)
	if err != nil {
		return nil, false, decimal.Zero, fmt.Errorf("SignTx: %v", err)
	}

	data, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, false, decimal.Zero, fmt.Errorf("EncodeToBytes: %v", err)
	}

	var replyTxHash string
	var errF error

	gutils.RemoteLog.PutDebugI(a.logID, "SignedTx: %s", common.ToHex(data))

	if a.Coin.TestMode {
		replyTxHash = a.Coin.TestTrans

		//err = errors.New("{\"code\":-32000,\"message\":\"insufficient funds for gas * price + value\"}")
		//errF = gutils.FormatErrorSI("eth_sendRawTransaction", g.ID, "%v", err)
	} else {
		err = a.client.Call("eth_sendRawTransaction", []string{common.ToHex(data)}, &replyTxHash)
		if err != nil {
			errF = fmt.Errorf("eth_sendRawTransaction: %v", err)
		}
	}

	if errF != nil {
		return nil, a.isRetryError(err), decimal.Zero, errF
	}

	a.Coin.E.Nonce++

	gutils.RemoteLog.PutDebugI(a.logID, "Hash: %s", replyTxHash)

	err = a.saveNonce(a.Coin.E.Nonce)
	if err != nil {
		gutils.RemoteLog.PutWarningSI("ethSaveNonce", a.logID, "can't store nonce to file %v", err)
	}

	/*if testMode {
		return nil, false, fmt.Errorf("test error")
	}*/

	return &replyTxHash, false, decimal.NewFromBigInt(gasPrice, 0), nil
}

func (a *EthereumAPI) ethWeiToETH(w *big.Int) decimal.Decimal {
	d := decimal.NewFromBigInt(w, 0)

	return d.Div(a.Coin.E.C2C)
}

//Spend -
func (a *EthereumAPI) Spend(addressFrom, addressTo string, inputUTXOs []UTXO, privateKey string, nonce uint64) (*string, bool, decimal.Decimal, decimal.Decimal, error) {
	var err error

	amount, err := a.GetBalance(addressFrom)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, err
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, false, decimal.Zero, decimal.Zero, fmt.Errorf("amount must not be zero or negative")
	}

	amountWei := amount.Mul(a.Coin.E.C2C)

	a.client = jsonrpcf.NewHTTPClient(a.Coin.URL)
	defer a.client.Close()

	gasPrice, err := a.getGasPrice()
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, fmt.Errorf("getGasPrice: %v", err)
	}

	if gasPrice.Cmp(ethMaxGasPrice) > 0 {
		gutils.RemoteLog.PutDebugI(a.logID, "GasPrice too high limiting it %s -> %s", ethWeiToGWei(gasPrice).String(), ethWeiToGWei(ethMaxGasPrice).String())

		gasPrice.Set(ethMaxGasPrice)
	}

	var amountI big.Int

	amountI.SetString(amountWei.String(), 10)

	feeI := big.NewInt(ethGasLimitStrict)

	feeI = feeI.Mul(feeI, gasPrice)

	amountI = *amountI.Sub(&amountI, feeI)

	tx := types.NewTransaction(
		nonce,                          //nonce
		common.HexToAddress(addressTo), //Address send to
		&amountI,                       //amount
		ethGasLimitStrict,              //gas limit
		gasPrice,                       //gas price
		nil,                            //contract
	)

	gutils.RemoteLog.PutDebugI(a.logID, "Address: %s -> %s",
		a.Coin.Address,
		addressTo,
	)

	gutils.RemoteLog.PutDebugI(a.logID, "amount(%s): %s, fee: %s, gasLimit: %d, gasPrice(Gwei): %s, Nonce: %d",
		a.Tag, a.ethWeiToETH(&amountI).String(),
		a.ethWeiToETH(feeI).String(),
		ethGasLimitStrict,
		ethWeiToGWei(gasPrice).String(),
		nonce,
	)

	privKey, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, fmt.Errorf("HexToECDSA: %v", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(a.Coin.E.ChainID)), privKey)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, fmt.Errorf("SignTx: %v", err)
	}

	data, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, fmt.Errorf("EncodeToBytes: %v", err)
	}

	var replyTxHash string
	var errF error

	gutils.RemoteLog.PutDebugI(a.logID, "SignedTx: %s", common.ToHex(data))

	if a.Coin.TestMode {
		replyTxHash = a.Coin.TestTrans

		//err = errors.New("{\"code\":-32000,\"message\":\"insufficient funds for gas * price + value\"}")
		//errF = gutils.FormatErrorSI("eth_sendRawTransaction", g.ID, "%v", err)
	} else {
		err = a.client.Call("eth_sendRawTransaction", []string{common.ToHex(data)}, &replyTxHash)
		if err != nil {
			errF = fmt.Errorf("eth_sendRawTransaction: %v", err)
		}
	}

	if errF != nil {
		return nil, a.isRetryError(err), decimal.Zero, decimal.Zero, errF
	}

	gutils.RemoteLog.PutDebugI(a.logID, "Hash: %s", replyTxHash)

	/*if testMode {
		return nil, false, fmt.Errorf("test error")
	}*/

	return &replyTxHash, false, amount, decimal.NewFromBigInt(gasPrice, 0), nil
}

//SendMany -
func (a *EthereumAPI) SendMany(wds *Transfers) (*string, bool, error) {
	if len(*wds) > 1 {
		return nil, false, fmt.Errorf("multiple outputs not supported by EthereumAPI")
	}

	a.logID = (*wds)[0].ID

	txHash, isRetry, sendFee, err := a.Send((*wds)[0].Amount, (*wds)[0].Address)

	(*wds)[0].TxFee = sendFee

	return txHash, isRetry, err
}

//Check -
func (a *EthereumAPI) Check(tx string, fee decimal.Decimal) (bool, decimal.Decimal, error) {
	var err error
	var res bool
	var feeCheck decimal.Decimal

	a.client = jsonrpcf.NewHTTPClient(a.Coin.URL)
	defer a.client.Close()

	//check transaction receip
	txRecipt, err := a.getTransactionReceipt(tx)
	if err != nil {
		return false, fee, gutils.FormatErrorSI("getTransactionReceipt", a.logID, "%v", err)
	}

	if txRecipt.BlockNumber == "" {
		return false, fee, gutils.FormatErrorI(a.logID, "transaction not mined yet")
	}

	res = (txRecipt.Status != "0x0")

	gasUsed := (*big.Int)(&txRecipt.GasUsed)

	feeCheck = fee.Mul(decimal.NewFromBigInt(gasUsed, 0))

	feeCheck = feeCheck.Div(a.Coin.E.C2C)

	return res, feeCheck, nil
}

//CreateAccount -
func (a *EthereumAPI) CreateAccount(privateKey string) (*Account, error) {
	var err error

	var key *ecdsa.PrivateKey

	if privateKey == "" {
		key, err = crypto.GenerateKey()
	} else {
		keyB, err := hex.DecodeString(privateKey)
		if err != nil {
			return nil, err
		}

		key, err = crypto.ToECDSA(keyB)
	}
	if err != nil {
		return nil, err
	}

	acc := Account{
		Address:    crypto.PubkeyToAddress(key.PublicKey).Hex(),
		PrivateKey: hex.EncodeToString(crypto.FromECDSA(key)),
	}

	return &acc, nil
}

//SetServiceAccount -
func (a *EthereumAPI) SetServiceAccount(address, privKey string) {
	var err error

	a.Coin.Address = address
	a.Coin.Key = privKey

	a.Coin.E.Nonce, err = ethGetNonce(a.Coin.Address, a.Coin.URL)
	if err != nil {
		gutils.RemoteLog.PutErrorS(a.Tag, "RPC test FAILED [%v]", err)

		return
	}
}

//GetRedeemScript -
func (a *EthereumAPI) GetRedeemScript(privKey string) (string, error) {
	return "", errOperationNotSupported
}

//GetAPIType -
func (a *EthereumAPI) GetAPIType() string {
	return APITypeEthereum
}
