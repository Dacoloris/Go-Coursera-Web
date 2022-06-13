package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	var in, out chan interface{}
	wg := &sync.WaitGroup{}

	for _, j := range jobs {
		out = make(chan interface{})
		wg.Add(1)

		go func(j job, in, out chan interface{}) {
			defer close(out)
			defer wg.Done()

			j(in, out)
		}(j, in, out)

		in = out
	}

	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	for i := range in {
		wg.Add(1)

		go func(data string) {
			defer wg.Done()

			res1 := make(chan string)
			go func(res chan string) {
				res <- DataSignerCrc32(data)
			}(res1)

			res2 := make(chan string)
			go func(res chan string, mu *sync.Mutex) {
				mu.Lock()
				md5 := DataSignerMd5(data)
				mu.Unlock()
				res <- DataSignerCrc32(md5)
			}(res2, mu)

			out <- fmt.Sprintf("%s~%s", <-res1, <-res2)
		}(fmt.Sprintf("%v", i))
	}

	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	for i := range in {
		wg.Add(1)
		go func(data string, mu *sync.Mutex) {
			defer wg.Done()
			res := make([]string, 6)
			wgInner := &sync.WaitGroup{}
			for th := 0; th < 6; th++ {
				wgInner.Add(1)
				go func(data string, th int, res []string, mu *sync.Mutex) {
					defer wgInner.Done()
					crc := DataSignerCrc32(fmt.Sprintf("%d", th) + data)
					mu.Lock()
					res[th] = crc
					mu.Unlock()
				}(data, th, res, mu)
			}

			wgInner.Wait()
			out <- strings.Join(res, "")

		}(fmt.Sprintf("%v", i), mu)
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	arr := make([]string, 0)
	for i := range in {
		arr = append(arr, fmt.Sprintf("%v", i))
	}
	sort.Strings(arr)
	out <- strings.Join(arr, "_")
}
