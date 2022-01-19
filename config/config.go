package config

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

//db database config
type db struct {
	LifeTime  time.Duration `json:"lifetime"`
	OpenConns int           `json:"openconns"`
	IdleConns int           `json:"idleconns"`
	Host      string        `json:"host"`
	Port      string        `json:"port"`
	DbName    string        `json:"dbname"`
	Usr       string        `json:"usr"`
	Pwd       string        `json:"pwd"`
}

type hedgex struct {
	Address    string `json:"address"`
	MarginCoin string `json:"margin_coin"`
	TradeCoin  string `json:"trade_coin"`
}

type chainNode struct {
	Https string `json:"http"`
	Wss   string `json:"ws"`
	From  int64  `json:"from"`
	To    int64  `json:"to"`
}

type explosive struct {
	Tick      time.Duration `json:"tick"`
	ToAddress string        `json:"to_address"`
}

type interest struct {
	Tick      time.Duration `json:"tick"`
	Begin     int64         `json:"begin"`
	End       int64         `json:"end"`
	ToAddress string        `json:"to_address"`
}

var (
	Env        string
	Db         db
	Explosive  explosive
	Interest   interest
	IndexTick  time.Duration
	ChainNode  chainNode
	Contract   []hedgex
	PrivateKey string
)

//Load load config file
func init() {
	file, err := os.Open("config/config.json")
	if err != nil {
		log.Panic(err)
	}
	defer file.Close()
	type Config struct {
		Env         string        `json:"env"`
		Explosive   explosive     `json:"explosive"`
		Interest    interest      `json:"interest"`
		ExplosiveTo string        `json:"explosive_to_address"`
		IndexTick   time.Duration `json:"index_tick"`
		Db          db            `json:"db"`
		ChainNode   chainNode     `json:"chain_node"`
		Contract    []hedgex      `json:"contract"`
		PrivateKey  string        `json:"wallet"`
	}
	all := &Config{}
	if err = json.NewDecoder(file).Decode(all); err != nil {
		log.Panic(err)
	}
	Env = all.Env
	Db = all.Db
	IndexTick = all.IndexTick
	Explosive = all.Explosive
	Interest = all.Interest
	ChainNode = all.ChainNode
	Contract = all.Contract
	PrivateKey = all.PrivateKey
	if Interest.Begin >= Interest.End {
		log.Panic("interest.begin must < interest.end")
	}
}
