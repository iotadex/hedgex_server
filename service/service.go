package service

import (
	"hedgex-server/config"
	"hedgex-server/gl"
	"hedgex-server/model"
	"log"
)

func InitUsers() {
	expUserList = make(map[string]*ExplosiveList)
	for i := range config.Contract {
		scale := gl.KeepMarginScale[config.Contract[i]]
		expUserList[config.Contract[i]] = NewExplosiveList(scale)
	}

	interestUserList = make(map[string]*TakeInterestList)
	for _, contract := range config.Contract {
		interestUserList[contract] = &TakeInterestList{
			luser: make(map[string]*interestUser),
			suser: make(map[string]*interestUser),
		}
	}
}

func Start() {
	//start listening the event of contracts
	go StartFilterEvents()

	//get users from database
	getHistoryUsersDataFromDb()

	if config.Explosive.Start {
		go StartExplosiveDetect()
	}

	if config.Interest.Start {
		go StartTakeInterest()
	}
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
