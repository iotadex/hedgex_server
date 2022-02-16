package service

import (
	"hedgex-server/config"
	"hedgex-server/gl"
	"sync/atomic"
	"time"
)

func StartPoolExplosiveService() {
	//get the pool's data
	ticker := time.NewTicker(time.Second * config.IndexTick)
	for range ticker.C {
		for i := range config.Contract {
			total, lp, lpr, sp, spr, state, err := gl.GetPoolPosition(config.Contract[i])
			if err != nil {
				gl.OutLogger.Error("Get pool's position from contract error. ", err)
				continue
			}
			var price int64
			if price = atomic.LoadInt64(IndexPrices[config.Contract[i]]); price == 0 {
				gl.OutLogger.Warn("Get index price error. %s", config.Contract[i])
				continue
			}
			poolNet := total + lp*(price-lpr) + sp*(spr-price)
			keep := ((lp - sp) * price) / 5
			if keep < 0 {
				keep = -keep
			}
			if (poolNet > keep) || state != 1 {
				continue
			}

			auth, err := gl.GetAccountAuth()
			if err != nil {
				gl.OutLogger.Error("Get auth error when explosive pool. %v", err)
				continue
			}
			if err := gl.ExplosivePool(auth, config.Contract[i]); err != nil {
				gl.OutLogger.Error("send explosive pool to contract error. %s : %v", config.Contract[i], err)
			}
		}
	}
}
