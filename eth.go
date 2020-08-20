package libschain

// docs: https://goethereumbook.org/zh

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"regexp"
)

type ethLib struct {
	client  *ethclient.Client
	priKey  *ecdsa.PrivateKey
	chainID *big.Int
}

type ethAccount struct {
	Address string
	Private string
}

func NewEthLib() *ethLib {
	return &ethLib{}
}

// 连接
func (e *ethLib) Connect(address string) error {
	c, err := ethclient.Dial(address)
	if err != nil {
		return err
	}
	e.client = c
	return nil
}

// 获取连接
func (e *ethLib) GetClient() *ethclient.Client {
	return e.client
}

// set chainId
func (e *ethLib) SetChainID(id int64) {
	e.chainID = big.NewInt(id)
}

// get chainId
func (e *ethLib) GetChainID() *big.Int {

	if e.chainID != nil {
		return e.chainID
	}
	if e.client == nil {
		return nil
	}

	id, err := e.client.NetworkID(context.Background())
	if err != nil {
		return nil
	}
	e.chainID = id
	return e.chainID
}

// 获取默认私钥
func (e *ethLib) GetDefaultPriKey() string {
	return crypto.PubkeyToAddress(e.priKey.PublicKey).Hex()
}

// 设置默认私钥
func (e *ethLib) SetDefaultPriKey(priKey string) error {
	k, err := crypto.HexToECDSA(priKey)
	if err != nil {
		return err
	}
	e.priKey = k
	return nil
}

// 根据私钥获取地址
func (e *ethLib) GetAddrFromPriKey(priKey string) (address string, err error) {
	k, err := crypto.HexToECDSA(priKey)
	if err != nil {
		return "", err
	}

	pubKey := k.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	return crypto.PubkeyToAddress(*pubKeyECDSA).Hex(), nil
}

// 获取额度
func (e *ethLib) GetBalance(address string) (*big.Int, error) {
	if e.client == nil {
		return nil, errors.New("server is not connected")
	}
	account := common.HexToAddress(address)
	balance, err := e.client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

// 创建账号 - 返回地址和私钥
func (e *ethLib) CreateAccount() (*ethAccount, error) {
	//Create an account
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	//Get the address
	address := crypto.PubkeyToAddress(key.PublicKey).Hex()

	//Get the private key
	privateKey := hex.EncodeToString(key.D.Bytes())

	return &ethAccount{Address: address, Private: privateKey}, nil
}

// 校验地址
func (e *ethLib) IsValidAddress(address string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(address)
}

// 检测是否为合约地址
func (e *ethLib) IsContract(address string) (bool, error) {
	if e.client == nil {
		return false, errors.New("server is not connected")
	}
	addr := common.HexToAddress(address)
	byteCode, err := e.client.CodeAt(context.Background(), addr, nil)
	if err != nil {
		return false, err
	}
	return len(byteCode) > 0, nil
}

// 交易 - 使用默认私钥
func (e *ethLib) Transfer(toAddress string, weiAmount *big.Int) (txHash string, err error) {
	return e.transferViaPriKey(e.priKey, toAddress, weiAmount)
}

// 交易 - 使用指定私钥
func (e *ethLib) TransferUsePriKey(priKey string, toAddress string, weiAmount *big.Int) (txHash string, err error) {
	k, err := crypto.HexToECDSA(priKey)
	if err != nil {
		return "", err
	}
	return e.transferViaPriKey(k, toAddress, weiAmount)
}

// 发送原始交易
func (e *ethLib) SendRawTX(rawTx string) (txHash string, err error) {

	if e.client == nil {
		return "", errors.New("server is not connected")
	}

	rawTxBytes, err := hex.DecodeString(rawTx)

	tx := new(types.Transaction)
	rlp.DecodeBytes(rawTxBytes, &tx)

	if err = e.client.SendTransaction(context.Background(), tx); err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}

// 生成原始交易 - 简单
func (e *ethLib) GenRawTxSimple(priKey string, toAddr string, wei *big.Int) (rawTX string, err error) {

	if e.client == nil {
		return "", errors.New("server is not connected")
	}
	client := e.client

	// 1. gasLimit gasPrice
	gasLimit := uint64(21000) // in units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}

	// 2. chainId
	chainID := e.GetChainID()

	// 3. return
	var data []byte

	return e.GenRawTxData(priKey, toAddr, wei, gasLimit, gasPrice, chainID, data)
}

// 生成原始交易 - 复杂
func (e *ethLib) GenRawTxData(priKey string, toAddr string, wei *big.Int, gasLimit uint64, gasPrice *big.Int, chainID *big.Int, data []byte) (rawTX string, err error) {

	if e.client == nil {
		return "", errors.New("server is not connected")
	}
	client := e.client

	// 1. 私钥
	privateKey, err := crypto.HexToECDSA(priKey)
	if err != nil {
		return "", err
	}

	// 2. 公钥
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	// 3. nonce
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", err
	}

	// 4.
	toAddress := common.HexToAddress(toAddr)
	tx := types.NewTransaction(nonce, toAddress, wei, gasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", err
	}

	ts := types.Transactions{signedTx}
	rawTxBytes := ts.GetRlp(0)
	rawTxHex := hex.EncodeToString(rawTxBytes)

	return rawTxHex, nil
}

func (e *ethLib) transferViaPriKey(priKey *ecdsa.PrivateKey, toAddress string, weiAmount *big.Int) (txHash string, err error) {
	if e.client == nil {
		return "", errors.New("server is not connected")
	}

	publicKey := priKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	from := crypto.PubkeyToAddress(*publicKeyECDSA)
	to := common.HexToAddress(toAddress)

	nonce, err := e.client.PendingNonceAt(context.Background(), from)
	if err != nil {
		return "", err
	}

	// gasPrice := big.NewInt(30000000000) // in wei (30 gwei)
	gasPrice, err := e.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}
	gasLimit := uint64(21000) // ETH转账的燃气应设上限为21000单位。

	tx := types.NewTransaction(nonce, to, weiAmount, gasLimit, gasPrice, nil)

	chainID := e.GetChainID()

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), priKey)
	if err != nil {
		return "", err
	}

	// SendTransaction
	err = e.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}

	return signedTx.Hash().Hex(), nil
}

// 签名
func (e *ethLib) GenSign(priKey string, data []byte) (signature string, err error) {
	privateKey, err := crypto.HexToECDSA(priKey)
	if err != nil {
		return "", err
	}

	hash := crypto.Keccak256Hash(data)

	sign, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(sign), nil
}

// 查询区块头 - number = nil 查询最新区块的头信息
func (e *ethLib) GetBlockHeader(number *big.Int) (*types.Header, error) {

	header, err := e.client.HeaderByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	return header, nil
}

// 查询区块 - number = nil 查询最新区块
func (e *ethLib) GetBlock(number *big.Int) (*types.Block, error) {

	block, err := e.client.BlockByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	return block, nil
}