package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

type Job interface {
	Execute(ctx context.Context) error
}

type SleepJob struct {
}

func (s *SleepJob) Execute(ctx context.Context) error {
	select {
	case <-time.After(2 * time.Second):
		fmt.Println("Sleep success")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type CalcJob struct {
	a int
}

func (c *CalcJob) Execute(ctx context.Context) error {
	ch := make(chan struct{})
	go func() {
		n := c.a
		res := 1
		for i := 2; i <= n; i++ {
			res = res * i
		}
		fmt.Println(res)
		ch <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}

type FileWriteJob struct {
	message string
}

func (f *FileWriteJob) Execute(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	file, err := os.CreateTemp(".", "tmp-*")
	if err != nil {
		return err
	}
	defer file.Close()

	content := []byte(f.message)
	if ctx.Err() != nil {
		os.Remove(file.Name())
		return ctx.Err()
	}
	if _, err = file.Write(content); err != nil {
		return err
	}
	fmt.Println("ok")
	return nil
}

func main() {
	ch := make(chan Job)
	slp := SleepJob{}
	calc := CalcJob{5}
	file := FileWriteJob{message: "hello world"}
	arr := []Job{&slp, &calc, &file}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for i := range arr {
			ch <- arr[i]
		}
		close(ch)
	}()

	WorkerPool(ctx, ch)
}

var workLimit = 100
var retry = 3

func WorkerPool(ctx context.Context, ch chan Job) {
	tokens := make(chan struct{}, workLimit)
	for range workLimit {
		tokens <- struct{}{}
	}

	for i := range ch {
		<-tokens
		go func() {
			err := i.Execute(ctx)
			if err != nil {
				for range retry {
					err = i.Execute(ctx)
					if err == nil {
						tokens <- struct{}{}
						return
					}
				}
				fmt.Println(err)
			}
			tokens <- struct{}{}
		}()
	}
	for range workLimit {
		<-tokens
	}

	fmt.Println("done")
}
