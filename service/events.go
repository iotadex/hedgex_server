package service

import (
	"context"
	"hedgex-server/config"
	"hedgex-server/gl"
	"hedgex-server/model"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

//StartFilterEvents start to filter the contract's events
func StartFilterEvents() {
	addresses := make([]common.Address, 0, len(config.Contract))
	for _, con := range config.Contract {
		addresses = append(addresses, common.HexToAddress(con))
	}
	getHistoryEventLogs(addresses)

	query := ethereum.FilterQuery{
		Addresses: addresses,
	}
StartFilter:
	eventLogChan := make(chan types.Log)
	sub, err := gl.EthWssClient.SubscribeFilterLogs(context.Background(), query, eventLogChan)
	if err != nil {
		log.Panic("Get event logs from eth wss client error. ", err)
	}
	for {
		select {
		case err := <-sub.Err():
			gl.OutLogger.Error("Event wss sub error. %v", err)
			gl.OutLogger.Warn("The EthWssClient will be redialed...")
			time.Sleep(time.Second * 10)
			gl.EthWssClient, err = ethclient.Dial(config.ChainNode.Wss)
			if err != nil {
				gl.OutLogger.Warn("The EthWssClient redial error. %v", err)
				continue
			}
			goto StartFilter
		case vLog := <-eventLogChan:
			dealEventLog(&vLog)
		}
	}
}

func getHistoryEventLogs(addresses []common.Address) {
	currBlock, err := gl.GetCurrentBlockNumber()
	if err != nil {
		log.Println("GetCurrentBlockNumber error. ", err)
		return
	}

	fromBlock := config.ChainNode.From
	if fromBlock < 24892730 {
		fromBlock = int64(currBlock) - config.ChainNode.BlockCountLimit + 10
	}

	for fromBlock < int64(currBlock) {
		toBlcok := fromBlock + config.ChainNode.BlockCountLimit
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(fromBlock),
			ToBlock:   big.NewInt(toBlcok),
			Addresses: addresses,
		}
		logs, err := gl.EthHttpsClient.FilterLogs(context.Background(), query)
		if err != nil {
			log.Println("Get event logs from eth client error. ", err)
			continue
		}
		gl.OutLogger.Info("There are %d event logs from %d to %d", len(logs), fromBlock, toBlcok)

		for _, log := range logs {
			dealEventLog(&log)
		}
		fromBlock += config.ChainNode.BlockCountLimit
	}
}

func dealEventLog(vLog *types.Log) {
	if len(vLog.Topics) > 0 {
		tx := vLog.TxHash.Hex()
		block := vLog.BlockNumber
		var err error
		var data []interface{}
		topic0 := vLog.Topics[0].Hex()
		if event, exist := gl.EventNames[topic0]; exist {
			if len(vLog.Data) > 0 {
				data, err = gl.ContractAbi.Unpack(event, vLog.Data)
				if err != nil {
					gl.OutLogger.Error("Unpack the event log error. tx(%s) : %s : %s : %v", tx, event, gl.EventNames[topic0], err)
					return
				}
			}
		}
		account := common.HexToAddress(vLog.Topics[1].Hex()).Hex()
		switch topic0 {
		case gl.MintEvent:
			amount := data[0].(*big.Int).Uint64()
			err = model.InsertMint(tx, vLog.Address.Hex(), account, amount, block)
		case gl.BurnEvent:
			amount := data[0].(*big.Int).Uint64()
			err = model.InsertBurn(tx, vLog.Address.Hex(), account, amount, block)
		case gl.RechargeEvent:
			amount := data[0].(*big.Int).Uint64()
			err = model.InsertRecharge(tx, vLog.Address.Hex(), account, amount, block)
			updateUser(vLog.Address.Hex(), account, block)
		case gl.WithdrawEvent:
			amount := data[0].(*big.Int).Uint64()
			err = model.InsertWithdraw(tx, vLog.Address.Hex(), account, amount, block)
			updateUser(vLog.Address.Hex(), account, block)
		case gl.TradeEvent:
			direction := data[0].(int8)
			amount := data[1].(*big.Int).Uint64()
			price := data[2].(*big.Int).Uint64()
			err = model.InsertTrade(tx, vLog.Address.Hex(), account, direction, amount, price, block)
			updateUser(vLog.Address.Hex(), account, block)
		case gl.ExplosiveEvent:
			direction := data[0].(int8)
			amount := data[1].(*big.Int).Uint64()
			price := data[2].(*big.Int).Uint64()
			err = model.InsertExplosive(tx, vLog.Address.Hex(), account, direction, amount, price, block)
			updateUser(vLog.Address.Hex(), account, block)
		case gl.TakeInterestEvent:
			direction := data[0].(int8)
			amount := data[1].(*big.Int).Uint64()
			price := data[2].(*big.Int).Uint64()
			err = model.InsertInterest(tx, vLog.Address.Hex(), account, direction, amount, price, block)
			updateUser(vLog.Address.Hex(), account, block)
		case gl.TransferEvent:
			gl.OutLogger.Info("Transfer from %s to %s : %v", account, vLog.Topics[2].Hex(), data[0])
		default:
			gl.OutLogger.Info("Topics : %v   ;   data  :  %v", vLog.Topics, vLog.Data)
		}
		if err != nil {
			gl.OutLogger.Error("insert into database error. %s : %v", gl.EventNames[vLog.Topics[0].Hex()], err)
		}
	}
}

func updateUser(contract string, account string, block uint64) {
	trader, err := gl.Contracts[contract].Traders(nil, common.HexToAddress(account))
	if err != nil {
		gl.OutLogger.Error("Get account's position data from blockchain error. %v", err)
		return
	}
	if h, err := gl.GetCurrentBlockNumber(); err != nil {
		gl.OutLogger.Error("Get block number error. %v", err)
	} else {
		block = h
	}
	user := model.User{
		Account:   account,
		Margin:    trader.Margin.Int64(),
		Lposition: trader.LongAmount.Uint64(),
		Lprice:    trader.LongPrice.Uint64(),
		Sposition: trader.ShortAmount.Uint64(),
		Sprice:    trader.ShortPrice.Uint64(),
		Block:     block,
	}
	if err := model.UpdateUser(contract, &user); err != nil {
		gl.OutLogger.Error("Update account's data in database error. %v", err)
	}

	//update the explosive list
	expUserList[contract].Update(&user)
	//update the takeinterest list
	interestUserList[contract].update(&user)
}
