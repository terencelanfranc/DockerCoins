package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"gopkg.in/redis.v4"
)

func main() {
	fmt.Printf("Docker Coins - Worker")

	worker, err := NewWorker("redis:6379", "http://rng:8002/32", "http://rng:8002/32")
	if err != nil {
		panic(err)
	}

	worker.Mine()
}

// Worker is the struct that will do the main work of mining for a DockerCoin
type Worker struct {
	client    *redis.Client
	dbURL     string
	rngURL    string
	hasherURL string
}

// NewWorker creates a new instance of a worker
func NewWorker(dbURL string, rngURL string, hasherURL string) (Worker, error) {

	var worker Worker

	client := redis.NewClient(&redis.Options{
		Addr:     dbURL,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping().Result()
	if err != nil {
		//TODO: Handle error properly
		return worker, errors.New("Unable to connect to database")
	}

	worker = Worker{
		dbURL:     dbURL,
		rngURL:    rngURL,
		hasherURL: hasherURL,
		client:    client,
	}

	return worker, nil
}

// Mine will start the work of mining for a DockerCoin
func (w Worker) Mine() error {
	//interval := 1
	deadline := time.Time{}
	loopsDone := 0

	for {
		current := time.Now()
		if current.After(deadline) {
			fmt.Printf("%d unit of work done, updating hash counter\n", loopsDone)
			err := w.client.Set("hashes", loopsDone, 0).Err()
			if err != nil {
				panic(err)
			}
			loopsDone = 0
			deadline = time.Now().Add(1 * time.Second)
		}
		err := w.workOnce()
		if err != nil {
			panic(err)
		}
		loopsDone++
	}
}

func (w Worker) workOnce() error {
	fmt.Printf("Doing one unit of work\n")
	time.Sleep(100 * time.Millisecond)

	randomBytes, err := w.getRandomBytes()
	if err != nil {
		return err
	}

	hexHash, err := w.hashBytes(randomBytes)
	if err != nil {
		return err
	}

	if strings.HasPrefix(hexHash, "0") {
		fmt.Printf("No coin found\n")
		return nil
	}
	fmt.Printf("Coind found %s", hexHash)
	//w.client.HMSet('wallet', hexHash, randomBytes)
	return nil
}

func (w Worker) getRandomBytes() (string, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	request, _ := http.NewRequest("GET", w.rngURL, nil)
	request.Header.Set("content-type", "text/plain")

	response, err := netClient.Do(request)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (w Worker) hashBytes(data string) (string, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	request, _ := http.NewRequest("POST", w.hasherURL, bytes.NewBufferString(data))
	request.Header.Set("content-type", "text/plain")

	response, err := netClient.Do(request)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
