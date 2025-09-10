package main

import (
	"sync"

	"github.com/google/uuid"
)

type WorkerId = uuid.UUID

type WorkerRequest = any
type WorkerResult = any

type Worker[Request WorkerRequest, Result WorkerResult] interface {
	GetId() WorkerId
	Run(request Request) (Result, error)
}

type WorkerProvider[Request WorkerRequest, Result WorkerResult] interface {
	Create() Worker[Request, Result]
}

type WorkerPoolOptions struct {
	PoolSize int
}

type WorkerPool[Request WorkerRequest, Result WorkerResult] struct {
	provider WorkerProvider[Request, Result]
	options  WorkerPoolOptions
	requests chan Request
	wg       sync.WaitGroup
	results  chan Result
	errors   chan error
}

func (pool *WorkerPool[Request, Result]) start() {
	for i := 0; i < pool.options.PoolSize; i++ {
		pool.wg.Add(1)

		worker := pool.provider.Create()

		go func() {
			defer pool.wg.Done()

			for request := range pool.requests {
				result, err := worker.Run(request)
				if err != nil {
					pool.errors <- err
					continue
				}

				pool.results <- result
			}
		}()
	}
}

func (pool *WorkerPool[Request, Result]) Shutdown() {
	close(pool.requests)
	pool.wg.Wait()
	close(pool.results)
	close(pool.errors)
}

func (pool *WorkerPool[Request, Result]) Run(request Request) {
	pool.requests <- request
}

func (pool *WorkerPool[Request, Result]) Results() <-chan Result {
	return pool.results
}

func NewWorkerPool[Request WorkerRequest, Result WorkerResult](
	provider WorkerProvider[Request, Result],
	options WorkerPoolOptions,
) *WorkerPool[Request, Result] {
	pool := &WorkerPool[Request, Result]{
		provider: provider,
		options:  options,
		requests: make(chan Request),
		results:  make(chan Result),
		wg:       sync.WaitGroup{},
		errors:   make(chan error),
	}

	pool.start()

	return pool
}
