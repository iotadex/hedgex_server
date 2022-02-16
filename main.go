package main

import (
	"hedgex-server/config"
	"hedgex-server/daemon"
	"hedgex-server/gl"
	"hedgex-server/model"
	"hedgex-server/service"
)

func main() {
	if config.Env == "product" {
		daemon.Background("./out.log", true)
	}

	//create out and err logs in logs dir
	gl.CreateLogFiles()

	//connect to mysql database
	model.ConnectToMysql()

	//init the contracts
	gl.InitContract()

	//start contract service
	service.Start()

	//wait to exit single
	daemon.WaitForKill()
}
