package workers

import (
	"compare/jobs"
	"compare/log"
	"fmt"
	"sync"
)

type Workers struct {
	workersPool []*Worker
	Done        chan string
	Progress    chan string
}

func Create(poolSize int) *Workers {
	workers := &Workers{
		workersPool: make([]*Worker, 0),
		Done:        make(chan string),
		Progress:    make(chan string),
	}

	for i := 0; i < poolSize; i++ {
		worker := CreateWorker(i, workers.Progress)
		workers.workersPool = append(workers.workersPool, worker)
	}

	return workers
}

func (w *Workers) Execute(wg *sync.WaitGroup, jobsChan chan *jobs.Job) {
	for _, worker := range w.workersPool {
		wg.Add(1)
		worker.Execute(wg, jobsChan, w.Done)
	}

	defer close(w.Done)
	wg.Wait()
}

func (w *Workers) PrintProgress() {
	for {
		select {
		case line := <-w.Progress:
			log.V5(fmt.Sprintf("%v", line))
		}
	}
}
