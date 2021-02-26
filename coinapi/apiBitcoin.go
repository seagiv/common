package coinapi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/seagiv/common/gutils"
	"github.com/seagiv/foreign/decimal"
	"github.com/seagiv/foreign/jsonrpcf"
)

//UTXO UTXO structure representation for bitcoind based coins
type UTXO struct {
	TxID          string `json:"txid"`
	Vout          uint32 `json:"vout"`
	Generated     bool
	Address       string
	Account       string
	RedeemScript  string          `json:"redeemScript"`
	ScriptPubKey  string          `json:"scriptPubKey"`
	Amount        decimal.Decimal `json:"amount"`
	Confirmations int
	Spendable     bool
}

type blockChain struct {
	Chain  string `json:"chain"`
	Blocks uint64 `json:"blocks"`
}

type replyAddress struct {
	IsValid      bool   `json:"isvalid"`
	Address      string `json:"address"`
	ScriptPubKey string `json:"scriptPubKey"`
	IsMine       bool   `json:"ismine"`
	IsWatchOnly  bool   `json:"iswatchonly"`
	IsScript     bool   `json:"isscript"`
	PubKey       string `json:"pubkey"`
	IsCompressed bool   `json:"iscompressed"`
	Account      string `json:"account"`
}

type replyGetTransaction struct {
	Confirmations int64  `json:"confirmations"`
	BlockHash     string `json:"blockhash"`
}

//BitcoinAPI interface for bitcoind based coins
type BitcoinAPI struct {
	Tag  string
	Coin *coinInfo

	logID int64

	client *jsonrpcf.Client
}

const signerUserName = "nobody"

var signerUID uint64
var signerGID uint64

func bitcoinTestRPC(URL, address string) error {
	client := jsonrpcf.NewHTTPClient(URL)
	defer client.Close()

	var replyValidate replyAddress

	return client.Call("validateaddress", []string{address}, &replyValidate)
}

func initSigner() error {
	signerUser, err := user.Lookup(signerUserName)
	if err != nil {
		return fmt.Errorf("user.Lookup %s failed %v", signerUserName, err)
	}

	signerUID, err = strconv.ParseUint(signerUser.Uid, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid user uid %s ", signerUser.Uid)
	}

	signerGID, err = strconv.ParseUint(signerUser.Gid, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid user gid %s ", signerUser.Gid)
	}

	return nil
}

//InitBitcoin initialises bitcoind based coins
func InitBitcoin(tag string, config gutils.CoinConfig, testMode bool) error {
	var err error

	if Coins[tag] == nil {
		return errCoinNotSupported
	}

	switch tag {
	case CoinBTC, CoinBCH, CoinZEC, CoinRVN, CoinDASH, CoinMONA:
	default:
		return errCoinNotSupported
	}

	if signerUID == 0 {
		err = initSigner()

		if err != nil {
			return gutils.FormatErrorSD("initSigner", tag, "%v", err)
		}
	}

	Coins[tag].B.Signer = config.Signer
	Coins[tag].URL = config.URL

	Coins[tag].TestMode = testMode

	gutils.RemoteLog.PutInfoS(tag, "addressFrom [%s]", Coins[tag].Address)

	jsonrpcf.EnableDebug()

	err = bitcoinTestRPC(Coins[tag].URL, Coins[tag].Address)
	if err != nil {
		return gutils.FormatErrorSD("RPC", tag, "(%s) test FAILED [%v]", Coins[tag].URL, err)
	}

	_, err = os.Stat(Coins[tag].B.Signer)
	if err != nil {
		return gutils.FormatErrorSD("Signer", tag, "signer [%s] not found", Coins[tag].B.Signer)
	}

	Coins[tag].TestTrans = config.TestTransaction

	initialized = append(initialized, tag)

	return nil
}

//NewBitcoinAPI creates interface instance for bitcoind based coins
func NewBitcoinAPI(logID int64, tag string, c *coinInfo) *BitcoinAPI {
	return &BitcoinAPI{logID: logID, Tag: tag, Coin: c}
}

//IsValidAddress -
func (a *BitcoinAPI) IsValidAddress(address string) error {
	var err error
	var replyValidate replyAddress

	c := a.client

	if c == nil {
		c = jsonrpcf.NewHTTPClient(a.Coin.URL)
		defer c.Close()
	}

	err = c.Call("validateaddress", []string{address}, &replyValidate)
	if err != nil {
		return gutils.FormatErrorSI("validateaddress", a.logID, "%v", err)
	}

	if !replyValidate.IsValid {
		return gutils.FormatErrorI(a.logID, "address [%s] is not valid for %s", address, a.Tag)
	}

	return nil
}

//GetServiceAddress -
func (a *BitcoinAPI) GetServiceAddress() string {
	return a.Coin.Address
}

//GetBalance -
func (a *BitcoinAPI) GetBalance(address string) (decimal.Decimal, error) {
	var err error
	var replyUTXOs []UTXO
	var balance decimal.Decimal

	c := jsonrpcf.NewHTTPClient(a.Coin.URL)
	defer c.Close()

	err = c.Call("listunspent", []interface{}{a.Coin.B.Confirmations, 9999999, []interface{}{address}}, &replyUTXOs)
	if err != nil {
		return balance, fmt.Errorf("listunspent: %v", err)
	}

	for i := 0; i < len(replyUTXOs); i++ {
		balance = balance.Add(replyUTXOs[i].Amount)
	}

	return balance, nil
}

func (a *BitcoinAPI) checkOnlineNode() error {
	var err error
	var replyAddress replyAddress

	switch a.Tag {
	case CoinBTC, CoinMONA:
		err = a.client.Call("getaddressinfo", []string{a.Coin.Address}, &replyAddress)
	default:
		err = a.client.Call("validateaddress", []string{a.Coin.Address}, &replyAddress)
	}

	if err != nil {
		return gutils.FormatErrorSI("addressinfo", a.logID, "%v", err)
	}

	if replyAddress.IsMine {
		gutils.RemoteLog.PutWarningS("IsMine", "online node must not know privKey for %s", a.Tag)

		// !!FIXME!! need to be fatal error here after RVN and BTC node is fixed
	}

	if !(replyAddress.IsWatchOnly || replyAddress.IsMine) {
		gutils.RemoteLog.PutWarningSI("address", a.logID, "%s", a.Coin.Address)

		return gutils.FormatErrorI(a.logID, "send address not found on an online node, use importaddress to add it")
	}

	return nil
}

func (a *BitcoinAPI) signTx(unsignedTx string, inputUTXOs []UTXO, height uint64) (string, error) {
	var err error

	prevTxs, err := json.Marshal(inputUTXOs)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %v", err)
	}

	signCommand := "sign=ALL"

	switch a.Tag {
	case CoinZEC:
		signCommand = fmt.Sprintf("sign=%d:ALL", height)
	case CoinBCH:
		signCommand = "sign=ALL|FORKID"
	}

	cmd := exec.Command(a.Coin.B.Signer,
		unsignedTx,
		"set=privatekeys:[\""+a.Coin.Key+"\"]",
		"set=prevtxs:"+string(prevTxs),
		signCommand,
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(signerUID), Gid: uint32(signerGID)}

	var o bytes.Buffer
	var e bytes.Buffer

	cmd.Stdout = &o
	cmd.Stderr = &e

	err = cmd.Run()
	if err != nil {
		if len(e.String()) > 0 {
			r := strings.Split(e.String(), ":")

			if r[0] == "error" {
				err = errors.New(strings.Trim(strings.Join(r[1:], " "), " "))
			}
		}

		return "", fmt.Errorf("cmd.Run: %v", err)
	}

	if len(o.String()) < 2*20 { // 2x of hash size
		return "", fmt.Errorf("signer reply too short [%s]", o.String())
	}

	return strings.Trim(o.String(), "\n"), nil
}

//GetRedeemScript -
func (a *BitcoinAPI) GetRedeemScript(privKey string) (string, error) {
	var err error

	var redeemScript []byte

	wif, err := btcutil.DecodeWIF(privKey)
	if err != nil {
		return "", err
	}

	pubKeyB := wif.PrivKey.PubKey().SerializeCompressed()

	redeemScript = append(redeemScript, txscript.OP_0)
	redeemScript = append(redeemScript, 20) // size of Hash160
	redeemScript = append(redeemScript, btcutil.Hash160(pubKeyB)...)

	return hex.EncodeToString(redeemScript), nil
}

//Send -
func (a *BitcoinAPI) Send(amount decimal.Decimal, addressTo string) (*string, bool, decimal.Decimal, error) {
	var err error
	var unsignedTx string
	var signedTx string
	var replyTxHash string
	var replyUTXOs []UTXO
	var inputUTXOs []UTXO
	var replyBlockChain blockChain
	var inputAmount decimal.Decimal

	//	jsonrpc1.JSONRPC_DEBUG = true

	a.client = jsonrpcf.NewHTTPClient(a.Coin.URL)
	defer a.client.Close()

	fee, err := decimal.NewFromString(a.Coin.B.Fee)
	if err != nil {
		return nil, false, decimal.Zero, fmt.Errorf("fee [%s] is invalid", a.Coin.B.Fee)
	}

	amountFull := amount.Add(fee)

	gutils.RemoteLog.PutDebugI(a.logID, "%s -> %s,",
		a.Coin.Address,
		addressTo,
	)

	gutils.RemoteLog.PutDebugI(a.logID, "amount: %s (%s), fee: %s, total: %s",
		amount.String(), a.Tag,
		fee.String(),
		amountFull.String(),
	)

	err = a.IsValidAddress(addressTo)
	if err != nil {
		return nil, false, decimal.Zero, err
	}

	err = a.checkOnlineNode()
	if err != nil {
		return nil, false, decimal.Zero, err
	}

	err = a.client.Call("listunspent", []interface{}{a.Coin.B.Confirmations, 9999999, []interface{}{a.Coin.Address}}, &replyUTXOs)
	if err != nil {
		return nil, false, decimal.Zero, fmt.Errorf("listunspent: %v", err)
	}

	for i := 0; i < len(replyUTXOs); i++ {
		replyUTXOs[i].RedeemScript, err = a.GetRedeemScript(a.Coin.Key)
		if err != nil {
			return nil, false, decimal.Zero, fmt.Errorf("getRedeemScript: %v", err)
		}

		inputUTXOs = append(inputUTXOs, replyUTXOs[i])
		inputAmount = inputAmount.Add(replyUTXOs[i].Amount)

		gutils.RemoteLog.PutDebugI(a.logID, "+INPUT: %s, %s, total %s", replyUTXOs[i].TxID, replyUTXOs[i].Amount.String(), inputAmount.String())

		if inputAmount.GreaterThanOrEqual(amountFull) {
			break
		}
	}

	if inputAmount.LessThan(amountFull) {
		return nil, true, decimal.Zero, fmt.Errorf("not enough unspent funds (%s < %s)", inputAmount.String(), amountFull.String())
	}

	change := inputAmount.Sub(amountFull)

	VOut := make(map[string]decimal.Decimal)

	VOut[addressTo] = amount

	gutils.RemoteLog.PutDebugI(a.logID, "+OUT(R): %s %s %s", addressTo, amount.String(), a.Tag)

	if change.GreaterThan(decimal.Zero) {
		VOut[a.Coin.Address] = change

		gutils.RemoteLog.PutDebugI(a.logID, "+OUT(C): %s %s %s", a.Coin.Address, change.String(), a.Tag)
	}

	err = a.client.Call("createrawtransaction", []interface{}{inputUTXOs, VOut}, &unsignedTx)
	if err != nil {
		return nil, false, decimal.Zero, fmt.Errorf("createrawtransaction: %v", err)
	}

	gutils.RemoteLog.PutDebugI(a.logID, "UnsignedTx: %s", unsignedTx)

	err = a.client.Call("getblockchaininfo", []interface{}{}, &replyBlockChain)
	if err != nil {
		return nil, false, decimal.Zero, fmt.Errorf("getblockchaininfo: %v", err)
	}

	signedTx, err = a.signTx(unsignedTx, inputUTXOs, replyBlockChain.Blocks)
	if err != nil {
		return nil, false, decimal.Zero, err
	}

	gutils.RemoteLog.PutDebugI(a.logID, "SignedTx: %s", signedTx)

	if a.Coin.TestMode {
		replyTxHash = a.Coin.TestTrans
	} else {
		err = a.client.Call("sendrawtransaction", []interface{}{signedTx}, &replyTxHash)
		if err != nil {
			return nil, false, decimal.Zero, fmt.Errorf("sendrawtransaction:  %v", err)
		}
	}

	gutils.RemoteLog.PutDebugI(a.logID, "Hash: %s", replyTxHash)

	/*if a.Coin.TestMode {
		return nil, false, decimal.Zero, errors.New("test mode")
	}*/

	return &replyTxHash, false, fee, err
}

//Spend -
func (a *BitcoinAPI) Spend(addressFrom, addressTo string, inputUTXOs []UTXO, privateKey string, nonce uint64) (*string, bool, decimal.Decimal, decimal.Decimal, error) {
	var err error
	var unsignedTx string
	var signedTx string
	var replyTxHash string
	var replyBlockChain blockChain

	//	jsonrpc1.JSONRPC_DEBUG = true

	a.client = jsonrpcf.NewHTTPClient(a.Coin.URL)
	defer a.client.Close()

	fee, err := decimal.NewFromString(a.Coin.B.Fee)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, fmt.Errorf("fee [%s] is invalid", a.Coin.B.Fee)
	}

	var amount decimal.Decimal

	for _, v := range inputUTXOs {
		gutils.RemoteLog.PutDebugI(a.logID, "+INPUT: %s, %s, total %s", v.TxID, v.Amount.String(), v.Amount.String())

		amount = amount.Add(v.Amount)
	}

	gutils.RemoteLog.PutDebugI(a.logID, " -> %s, amount: %s (%s), fee: %s",
		addressTo,
		amount.String(), a.Tag,
		fee.String(),
	)

	err = a.IsValidAddress(addressTo)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, err
	}

	VOut := make(map[string]decimal.Decimal)

	VOut[addressTo] = amount.Sub(fee)

	gutils.RemoteLog.PutDebugI(a.logID, "+OUT(R): %s %s %s", addressTo, VOut[addressTo].String(), a.Tag)

	err = a.client.Call("createrawtransaction", []interface{}{inputUTXOs, VOut}, &unsignedTx)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, fmt.Errorf("createrawtransaction: %v", err)
	}

	gutils.RemoteLog.PutDebugI(a.logID, "UnsignedTx: %s", unsignedTx)

	err = a.client.Call("getblockchaininfo", []interface{}{}, &replyBlockChain)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, fmt.Errorf("getblockchaininfo: %v", err)
	}

	signedTx, err = a.signTx(unsignedTx, inputUTXOs, replyBlockChain.Blocks)
	if err != nil {
		return nil, false, decimal.Zero, decimal.Zero, err
	}

	gutils.RemoteLog.PutDebugI(a.logID, "SignedTx: %s", signedTx)

	if a.Coin.TestMode {
		replyTxHash = a.Coin.TestTrans
	} else {
		err = a.client.Call("sendrawtransaction", []interface{}{signedTx}, &replyTxHash)
		if err != nil {
			return nil, false, decimal.Zero, decimal.Zero, fmt.Errorf("sendrawtransaction:  %v", err)
		}
	}

	gutils.RemoteLog.PutDebugI(a.logID, "Hash: %s", replyTxHash)

	/*if a.Coin.TestMode {
		return nil, false, errors.New("test mode")
	}*/

	return &replyTxHash, false, amount, fee, err
}

//SendMany -
func (a *BitcoinAPI) SendMany(wds *Transfers) (*string, bool, error) {
	var err error
	var unsignedTx string
	var signedTx string
	var replyTxHash string
	var replyUTXOs []UTXO
	var inputUTXOs []UTXO
	var replyBlockChain blockChain
	var inputAmount decimal.Decimal

	//jsonrpc1.JSONRPC_DEBUG = true

	a.client = jsonrpcf.NewHTTPClient(a.Coin.URL)
	defer a.client.Close()

	fee, err := decimal.NewFromString(a.Coin.B.Fee)
	if err != nil {
		return nil, false, fmt.Errorf("fee [%s] is invalid", a.Coin.B.Fee)
	}

	gutils.RemoteLog.PutDebugS(a.Tag, "From: %s, fee: %s %s (per out)", a.Coin.Address, fee.String(), a.Tag)

	var amountTotal decimal.Decimal

	vOut := make(map[string]decimal.Decimal)

	for i := 0; i < len(*wds); i++ {
		amountTotal = amountTotal.Add((*wds)[i].Amount)

		vOut[(*wds)[i].Address] = vOut[(*wds)[i].Address].Add((*wds)[i].Amount)

		gutils.RemoteLog.PutDebugSI(a.Tag, (*wds)[i].ID, "+OUT(R): %s %s %s", (*wds)[i].Address, (*wds)[i].Amount.String(), a.Tag)

		(*wds)[i].TxFee = decimal.Zero
	}

	amountFee := fee.Mul(decimal.New(int64(len(vOut)), 0))

	(*wds)[0].TxFee = amountFee

	amountTotal = amountTotal.Add(amountFee)

	gutils.RemoteLog.PutDebugS(a.Tag, "feeTotal: %s %s", amountFee.String(), a.Tag)
	gutils.RemoteLog.PutDebugS(a.Tag, "amountTotal: %s %s", amountTotal.String(), a.Tag)

	err = a.checkOnlineNode()
	if err != nil {
		return nil, false, err
	}

	err = a.client.Call("listunspent", []interface{}{a.Coin.B.Confirmations, 9999999, []interface{}{a.Coin.Address}}, &replyUTXOs)
	if err != nil {
		return nil, false, fmt.Errorf("listunspent: %v", err)
	}

	for i := 0; i < len(replyUTXOs); i++ {
		replyUTXOs[i].RedeemScript, err = a.GetRedeemScript(a.Coin.Key)
		if err != nil {
			return nil, false, fmt.Errorf("getRedeemScript: %v", err)
		}

		inputUTXOs = append(inputUTXOs, replyUTXOs[i])
		inputAmount = inputAmount.Add(replyUTXOs[i].Amount)

		gutils.RemoteLog.PutDebugS(a.Tag, "+INPUT: %s, %s, total %s", replyUTXOs[i].TxID, replyUTXOs[i].Amount.String(), inputAmount.String())

		if inputAmount.GreaterThanOrEqual(amountTotal) {
			break
		}
	}

	if inputAmount.LessThan(amountTotal) {
		return nil, true, fmt.Errorf("not enough unspent funds (%s < %s)", inputAmount.String(), amountTotal.String())
	}

	change := inputAmount.Sub(amountTotal)

	if change.GreaterThan(decimal.Zero) {
		vOut[a.Coin.Address] = change

		gutils.RemoteLog.PutDebugS(a.Tag, "+OUT(C): %s %s %s", a.Coin.Address, change.String(), a.Tag)
	}

	err = a.client.Call("createrawtransaction", []interface{}{inputUTXOs, vOut}, &unsignedTx)
	if err != nil {
		return nil, false, fmt.Errorf("createrawtransaction: %v", err)
	}

	gutils.RemoteLog.PutDebugI(a.logID, "UnsignedTx: %s", unsignedTx)

	err = a.client.Call("getblockchaininfo", []interface{}{}, &replyBlockChain)
	if err != nil {
		return nil, false, fmt.Errorf("getblockchaininfo: %v", err)
	}

	signedTx, err = a.signTx(unsignedTx, inputUTXOs, replyBlockChain.Blocks)
	if err != nil {
		return nil, false, err
	}

	gutils.RemoteLog.PutDebugS(a.Tag, "SignedTx: %s", signedTx)

	if a.Coin.TestMode {
		replyTxHash = a.Coin.TestTrans
	} else {
		err = a.client.Call("sendrawtransaction", []interface{}{signedTx}, &replyTxHash)
		if err != nil {
			return nil, false, fmt.Errorf("sendrawtransaction: %v", err)
		}
	}

	gutils.RemoteLog.PutDebugS(a.Tag, "Hash: %s", replyTxHash)

	/*if a.Coin.TestMode {
		return nil, false, errors.New("test error")
	}*/

	return &replyTxHash, false, err
}

//Check -
func (a *BitcoinAPI) Check(txHash string, fee decimal.Decimal) (bool, decimal.Decimal, error) {
	var err error
	var replyTx replyGetTransaction

	//jsonrpc1.JSONRPC1_DEBUG = true

	a.client = jsonrpcf.NewHTTPClient(a.Coin.URL)
	defer a.client.Close()

	err = a.client.Call("gettransaction", []string{txHash}, &replyTx)
	if err != nil {
		return false, fee, gutils.FormatErrorSI("gettransaction", a.logID, "%v", err)
	}

	if replyTx.Confirmations < a.Coin.B.Confirmations {
		return false, fee, gutils.FormatErrorSI("checkConfirmations", a.logID, "not enough confirmations")
	}

	return true, fee, nil
}

func (a *BitcoinAPI) getNetworkParams() *chaincfg.Params {
	networkParams := &chaincfg.MainNetParams
	networkParams.PubKeyHashAddrID = a.Coin.B.PubKeyID
	networkParams.PrivateKeyID = a.Coin.B.PrivKeyID
	networkParams.ScriptHashAddrID = a.Coin.B.ScriptID
	return networkParams
}

func (a *BitcoinAPI) createPrivateKey(params *chaincfg.Params) (*btcutil.WIF, error) {
	secret, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, err
	}
	return btcutil.NewWIF(secret, params, true)
}

func (a *BitcoinAPI) getAddressP2PKH(wif *btcutil.WIF, params *chaincfg.Params) (*btcutil.AddressPubKey, error) {
	return btcutil.NewAddressPubKey(wif.PrivKey.PubKey().SerializeCompressed(), params)
}

func (a *BitcoinAPI) getAddressP2SH(wif *btcutil.WIF, params *chaincfg.Params) (*btcutil.AddressScriptHash, error) {
	var err error

	redeemScriptS, err := a.GetRedeemScript(wif.String())
	if err != nil {
		return nil, err
	}

	redeemScriptB, _ := hex.DecodeString(redeemScriptS)

	return btcutil.NewAddressScriptHashFromHash(btcutil.Hash160(redeemScriptB), params)
}

//CreateAccount -
func (a *BitcoinAPI) CreateAccount(privateKey string) (*Account, error) {
	var err error

	/*switch a.Tag {
	case CoinBTC, CoinBCH:
	default:
		return nil, errCoinNotSupported
	}*/

	params := a.getNetworkParams()

	var wif *btcutil.WIF

	if privateKey == "" {
		wif, err = a.createPrivateKey(params)
	} else {
		wif, err = btcutil.DecodeWIF(privateKey)
	}
	if err != nil {
		return nil, err
	}

	addrPubKeyHash, err := a.getAddressP2PKH(wif, params)
	if err != nil {
		return nil, err
	}

	addrScriptHash, err := a.getAddressP2SH(wif, params)
	if err != nil {
		return nil, err
	}

	gutils.RemoteLog.PutDebugS(a.Tag, "P2PKH %s", addrPubKeyHash.EncodeAddress())
	gutils.RemoteLog.PutDebugS(a.Tag, "P2SH %s", addrScriptHash.EncodeAddress())

	acc := Account{
		//Address:    addrPubKeyHash.EncodeAddress(),
		Address:    addrScriptHash.EncodeAddress(),
		PrivateKey: wif.String(),
	}

	return &acc, nil
}

//SetServiceAccount -
func (a *BitcoinAPI) SetServiceAccount(address, privKey string) {
	a.Coin.Address = address
	a.Coin.Key = privKey
}

//GetAPIType -
func (a *BitcoinAPI) GetAPIType() string {
	return APITypeBitcoin
}
