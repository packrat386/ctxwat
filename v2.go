package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type message struct {
	value  string
	ctx    context.Context
	cancel context.CancelFunc
}

func split(input <-chan int, output chan<- message) {
	for i := range input {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		values, err := generateWithContext(ctx, i)
		if err != nil {
			log.Println("darn")
			cancel()
			continue
		}

		for _, v := range values {
			output <- message{value: v, ctx: ctx, cancel: cancel}
		}

		log.Println("split: ", i)
	}
}

func generateWithContext(ctx context.Context, num int) ([]string, error) {
	values := make([]string, 0, num)
	for i := 0; i < num; i++ {
		values = append(values, fmt.Sprintf("%d", i))
	}

	return values, nil
}

func execute(input <-chan message) {
	for msg := range input {
		doWithContext(msg.ctx, msg.value)
		msg.cancel()
	}
}

func doWithContext(ctx context.Context, msg string) {
	if err := ctx.Err(); err != nil {
		return
	}

	log.Println("completed: ", msg)
}

func main() {
	var wg sync.WaitGroup
	numbers := make(chan int)
	messages := make(chan message)

	wg.Add(1)
	go func() {
		split(numbers, messages)
		close(messages)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		execute(messages)
		wg.Done()
	}()

	for i := 0; i < 5; i++ {
		numbers <- i
	}

	close(numbers)

	wg.Wait()
}
