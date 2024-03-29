package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	sse "github.com/r3labs/sse"
)

type logWriter struct{}

type route struct {
	Source string
	Target string
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format("2006-01-02T15:04:05Z") + ": " + string(bytes))
}

func stop(route route, wg *sync.WaitGroup, routeChan chan route) {
	wg.Done()
	// In case of any sse client error we would like to reconnect
	routeChan <- route
}

func subscribeToStream(route route, wg *sync.WaitGroup, routeChan chan route) {
	defer stop(route, wg, routeChan)
	client := sse.NewClient(route.Source)
	client.OnDisconnect(func(client *sse.Client) {
		log.Println("disconnected from", route.Source)
	})
	err := client.Subscribe("", func(msg *sse.Event) {
		event := string(msg.Event[:])
		switch {
		case event == "ready":
			log.Println("forwarding", route.Source, "to", route.Target)
		case event == "ping":
		default:
			deliverPayload(msg.Data, route.Source, route.Target)
		}
	})
	if err != nil {
		log.Println(err)
	}
}

func prepareRequest(target string, payload []byte) http.Request {
	var parsedStructure interface{}
	err := json.Unmarshal(payload, &parsedStructure)
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

func deliverPayload(payload []byte, source string, target string) {
	request := prepareRequest(target, payload)
	client := http.Client{}
	resp, err := client.Do(&request)
	if err != nil {
		log.Println("delivering payload failed:", err)
		return
	}
	log.Println("payload from", source, "has been sent to", target, "status code", resp.StatusCode)
}

func parseEnviron() []route {
	var routes []route
	for _, envVar := range os.Environ() {
		if strings.Contains(envVar, "XAKAC_SOURCE_TARGET_") {
			envVarValue := strings.Join(strings.Split(envVar, "=")[1:], "=")
			pair := strings.Split(envVarValue, ",")
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

func supervise(wg *sync.WaitGroup, routeChan chan route) {
	defer wg.Done()
	for route := range routeChan {
		wg.Add(1)
		go subscribeToStream(route, wg, routeChan)
	}
}

func startListeners(routes []route) {
	var wg sync.WaitGroup
	routeChan := make(chan route)
	wg.Add(1)
	go supervise(&wg, routeChan)
	for _, route := range routes {
		routeChan <- route
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
