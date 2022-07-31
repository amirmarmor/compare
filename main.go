package main

import (
	"compare/log"
)

func main() {
	e := Create()
	err := e.Execute()
	if err != nil {
		panic(err)
	}

	log.Info("Done")
}
