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
	Contracts map[string]*hedgex.Hedgex

	//
	privateKey    *ecdsa.PrivateKey
	PublicAddress common.Address

	erc20TransferID []byte
	chainID         *big.Int
)

func InitContract() {
	var err error
	EthHttpsClient, err = ethclient.Dial(config.ChainNode.Https)
	if err != nil {
		log.Panic("ChainNode : ", config.ChainNode.Https, err)
	}
	block, _ := EthHttpsClient.BlockByNumber(context.Background(), big.NewInt(100000))
	block.Time()

	Contracts = make(map[string]*hedgex.Hedgex)
	for i := range config.Contract {
		Contracts[config.Contract[i].Address], err = hedgex.NewHedgex(common.HexToAddress(config.Contract[i].Address), EthHttpsClient)
		if err != nil {
			log.Panic(err)
		}
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

	erc20TransferID = []byte{0xa9, 0x05, 0x9c, 0xbb} //transfer(address,uint256)

	chainID, err = EthHttpsClient.NetworkID(context.Background())
	if err != nil {
		log.Panic(err)
	}

	key := tools.InputKey()
	pk := tools.AesCBCDecrypt(config.PrivateKey, key)
	SetPrivateKey(pk)
}

func SetPrivateKey(pk string) {
	var err error
	privateKey, err = crypto.HexToECDSA(pk)
	if err != nil {
		log.Panic("Get privatekey error.", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Panic("error casting public key to ECDSA")
	}
	PublicAddress = crypto.PubkeyToAddress(*publicKeyECDSA)
}

func GetAccountAuth() (*bind.TransactOpts, error) {
	gasPrice, err := EthHttpsClient.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	chainID, err := EthHttpsClient.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID) // bind.NewKeyedTransactor(privateKey)
	if err != nil {
		return nil, err
	}
	auth.Value = big.NewInt(0)     // in wei
	auth.GasLimit = uint64(300000) // in units
	auth.GasPrice = gasPrice
	return auth, nil
}

func SendEth(to string, amountS string) error {
	value, _ := new(big.Int).SetString(amountS, 10)
	return sendTransaction(common.HexToAddress(to), value, nil)
}

func SendERC20(token string, to string, amountS string) error {
	paddedAddress := common.LeftPadBytes(common.HexToAddress(to).Bytes(), 32) // 0x0000000000000000000000004592d8f8d7b001e72cb26a73e4fa1806a51ac79d
	amount, _ := new(big.Int).SetString(amountS, 10)                          // amount
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)                   // 0x00000000000000000000000000000000000000000000003635c9adc5dea00000

	var data []byte
	data = append(data, erc20TransferID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)
	value := big.NewInt(0) // in wei (0 eth)
	return sendTransaction(common.HexToAddress(token), value, data)
}

func SendTransaction(to common.Address, value *big.Int, data []byte) error {
	gasPrice, err := EthHttpsClient.SuggestGasPrice(context.Background())
	if err != nil {
		OutLogger.Error("get gas price error. %v", err)
		return err
	}
	nonce, err := EthHttpsClient.PendingNonceAt(context.Background(), PublicAddress)
	if err != nil {
		OutLogger.Error("get nonce error address(%s). %v", PublicAddress, err)
		return err
	}
	gasLimit := uint64(3000000)
	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		OutLogger.Error("create signedTx error. %v", err)
		return err
	}

	err = EthHttpsClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		OutLogger.Error("send signedTx error. %v", err)
		return err
	}
	return nil
}

func sendTransaction(to common.Address, value *big.Int, data []byte) error {
	gasPrice, err := EthHttpsClient.SuggestGasPrice(context.Background())
	if err != nil {
		OutLogger.Error("get gas price error. %v", err)
		return err
	}
	nonce, err := EthHttpsClient.PendingNonceAt(context.Background(), PublicAddress)
	if err != nil {
		OutLogger.Error("get nonce error address(%s). %v", PublicAddress, err)
		return err
	}
	gasLimit := uint64(3000000)
	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		OutLogger.Error("create signedTx error. %v", err)
		return err
	}

	err = EthHttpsClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		OutLogger.Error("send signedTx error. %v", err)
		return err
	}
	return nil
}

func Explosive(auth *bind.TransactOpts, contract string, account string) error {
	nonce, err := EthHttpsClient.PendingNonceAt(context.Background(), PublicAddress)
	if err != nil {
		return err
	}
	auth.Nonce = big.NewInt(int64(nonce))
	_, err = Contracts[contract].Explosive(auth, common.HexToAddress(account), common.HexToAddress(config.Explosive.ToAddress))
	return err
}

func DetectSlide(auth *bind.TransactOpts, add string, account string) error {
	nonce, err := EthHttpsClient.PendingNonceAt(context.Background(), PublicAddress)
	if err != nil {
		OutLogger.Error("Take interest : Get nonce error address(%s). %v", PublicAddress, err)
		return err
	}
	auth.Nonce = big.NewInt(int64(nonce))
	if _, err := Contracts[add].DetectSlide(auth, common.HexToAddress(account), common.HexToAddress(config.Interest.ToAddress)); err != nil {
		OutLogger.Error("Transaction with detect slide error. %v", err)
		return err
	}
	return nil
}

func ExplosivePool(auth *bind.TransactOpts, contract string) error {
	nonce, err := EthHttpsClient.PendingNonceAt(context.Background(), PublicAddress)
	if err != nil {
		return err
	}
	auth.Nonce = big.NewInt(int64(nonce))
	_, err = Contracts[contract].ExplosivePool(auth)
	return err
}

func ForceClose(auth *bind.TransactOpts, contract string, account string) error {
	nonce, err := EthHttpsClient.PendingNonceAt(context.Background(), PublicAddress)
	if err != nil {
		return err
	}
	auth.Nonce = big.NewInt(int64(nonce))
	_, err = Contracts[contract].ForceCloseAccount(auth, common.HexToAddress(account), common.HexToAddress(config.Explosive.ToAddress))
	return err
}

func GetIndexPrice(add string) (int64, error) {
	price, err := Contracts[add].GetLatestPrice(nil)
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
