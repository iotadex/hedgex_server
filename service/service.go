package service

import (
	"container/list"
	"hedgex-server/config"
	"hedgex-server/model"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

func Start() {
	//start index price service
	go StartIndexPriceService()

	//start listening the event of contracts
	go StartFilterEvents()

	//get users from database
	getHistoryUsersDataFromDb()

	//go StartExplosiveDetectServer()

	//go StartTakeInterestServer()
}

func getHistoryUsersDataFromDb() {
	//load user's data from database
	for _, contract := range config.Contract {
		users, _, err := model.GetUsers(contract)
		if err != nil {
			log.Panic("Get users from db error. ", err)
			return
		}
		l := len(users)
		for i := 0; i < l; i++ {
			expUserList[contract].Insert(&users[i])
			interestUserList[contract].update(&users[i])
		}
	}
}

type txNode struct {
	tx   *types.Transaction
	auth *bind.TransactOpts
}

type TxList struct {
	txList list.List
	mu     sync.Mutex
}

func StartDetectTransactions() {

}
