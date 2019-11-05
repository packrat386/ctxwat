package main

import (
	"fmt"
	"log"
	"sync"
)

func split(input <-chan int, output chan<- string) {
	for i := range input {
		values, err := generate(i)
		if err != nil {
			log.Println("darn")
			continue
		}

		for _, v := range values {
			output <- v
		}

		log.Println("split: ", i)
	}
}

func generate(num int) ([]string, error) {
	values := make([]string, 0, num)
	for i := 0; i < num; i++ {
		values = append(values, fmt.Sprintf("%d", i))
	}
	return values, nil
}

func execute(input <-chan string) {
	for msg := range input {
		do(msg)
	}
}

func do(msg string) {
	log.Println("completed: ", msg)
}

func main() {
	var wg sync.WaitGroup
	numbers := make(chan int)
	messages := make(chan string)

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
