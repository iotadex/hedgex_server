package service

import (
	"hedgex-server/config"
	"hedgex-server/gl"
	"time"
)

func StartPoolExplosiveService() {
	//get the pool's data
	ticker := time.NewTicker(time.Second * 600)
	for range ticker.C {
		for i := range config.Contract {
			total, lp, lpr, sp, spr, state, err := gl.GetPoolPosition(config.Contract[i])
			if err != nil {
				gl.OutLogger.Error("Get pool's position from contract error. ", err)
				continue
			}

			//get the current price of contract
			price, err := gl.GetIndexPrice(config.Contract[i])
			if err != nil {
				gl.OutLogger.Error("Get price from contract error. %s : %v", config.Contract[i], err)
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

			err = gl.GetGasPriceAndNonce(config.Interest.GasPriceMin, gl.InterestAuth, gl.InterestPA)
			if err != nil {
				gl.OutLogger.Error("Get auth error when explosive pool. %v", err)
				continue
			}
			if _, err := gl.ExplosivePool(gl.InterestAuth, config.Contract[i]); err != nil {
				gl.OutLogger.Error("send explosive pool to contract error. %s : %v", config.Contract[i], err)
			}
		}
	}
}
