package main

import (
	"compare/core"
	"compare/jobs"
	"compare/log"
	"compare/workers"
	"fmt"
	"sync"
)

type EntryPoint struct {
	jobs    *jobs.Jobs
	workers *workers.Workers
}

func Create() *EntryPoint {
	return &EntryPoint{}
}

func (e *EntryPoint) Execute(version string) error {
	core.InitFlags()
	log.InitFlags()
	log.ParseFlags()

	log.Info("Starting - " + version)

	err := e.buildBLocks()
	if err != nil {
		return fmt.Errorf("failed to build blocks: %v", err)
	}

	var wg sync.WaitGroup

	go e.workers.Execute(&wg, e.jobs.Pool)
	e.jobs.GenerateJobs()

	go e.workers.PrintProgress()

	for r := range e.workers.Done {
		log.V5(fmt.Sprintf("%v Done", r))
	}

	return nil
}

func (e *EntryPoint) buildBLocks() error {
	var err error
	e.jobs, err = jobs.Create()
	if err != nil {
		return fmt.Errorf("failed to create Jobs: %v", err)
	}

	e.workers = workers.Create(core.Config.Workers)
	return nil
}
