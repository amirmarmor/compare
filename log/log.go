package log

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"time"
)

var zapLogger *zap.Logger

func initLogger() {
	configuration := zap.NewProductionConfig()
	configuration.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	configuration.EncoderConfig.CallerKey = ""
	var err error
	zapLogger, err = configuration.Build()
	if err != nil {
		panic(err)
	}
}

func logWrite(level string, format string, v ...interface{}) {
	if Config.ZapLogger {
		if zapLogger == nil {
			initLogger()
		}
		message := fmt.Sprintf(format, v...)
		switch level {
		case "VERB":
			zapLogger.Debug(message)
			break
		case "INFO":
			zapLogger.Info(message)
			break
		case "WARN":
			zapLogger.Warn(message)
			break
		case "ERROR":
			zapLogger.Error(message)
			break
		case "FATAL":
			zapLogger.Fatal(message)
			break
		default:
			zapLogger.Info(message)
		}
	} else {
		formattedTimestamp := time.Now().UTC().Format("2006-01-02 15:04:05.000")
		updatedFormat := fmt.Sprintf("%v %v: %v\n", formattedTimestamp, level, format)
		fmt.Printf(updatedFormat, v...)
	}
}

func Error(format string, v ...interface{}) {
	logWrite("ERROR", format, v...)
	os.Exit(1)
}

func Fatal(format string, v ...interface{}) {
	logWrite("FATAL", format, v...)
	os.Exit(1)
}

func Warn(format string, v ...interface{}) {
	logWrite("WARN", format, v...)
}

func Info(format string, v ...interface{}) {
	logWrite("INFO", format, v...)
}

func V1(format string, v ...interface{}) {
	if Config.Verbose < 1 {
		return
	}
	logWrite("VERB", format, v...)
}

func V2(format string, v ...interface{}) {
	if Config.Verbose < 2 {
		return
	}
	logWrite("VERB", format, v...)
}

func V5(format string, v ...interface{}) {
	if Config.Verbose < 5 {
		return
	}
	logWrite("VERB", format, v...)
}
