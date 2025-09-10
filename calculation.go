package main

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CalculationRequest struct {
	context context.Context
	Number  int
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

func (w *MultiplierWorker) Run(request CalculationRequest) (CalculationResult, error) {
	result := make(chan CalculationResult, 1)

	go func() {
		time.Sleep(50 * time.Millisecond)
		result <- CalculationResult{Result: request.Number * 2}
	}()

	select {
	case <-request.context.Done():
		return CalculationResult{}, request.context.Err()
	case r, ok := <-result:
		if !ok {
			return CalculationResult{}, request.context.Err()
		}
		return r, nil
	}
}

type MultiplierWorkerProvider struct{}

func (p *MultiplierWorkerProvider) Create() Worker[CalculationRequest, CalculationResult] {
	return &MultiplierWorker{
		id: uuid.New(),
	}
}
