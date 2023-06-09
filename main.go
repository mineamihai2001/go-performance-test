package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"insert-performance-test/cli"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Request struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Response struct {
	Data string `json:"data"`
}

type Type int64

const (
	DB Type = iota
	BUCKET
)

var (
	conns      int64
	concurrent int64
)

func worker(index int, responses chan<- Response, jobs <-chan int, wg *sync.WaitGroup, t Type) {
	defer wg.Done()
	concurrent++
	for range jobs {
		request(index, responses, t)
	}
}

func request(index int, responses chan<- Response, t Type) {
	conns++

	url := "http://localhost:9999?type="
	if t == DB {
		url += "db"
	} else {
		url += "bucket"
	}

	buffer, _ := json.Marshal(Request{
		Email:    "test@mail.com",
		Password: "test@mail.com",
	})

	tr := &http.Transport{
		MaxIdleConns:        10000,
		MaxIdleConnsPerHost: 10000,
	}
	client := &http.Client{Transport: tr}
	res, err := client.Post(url+"/", "application/json", bytes.NewBuffer(buffer))
	if err != nil {
		responses <- Response{Data: err.Error()}
		return
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		responses <- Response{Data: err.Error()}
		return
	}

	response := Response{
		Data: string(data),
	}

	responses <- response
}

func saveResponses(responses *[]Response) {
	const path = "data"

	for _, res := range *responses {
		buffer, _ := json.Marshal(res)
		newLine := "\n"
		buffer = append(buffer, newLine...)

		file, err := os.OpenFile(path+"/responses.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Cloud not open file \n")
		}

		if _, err := file.Write(buffer); err != nil {
			fmt.Printf("Cloud not write to file \n")
		}

	}
}

func main() {
	manager := cli.Create()

	var totalWorkers int = 20
	var totalRequest int = 1
	var insertType Type = DB

	manager.Add(&cli.Command{
		Name:        "total",
		Short:       "t",
		Description: "The total number of requests that are going to be fired",
		Handler: func(params ...string) {
			total, _ := strconv.Atoi(params[0])
			totalRequest = total
		},
	})

	manager.Add(&cli.Command{
		Name:        "workers",
		Short:       "t",
		Description: "Number of workers which will process the requests. (defaults to 20)",
		Handler: func(params ...string) {
			workers, _ := strconv.Atoi(params[0])
			totalWorkers = workers
		},
	})

	manager.Add(&cli.Command{
		Name:        "insert",
		Short:       "i",
		Description: "Type of insert that will be made options: bucket | db (defaults to db)",
		Handler: func(params ...string) {
			if params[0] == "bucket" {
				insertType = BUCKET
			} else {
				insertType = DB
			}
		},
	})

	manager.Start()

	fmt.Printf(">>> [INFO] - Started %d requests\n\n", totalRequest)

	conns = 0
	concurrent = 0
	start := time.Now()

	var results []Response
	var wg sync.WaitGroup
	responses := make(chan Response, totalRequest)
	jobs := make(chan int, totalRequest)

	for i := 0; i < totalWorkers; i++ {
		wg.Add(1)
		i := i
		// go request(i, responses, &wg, t)
		go worker(i, responses, jobs, &wg, insertType)

	}

	for j := 1; j <= totalRequest; j++ {
		jobs <- j
	}
	close(jobs)

	go func() {
		for response := range responses {
			results = append(results, response)
		}
	}()

	wg.Wait()
	close(responses)

	saveResponses(&results)
	fmt.Printf(">>> [INFO] - collected results from %d requests\n", len(results))

	took := time.Since(start)
	ns := took.Nanoseconds()
	av := ns / conns
	average, _ := time.ParseDuration(fmt.Sprintf("%d", av) + "ns")
	fmt.Printf("Connections:\t%d\nConcurrent:\t%d\nTotal time:\t%s\nAverage time:\t%s\n", conns, concurrent, took, average)

}
