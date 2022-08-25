package xssdemo

import (
	"log"

	"go.uber.org/zap"
)

var logger *log.Logger

func init() {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Printf("failed NewProduction: %s", err)
	}

	logger = zap.NewStdLog(zapLogger)
}
