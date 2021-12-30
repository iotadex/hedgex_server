package service

import (
	"hedgex-server/config"
	"hedgex-server/model"
	"log"
	"sync"
)

var ServiceWaitGroup sync.WaitGroup
var QuitIndexPrice chan int
var QuitEvent map[string]chan int
var QuitExplosiveDetect chan int
var QuitInterestDetect chan int
var QuitExplosivePool chan int
var IsRunForceClose int32

func init() {
	QuitIndexPrice = make(chan int)
	QuitEvent = make(map[string]chan int)
	QuitExplosiveDetect = make(chan int)
	QuitExplosivePool = make(chan int)
	IsRunForceClose = 0
}

func Start() {
	//start index price service
	go StartIndexPriceService()

	//start listening the event of contracts
	for _, contract := range config.Contract {
		go StartFilterEvents(contract.Address)
	}

	//get users from database
	getHistoryUsersDataFromDb()

	go StartExplosiveDetectServer()

	go StartTakeInterestServer()
}

func Stop() {
	QuitExplosiveDetect <- 1
	QuitInterestDetect <- 1

	//stop the event listening service
	for _, contract := range config.Contract {
		QuitEvent[contract.Address] <- 1
	}

	//stop the indexprice service
	QuitIndexPrice <- 1

	ServiceWaitGroup.Wait()
}

func getHistoryUsersDataFromDb() {
	//load user's data from database
	for _, contract := range config.Contract {
		users, _, err := model.GetUsers(contract.Address)
		if err != nil {
			log.Panic("Get users from db error. ", err)
			return
		}
		l := len(users)
		for i := 0; i < l; i++ {
			expUserList[contract.Address].Insert(&users[i])
			interestUserList[contract.Address].update(&users[i])
		}
	}
}
