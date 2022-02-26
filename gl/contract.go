package gl

import (
	"context"
	"crypto/ecdsa"
	"hedgex-server/config"
	"hedgex-server/contract/hedgex"
	"hedgex-server/tools"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	//define the client to connect to the ethereum network
	EthHttpsClient *ethclient.Client
	EthWssClient   *ethclient.Client

	//define the contract's abi
	ContractAbi abi.ABI

	//definde the hash string of contract's event
	MintEvent         string
	BurnEvent         string
	RechargeEvent     string
	WithdrawEvent     string
	TradeEvent        string
	ExplosiveEvent    string
	TakeInterestEvent string
	TransferEvent     string
	EventNames        map[string]string

	//contract's instance
	Contracts       map[string]*hedgex.Hedgex
	KeepMarginScale map[string]uint64

	ExplosiveAuth *bind.TransactOpts
	ExplosivePA   common.Address
	InterestAuth  *bind.TransactOpts
	InterestPA    common.Address

	chainID *big.Int
)

func InitContract() {
	var err error
	EthHttpsClient, err = ethclient.Dial(config.ChainNode.Https)
	if err != nil {
		log.Panic("ChainNode Connect Error :", config.ChainNode.Https, err)
	}

	chainID, err = EthHttpsClient.NetworkID(context.Background())
	if err != nil {
		log.Panic(err)
	}

	key := tools.InputKey()
	pk := tools.AesCBCDecrypt(config.Explosive.Wallet, key)
	ExplosiveAuth, ExplosivePA = getAccountAuth(pk)
	pk = tools.AesCBCDecrypt(config.Interest.Wallet, key)
	InterestAuth, InterestPA = getAccountAuth(pk)

	KeepMarginScale = make(map[string]uint64)
	Contracts = make(map[string]*hedgex.Hedgex)
	for i := range config.Contract {
		cont, err := hedgex.NewHedgex(common.HexToAddress(config.Contract[i]), EthHttpsClient)
		if err != nil {
			log.Panic(err)
		}
		scale, err := cont.KeepMarginScale(nil)
		if err != nil {
			log.Panic(err)
		}
		Contracts[config.Contract[i]] = cont
		KeepMarginScale[config.Contract[i]] = uint64(scale)
	}

	EthWssClient, err = ethclient.Dial(config.ChainNode.Wss)
	if err != nil {
		log.Panic(err)
	}

	ContractAbi, err = abi.JSON(strings.NewReader(string(hedgex.HedgexABI)))
	if err != nil {
		log.Panic(err)
	}

	MintEvent = crypto.Keccak256Hash([]byte(ContractAbi.Events["Mint"].Sig)).Hex()
	BurnEvent = crypto.Keccak256Hash([]byte(ContractAbi.Events["Burn"].Sig)).Hex()
	RechargeEvent = crypto.Keccak256Hash([]byte(ContractAbi.Events["Recharge"].Sig)).Hex()
	WithdrawEvent = crypto.Keccak256Hash([]byte(ContractAbi.Events["Withdraw"].Sig)).Hex()
	TradeEvent = crypto.Keccak256Hash([]byte(ContractAbi.Events["Trade"].Sig)).Hex()
	ExplosiveEvent = crypto.Keccak256Hash([]byte(ContractAbi.Events["Explosive"].Sig)).Hex()
	TakeInterestEvent = crypto.Keccak256Hash([]byte(ContractAbi.Events["TakeInterest"].Sig)).Hex()
	TransferEvent = crypto.Keccak256Hash([]byte(ContractAbi.Events["Transfer"].Sig)).Hex()

	EventNames = make(map[string]string)
	EventNames[MintEvent] = "Mint"
	EventNames[BurnEvent] = "Burn"
	EventNames[RechargeEvent] = "Recharge"
	EventNames[WithdrawEvent] = "Withdraw"
	EventNames[TradeEvent] = "Trade"
	EventNames[ExplosiveEvent] = "Explosive"
	EventNames[TakeInterestEvent] = "TakeInterest"
	EventNames[TransferEvent] = "Transfer"
}

func getAccountAuth(pk string) (*bind.TransactOpts, common.Address) {
	var err error
	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		log.Panic("Get privatekey error.", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Panic("error casting public key to ECDSA")
	}
	publicAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		panic(err)
	}
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(300000)
	return auth, publicAddress
}

func GetGasPriceAndNonce(minGasPrice int64, auth *bind.TransactOpts, pa common.Address) error {
	gasPrice, err := EthHttpsClient.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	if gasPrice.Int64() < minGasPrice {
		gasPrice = big.NewInt(minGasPrice)
	}
	auth.GasPrice = gasPrice
	nonce, err := EthHttpsClient.PendingNonceAt(context.Background(), pa)
	if err != nil {
		return err
	}
	auth.Nonce = big.NewInt(int64(nonce))
	return nil
}

func GetTransaction(txHash common.Hash) (*types.Receipt, error) {
	return EthHttpsClient.TransactionReceipt(context.Background(), txHash)
}

func Explosive(auth *bind.TransactOpts, contract string, account string) (*types.Transaction, error) {
	return Contracts[contract].Explosive(auth, common.HexToAddress(account), common.HexToAddress(config.Explosive.ToAddress))
}

func DetectSlide(auth *bind.TransactOpts, add string, account string) (*types.Transaction, error) {
	return Contracts[add].DetectSlide(auth, common.HexToAddress(account), common.HexToAddress(config.Interest.ToAddress))
}

func ExplosivePool(auth *bind.TransactOpts, contract string) (*types.Transaction, error) {
	return Contracts[contract].ExplosivePool(auth)
}

func ForceClose(auth *bind.TransactOpts, contract string, account string) (*types.Transaction, error) {
	return Contracts[contract].ForceCloseAccount(auth, common.HexToAddress(account), common.HexToAddress(config.Explosive.ToAddress))
}

func GetIndexPrice(add string) (int64, error) {
	price, _, _, err := Contracts[add].GetLatestPrice(nil)
	if err != nil {
		return 0, err
	}
	return price.Int64(), err
}

func GetPoolPosition(add string) (int64, int64, int64, int64, int64, uint8, error) {
	_total, _lp, _lprice, _sp, _sprice, _state, err := Contracts[add].GetPoolPosition(nil)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	return _total.Int64(), _lp.Int64(), _lprice.Int64(), _sp.Int64(), _sprice.Int64(), _state, nil
}

func GetPoolState(add string) (uint8, error) {
	return Contracts[add].PoolState(nil)
}

func GetCurrentBlockNumber() (uint64, error) {
	return EthHttpsClient.BlockNumber(context.Background())
}

func GetTransactionByHash(txHash common.Hash) {
	EthHttpsClient.TransactionReceipt(context.Background(), txHash)
}
