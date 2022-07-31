package main

import (
	"compare/log"
)

func main() {
	version := "2.0.0"
	e := Create()
	err := e.Execute(version)
	if err != nil {
		panic(err)
	}

	log.Info("Done")
}
