package main

import (
	"fmt"
	"strings"
)

type TextProcessor interface {
	Process(string) string
}

type Pipeline struct {
	Processors []TextProcessor
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		Processors: []TextProcessor{},
	}
}

func (p *Pipeline) Add(proc ...TextProcessor) {
	for _, v := range proc {
		p.Processors = append(p.Processors, v)
	}
}

func (p *Pipeline) Processing(s string) string {
	for i := range p.Processors {
		fmt.Println(s)
		s = p.Processors[i].Process(s)
	}
	return s
}

func main() {
	up := UpperCaseProcessor{}
	lo := LowerCaseProcessor{}
	re := ReverseProcessor{}
	ce := CensorProcessor{}
	s := "Process_machine"
	pipeline := NewPipeline()
	pipeline.Add(&up, &lo, &re, &ce)

	res := pipeline.Processing(s)
	fmt.Println(res)
}

type UpperCaseProcessor struct {
}

func (u *UpperCaseProcessor) Process(s string) string {
	return strings.ToUpper(s)
}

type LowerCaseProcessor struct {
}

func (l *LowerCaseProcessor) Process(s string) string {
	return strings.ToLower(s)
}

type ReverseProcessor struct {
}

func (r *ReverseProcessor) Process(s string) string {
	arr := make([]byte, len(s))
	j := 0
	for i := len(s) - 1; i >= 0; i-- {
		arr[j] = s[i]
		j++
	}
	return string(arr)
}

type CensorProcessor struct {
}

func (c *CensorProcessor) Process(s string) string {
	ln := len(s)
	arr := make([]byte, ln)
	for i := range arr {
		arr[i] = '*'
	}
	return string(arr)
}
