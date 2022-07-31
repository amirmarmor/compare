package core

import (
	"compare/log"
	"encoding/json"
	"github.com/namsral/flag"
)

type Configuration = struct {
	Verbose    int
	ZapLogger  bool
	Workers    int
	TargetPath string
	SourcePath string
	ResultPath string
}

var Config Configuration

func InitFlags() {
	flag.IntVar(&Config.Workers, "workers", 3, "number of workers to use")
	flag.StringVar(&Config.TargetPath, "target", "", "directory for target files")
	flag.StringVar(&Config.SourcePath, "source", "", "directory for source files")
	flag.StringVar(&Config.ResultPath, "result", "", "directory for result files")

	log.AddNotify(PostParse)
}

func PostParse() {
	marshal, err := json.Marshal(Config)
	if err != nil {
		log.Fatal("marshal config failed: %v", err)
	}

	log.V5("V5 mode activated")
	log.V5("common configuration loaded: %v", string(marshal))
}
