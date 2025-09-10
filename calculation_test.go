package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestWorkerPool_SingleJob(t *testing.T) {
	pool := NewWorkerPool(&MultiplierWorkerProvider{}, WorkerPoolOptions{PoolSize: 1})
	defer pool.Shutdown()

	req := CalculationRequest{context: context.Background(), Number: 5}
	pool.Run(req)

	select {
	case res := <-pool.Results():
		if res.Result != 10 {
			t.Fatalf("expected 10, got %d", res.Result)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestWorkerPool_MultipleJobs(t *testing.T) {
	pool := NewWorkerPool(&MultiplierWorkerProvider{}, WorkerPoolOptions{PoolSize: 1})

	results := make(map[int]bool)

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()

		for result := range pool.Results() {
			results[result.Result] = true
		}
	}()

	const jobCount = 5
	for i := 1; i <= jobCount; i++ {
		req := CalculationRequest{context: context.Background(), Number: i}
		pool.Run(req)
	}

	pool.Shutdown()
	wg.Wait()

	if len(results) != jobCount {
		t.Fatal("missing result")
	}
}

func TestWorkerPool_ManyWorkers_ManyJobs(t *testing.T) {
	const poolSize = 10
	const jobCount = 100

	pool := NewWorkerPool(&MultiplierWorkerProvider{}, WorkerPoolOptions{PoolSize: poolSize})

	results := make(map[int]bool)
	var wg sync.WaitGroup

	go func() {
		wg.Add(1)
		defer wg.Done()
		for result := range pool.Results() {
			results[result.Result] = true
		}
	}()

	for i := 1; i <= jobCount; i++ {
		req := CalculationRequest{context: context.Background(), Number: i}
		pool.Run(req)
	}

	pool.Shutdown()
	wg.Wait()

	if len(results) != jobCount {
		t.Fatalf("expected %d results, got %d", jobCount, len(results))
	}
	for i := 1; i <= jobCount; i++ {
		if !results[i*2] {
			t.Errorf("missing result for %d", i)
		}
	}
}

func TestWorkerPool_CancelJob(t *testing.T) {
	pool := NewWorkerPool(&MultiplierWorkerProvider{}, WorkerPoolOptions{PoolSize: 2})
	defer pool.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	req := CalculationRequest{context: ctx, Number: 99}

	cancel()
	pool.Run(req)

	select {
	case <-pool.Results():
		t.Fatal("expected no result for cancelled job")
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case err := <-pool.errors:
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for error")
	}
}
