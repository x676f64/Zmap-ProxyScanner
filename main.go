/*
   (c) Yariya
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"
	"sync/atomic"
)

var port = flag.Int("p", 80, "proxy port")
var output = flag.String("o", "output.txt", "output file")
var configFile = flag.String("cfg", "config.json", "configuration file")

var input = flag.String("in", "", "input file to check")
var fetch = flag.String("url", "", "url proxy fetch")

var (
	jobsCompleted int64
	totalJobs     int64
)

const wt = 3

type Api struct {
	Status string `json:"Status"`
	Reason string `json:"Reason"`
}

type Config struct {
	CheckSite   string `json:"check-site"`
	ProxyType   string `json:"proxy-type"`
	HttpThreads int    `json:"http_threads"`
	Headers     struct {
		UserAgent string `json:"user-agent"`
		Accept    string `json:"accept"`
	} `json:"headers"`
	PrintIps struct {
		Enabled       bool `json:"enabled"`
		DisplayIpInfo bool `json:"display-ip-info"`
	} `json:"print_ips"`
	Timeout struct {
		HttpTimeout   int `json:"http_timeout"`
		Socks4Timeout int `json:"socks4_timeout"`
		Socks5Timeout int `json:"socks5_timeout"`
	} `json:"timeout"`
}

var config Config

func main() {
	if strings.Contains(strings.Join(os.Args[0:], ""), "-h") {
		fmt.Printf("\t\tZmap ProxyScanner @tcpfin\nHelp:\n\t-p <port> - Port you want to scan.\n\t-o <proxies.txt> - Writes proxy hits to file.\n\n\t-input <proxies.txt> - Loads the proxy list and checks it.\n\t-url https://api.com/proxies - Loads the proxies from an api and checks it.\n\n\tconfig.json - Customize the whole proxy checker\n")
		return
	}
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	cfgBytes, err := os.ReadFile(*configFile)
	if err != nil {
		log.Println("error while opening config file")
		return
	}
	err = json.Unmarshal(cfgBytes, &config)
	if err != nil {
		fmt.Println("error while parsing config json")
		return
	}

	_ = os.Remove(*output)

	exporter = &Exporter{
		out: *output,
	}


	go exporter.create()
	go Queue()
	go Scanner()
	for x := 0; x < wt; x++ {
		go Proxies.WorkerThread()
	}
	go Stater()
	
	// Start a goroutine to check for job completion
	done := make(chan bool)
	go checkJobCompletion(done)

	// Wait for job completion or interrupt signal
	select {
	case <-done:
		fmt.Println("All jobs completed. Exiting...")
	case <-waitForInterrupt():
		fmt.Println("Interrupt received. Exiting...")
	}

	exporter.Close()
}

func checkJobCompletion(done chan<- bool) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if atomic.LoadInt64(&jobsCompleted) == atomic.LoadInt64(&totalJobs) && atomic.LoadInt64(&totalJobs) > 0 {
			done <- true
			return
		}
	}
}

func waitForInterrupt() <-chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	return c
}