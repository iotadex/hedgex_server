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

type chainNode struct {
	Https           string `json:"http"`
	Wss             string `json:"ws"`
	From            int64  `json:"from"`
	BlockCountLimit int64  `json:"block_count_limit"`
}

type explosive struct {
	Start       bool          `json:"start"`
	Tick        time.Duration `json:"tick"`
	GasPriceUp  float64       `json:"gas_price_up"`
	GasPriceMin int64         `json:"gas_price_min"`
	ToAddress   string        `json:"to_address"`
	Wallet      string        `json:"wallet"`
}

type interest struct {
	Start       bool          `json:"start"`
	Tick        time.Duration `json:"tick"`
	Begin       int64         `json:"begin"`
	End         int64         `json:"end"`
	GasPriceUp  float64       `json:"gas_price_up"`
	GasPriceMin int64         `json:"gas_price_min"`
	ToAddress   string        `json:"to_address"`
	Wallet      string        `json:"wallet"`
}

var (
	Env       string
	Db        db
	Explosive explosive
	Interest  interest
	ChainNode chainNode
	Contract  []string
)

//Load load config file
func init() {
	file, err := os.Open("config/config.json")
	if err != nil {
		log.Panic(err)
	}
	defer file.Close()
	type Config struct {
		Env       string    `json:"env"`
		Explosive explosive `json:"explosive"`
		Interest  interest  `json:"interest"`
		Db        db        `json:"db"`
		ChainNode chainNode `json:"chain_node"`
		Contract  []string  `json:"contract"`
	}
	all := &Config{}
	if err = json.NewDecoder(file).Decode(all); err != nil {
		log.Panic(err)
	}
	Env = all.Env
	Db = all.Db
	Explosive = all.Explosive
	Interest = all.Interest
	ChainNode = all.ChainNode
	Contract = all.Contract
	if Interest.Begin >= Interest.End {
		log.Panic("interest.begin must < interest.end")
	}
}
