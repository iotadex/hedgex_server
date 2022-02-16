package service

import (
	"hedgex-server/config"
	"hedgex-server/model"
	"log"
)

func Start() {
	//start index price service
	go StartIndexPriceService()

	//start listening the event of contracts
	for _, contract := range config.Contract {
		go StartFilterEvents(contract)
	}

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
