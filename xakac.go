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

    "github.com/donovanhide/eventsource"
)

type logWriter struct{}
type Route struct {
    Source string
    Target string
}

func (writer logWriter) Write(bytes []byte) (int, error) {
    return fmt.Print(time.Now().UTC().Format("2006-01-02T15:04:05Z") + ": " + string(bytes))
}

func listenToChannel(source string, target string, wg sync.WaitGroup) {
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

func deliverPayload(payload string, source string, target string) {
    resp, err := http.Post(target, "application/json", bytes.NewBuffer([]byte(payload)))
    if err != nil {
        log.Fatal(err)
    }
    log.Println("payload from", source, "has been sent to", target, "status code", resp.StatusCode)
}

func parseEnviron() []Route {
    var routes []Route
    for _, envVar := range os.Environ() {
        if strings.Contains(envVar, "XAKAC_SOURCE_TARGET_") {
            pair := strings.Split(strings.Split(envVar, "=")[1], ",")
            routes = append(routes, Route{pair[0], pair[1]})
        }
    }
    return routes
}

func parseConfig(path string) []Route {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        log.Fatal(err)
    }
    var routes []Route
    err = json.Unmarshal(data, &routes)
    if err != nil {
        log.Fatal(err)
    }
    return routes
}

func startListeners(routes []Route) {
    var wg sync.WaitGroup
    for _, route := range routes {
        go listenToChannel(route.Source, route.Target, wg)
        wg.Add(1)
    }
    wg.Wait()
}

func getRoutes() []Route {
    configPathPtr := flag.String("config", "", "path to a config file in json format")
    flag.Parse()
    if *configPathPtr != "" {
        return parseConfig(*configPathPtr)
    }
    return parseEnviron()
}

func main() {
    log.SetFlags(0)
    log.SetOutput(new(logWriter))
    startListeners(getRoutes())
}
