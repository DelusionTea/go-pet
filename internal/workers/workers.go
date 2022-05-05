package workers

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type Workers struct {
	numOfWorkers int
	input        chan func(ctx context.Context) error
}

func New(ctx context.Context, numOfWorkers int, buffer int) *Workers {
	wp := &Workers{
		numOfWorkers: numOfWorkers,
		input:        make(chan func(ctx context.Context) error, buffer),
	}
	return wp
}

func (wp *Workers) Run(ctx context.Context) {
	wg := &sync.WaitGroup{}
	for i := 0; i < wp.numOfWorkers; i++ {
		wg.Add(1)
		go func(i int) {
			fmt.Printf("Worker #%v start \n", i)
		outer:
			for {
				select {
				case f := <-wp.input:
					err := f(ctx)
					if err != nil {
						fmt.Printf("Error on worker #%v: %v\n", i, err.Error())
					}
				case <-ctx.Done():
					break outer
				}

			}
			log.Printf("Worker #%v close\n", i)
			wg.Done()
		}(i)
	}
	wg.Wait()
	close(wp.input)
}

func (wp *Workers) Push(task func(ctx context.Context) error) {
	wp.input <- task
}
