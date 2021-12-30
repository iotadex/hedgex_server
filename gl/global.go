package gl

import (
	"hedgex-server/logger"
	"log"
	"os"
)

type ErrorCode int

const (
	DATABASE_ERROR ErrorCode = 10001
)

// OutLogger global logger
var OutLogger *logger.Logger

func CreateLogFiles() {
	var err error
	// 初始化全局日志文件
	if err = os.MkdirAll("./logs", os.ModePerm); err != nil {
		log.Panic("Create dir './logs' error. " + err.Error())
	}
	if OutLogger, err = logger.New("logs/out.log", 1, 3, 0); err != nil {
		log.Panic("Create Outlogger file error. " + err.Error())
	}
}
