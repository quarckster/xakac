package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/donovanhide/eventsource"
)

type logWriter struct{}

type route struct {
	Source string
	Target string
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format("2006-01-02T15:04:05Z") + ": " + string(bytes))
}

func listenToChannel(source string, target string, wg *sync.WaitGroup) {
	defer wg.Done()
	stream, err := eventsource.Subscribe(source, "")
	if err != nil {
		log.Fatal(err)
	}
	for {
		ev := <-stream.Events
		if ev.Event() == "ready" {
			log.Println("listening to", source)
		}
		if ev.Event() == "" {
			deliverPayload(ev.Data(), source, target)
		}
	}
}

func prepareRequest(target string, payload string) http.Request {
	var parsedStructure interface{}
	err := json.Unmarshal([]byte(payload), &parsedStructure)
	if err != nil {
		log.Println(err)
	}
	payloadMap := parsedStructure.(map[string]interface{})
	body, _ := json.Marshal(payloadMap["body"])
	req, _ := http.NewRequest("POST", target, bytes.NewBuffer(body))
	delete(payloadMap, "body")
	for header, value := range payloadMap {
		req.Header.Add(header, fmt.Sprintf("%v", value))
	}
	return *req
}

func deliverPayload(payload string, source string, target string) {
	request := prepareRequest(target, payload)
	client := http.Client{}
	resp, err := client.Do(&request)
	if err != nil {
		var DNSError *net.DNSError
		if errors.As(err, &DNSError) {
			log.Println("delivering payload to", target, "failed:", DNSError.Err)
			return
		}
		log.Println("delivering payload to", target, "failed:", err)
	}
	log.Println("payload from", source, "has been sent to", target, "status code", resp.StatusCode)
}

func parseEnviron() []route {
	var routes []route
	for _, envVar := range os.Environ() {
		if strings.Contains(envVar, "XAKAC_SOURCE_TARGET_") {
			pair := strings.Split(strings.Split(envVar, "=")[1], ",")
			routes = append(routes, route{pair[0], pair[1]})
		}
	}
	return routes
}

func parseConfig(path string) []route {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var routes []route
	err = json.Unmarshal(data, &routes)
	if err != nil {
		log.Fatal(err)
	}
	return routes
}

func startListeners(routes []route) {
	var wg sync.WaitGroup
	for _, route := range routes {
		go listenToChannel(route.Source, route.Target, &wg)
		wg.Add(1)
	}
	wg.Wait()
}

func getRoutes() []route {
	configPath := *flag.String("config", "", "path to a config file in json format")
	flag.Parse()
	if configPath != "" {
		return parseConfig(configPath)
	}
	return parseEnviron()
}

func main() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))
	startListeners(getRoutes())
}
