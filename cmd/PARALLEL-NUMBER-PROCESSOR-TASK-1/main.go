package main

import (
	"fmt"
	"math/rand"
	"sync"
)

const randNumber = 1000
const workerPool = 5
const rps = 100

func main() {
	slc := genSlc()
	fmt.Println(slc)

	Worker(slc)
	fmt.Println(slc)
}

func Worker(slc *[]int) {
	tmp := randNumber / workerPool
	wg := &sync.WaitGroup{}
	for w := 0; w < workerPool; w += 1 {
		wg.Add(1)
		start := w * tmp
		end := start + tmp
		go func(start, end int) {
			defer wg.Done()
			for i, v := range (*slc)[start:end] {
				if v%2 == 0 {
					(*slc)[start:end][i] = v * v
					continue
				}
				(*slc)[start:end][i] = v * 2
			}
		}(start, end)
	}
	wg.Wait()
}
func genSlc() *[]int {
	slc := make([]int, randNumber)

	for i := range randNumber {
		slc[i] = rand.Intn(2000)
	}
	return &slc
}
