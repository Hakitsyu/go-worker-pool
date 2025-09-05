package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type WorkerId = uuid.UUID

type WorkerRequest = any
type WorkerResult = any

type Worker[Request WorkerRequest, Result WorkerResult] interface {
	GetId() WorkerId
	Run(request Request) Result
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
}

func (pool *WorkerPool[Request, Result]) start() {
	for i := 0; i < pool.options.PoolSize; i++ {
		pool.wg.Add(1)

		worker := pool.provider.Create()

		go func() {
			defer pool.wg.Done()

			for request := range pool.requests {
				pool.results <- worker.Run(request)
			}
		}()
	}
}

func (pool *WorkerPool[Request, Result]) Shutdown() {
	close(pool.requests)
	pool.wg.Wait()
	close(pool.results)
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
	}

	pool.start()

	return pool
}

type CalculationRequest struct {
	Number int
}

type CalculationResult struct {
	Result int
}

type MultiplierWorker struct {
	id WorkerId
}

func (w *MultiplierWorker) GetId() WorkerId {
	return w.id
}

func (w *MultiplierWorker) Run(request CalculationRequest) CalculationResult {
	time.Sleep(2000 * time.Millisecond)
	return CalculationResult{Result: request.Number * 2}
}

type MultiplierWorkerProvider struct{}

func (p *MultiplierWorkerProvider) Create() Worker[CalculationRequest, CalculationResult] {
	return &MultiplierWorker{
		id: uuid.New(),
	}
}

func main() {
	fmt.Println("Iniciando o Worker Pool...")

	const numberOfJobs = 10
	const poolSize = 3

	pool := NewWorkerPool(&MultiplierWorkerProvider{}, WorkerPoolOptions{
		PoolSize: poolSize,
	})

	go func() {
		for i := 1; i <= numberOfJobs; i++ {
			req := CalculationRequest{Number: i}
			fmt.Printf("Enviando trabalho: %d\n", i)
			pool.Run(req)
		}
		pool.Shutdown()
	}()

	totalSum := 0
	for result := range pool.Results() {
		fmt.Printf("Resultado recebido: %d\n", result.Result)
		totalSum += result.Result
	}

	fmt.Printf("\nProcessamento concluído. Soma total: %d\n", totalSum)
}
