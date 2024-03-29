package service

import (
	"hedgex-server/config"
	"hedgex-server/gl"
	"hedgex-server/model"
	"sync"
	"time"
)

var interestUserList map[string]*TakeInterestList

//StartExplosiveDetectServer, no blocking function
func StartTakeInterest() {
	for {
		ts := time.Now().Unix()
		dayCount := ts / 86400
		offset := ts - dayCount*86400
		if offset <= config.Interest.Begin || offset > config.Interest.End {
			sleepTime := (86400 - offset) / 2
			if offset <= config.Interest.Begin {
				sleepTime = config.Interest.Begin - offset + 1
			} else if sleepTime < int64(config.Interest.Tick) {
				sleepTime = int64(config.Interest.Tick)
			}
			time.Sleep(time.Duration(sleepTime) * time.Second)
			continue
		}

		gl.OutLogger.Info("Start take interests. %v", time.Now())
		//start taking interests
		for i := range config.Contract {
			takeInterest(uint(dayCount), config.Contract[i])
		}
		time.Sleep(time.Second * time.Duration(config.Interest.End-config.Interest.Begin))
	}
}

func takeInterest(dayCount uint, conAdd string) {
	//get the pool's position
	_, lp, _, sp, _, _, err := gl.GetPoolPosition(conAdd)
	if err != nil {
		gl.OutLogger.Error("Get account's position data from blockchain error. %s : %v", conAdd, err)
		time.Sleep(time.Second)
		return
	}
	til := interestUserList[conAdd]
	var l map[string]*interestUser
	if lp > sp {
		l = til.getShortUsers()
	} else if lp < sp {
		l = til.getLongUsers()
	}

	for k, v := range l {
		if v.day < dayCount {
			err := gl.GetGasPriceAndNonce(config.Interest.GasPriceMin, gl.InterestAuth, gl.InterestPA)
			if err != nil {
				gl.OutLogger.Error("Get auth error when take interest. %v", err)
				continue
			}

			if tx, err := gl.DetectSlide(gl.InterestAuth, conAdd, k); err == nil {
				gl.OutLogger.Info("send interest over. %s : %s", k, tx.Hash().Hex())
				til.flag(k, dayCount)
			} else {
				gl.OutLogger.Error("send interest error. %s : %s : %v", conAdd, k, err)
			}
		}
	}
}

type interestUser struct {
	block uint64
	day   uint
}

type TakeInterestList struct {
	luser map[string]*interestUser // long position user
	suser map[string]*interestUser // short position user
	mu    sync.Mutex               //user's locker
}

func (til *TakeInterestList) flag(account string, day uint) {
	til.mu.Lock()
	defer til.mu.Unlock()
	if _, exist := til.luser[account]; exist {
		til.luser[account].day = day
	} else if _, exist := til.suser[account]; exist {
		til.suser[account].day = day
	}
}

func (til *TakeInterestList) update(u *model.User) {
	til.mu.Lock()
	defer til.mu.Unlock()
	v, exist := til.luser[u.Account]
	if !exist {
		v, exist = til.suser[u.Account]
	}
	if exist && (v.block > u.Block) {
		return
	}
	if u.Lposition > u.Sposition {
		til.luser[u.Account] = &interestUser{u.Block, u.InterestDay}
		delete(til.suser, u.Account)
	} else if u.Lposition < u.Sposition {
		til.suser[u.Account] = &interestUser{u.Block, u.InterestDay}
		delete(til.luser, u.Account)
	} else {
		delete(til.luser, u.Account)
		delete(til.suser, u.Account)
	}
}

func (til *TakeInterestList) getShortUsers() map[string]*interestUser {
	su := make(map[string]*interestUser)
	til.mu.Lock()
	defer til.mu.Unlock()
	m := til.suser
	for k, v := range m {
		su[k] = v
	}
	return su
}

func (til *TakeInterestList) getLongUsers() map[string]*interestUser {
	lu := make(map[string]*interestUser)
	til.mu.Lock()
	defer til.mu.Unlock()
	m := til.luser
	for k, v := range m {
		lu[k] = v
	}
	return lu
}
