package service

import (
	"hedgex-server/config"
	"hedgex-server/gl"
	"sync/atomic"
	"time"
)

var IndexPrices map[string]*int64

func init() {
	IndexPrices = make(map[string]*int64)
	for i := range config.Contract {
		IndexPrices[config.Contract[i]] = new(int64)
	}
}

func StartIndexPriceService() {
	ticker := time.NewTicker(time.Second * config.IndexTick)
	for range ticker.C {
		for i := range config.Contract {
			if price, err := gl.GetIndexPrice(config.Contract[i]); err != nil {
				gl.OutLogger.Error("Get price from contract error. %v", err)
			} else {
				atomic.StoreInt64(IndexPrices[config.Contract[i]], price)
			}
		}
	}
}
