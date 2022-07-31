package log

import (
	"encoding/json"
	"github.com/namsral/flag"
)

type PostParseNotify func()

var notifications []PostParseNotify

type Configuration = struct {
	Verbose   int
	ZapLogger bool
}

var Config Configuration

func AddNotify(notify PostParseNotify) {
	notifications = append(notifications, notify)
}

func ParseFlags() {
	flag.Parse()

	for i := 0; i < len(notifications); i++ {
		notify := notifications[i]
		notify()
	}
}

func InitFlags() {
	flag.IntVar(&Config.Verbose, "verbose", 5, "print verbose information 0=nothing 5=all")
	flag.BoolVar(&Config.ZapLogger, "zap-logger", true, "use zap logger")

	AddNotify(PostParse)
}

func PostParse() {
	initLogger()
	marshal, err := json.Marshal(Config)
	if err != nil {
		Fatal("marshal config failed: %v", err)
	}

	V5("V5 mode activated")
	V5("common configuration loaded: %v", string(marshal))
}
