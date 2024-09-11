/*
	(c) Yariya
*/

package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"h12.io/socks"
)

type Proxy struct {
	ips                  map[string]struct{}
	targetSites          []string
	httpStatusValidation bool
	timeout              time.Duration
	maxHttpThreads       int64

	openHttpThreads int64
	mu              sync.Mutex
}

var Proxies = &Proxy{
	// in work
	targetSites: []string{"https://google.com", "https://cloudflare.com"},

	httpStatusValidation: false,
	// now cfg file
	timeout:        time.Second * 5,
	maxHttpThreads: int64(config.HttpThreads),
	ips:            make(map[string]struct{}),
}

func (p *Proxy) WorkerThread() {
	for {
		for atomic.LoadInt64(&p.openHttpThreads) < int64(config.HttpThreads) {
			p.mu.Lock()
			for proxyStr, _ := range p.ips {
				proxyType, host, port := parseProxyString(proxyStr, *port)
				switch proxyType {
				case "http":
					go p.CheckProxyHTTP(host, port)
				case "socks4":
					go p.CheckProxySocks4(host, port)
				case "socks5":
					go p.CheckProxySocks5(host, port)
				default:
					log.Printf("Unknown proxy type for %s, defaulting to HTTP", proxyStr)
					go p.CheckProxyHTTP(host, port)
				}
				delete(p.ips, proxyStr)
				break
			}
			p.mu.Unlock()
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func parseProxyString(proxyStr string, defaultPort int) (proxyType string, host string, port int) {
	proxyStr = strings.TrimSpace(proxyStr)
	var err error

	if strings.HasPrefix(proxyStr, "http://") {
		proxyType = "http"
		proxyStr = strings.TrimPrefix(proxyStr, "http://")
	} else if strings.HasPrefix(proxyStr, "socks4://") {
		proxyType = "socks4"
		proxyStr = strings.TrimPrefix(proxyStr, "socks4://")
	} else if strings.HasPrefix(proxyStr, "socks5://") {
		proxyType = "socks5"
		proxyStr = strings.TrimPrefix(proxyStr, "socks5://")
	} else {
		proxyType = strings.ToLower(config.ProxyType)
	}

	parts := strings.Split(proxyStr, ":")
	host = parts[0]
	if len(parts) > 1 {
		port, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			log.Printf("Error parsing port for %s: %v", proxyStr, err)
			port = defaultPort
		}
	} else {
		port = defaultPort
	}

	return proxyType, host, port
}

func (p *Proxy) CheckProxyHTTP(host string, port int) {
	atomic.AddInt64(&p.openHttpThreads, 1)
	defer func() {
		atomic.AddInt64(&p.openHttpThreads, -1)
		atomic.AddUint64(&checked, 1)
		atomic.AddInt64(&jobsCompleted, 1)
	}()

	proxyUrl, err := url.Parse(fmt.Sprintf("http://%s:%d", host, port))
	if err != nil {
		log.Println(err)
		return
	}

	tr := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
		DialContext: (&net.Dialer{
			Timeout:   time.Second * time.Duration(config.Timeout.HttpTimeout),
			KeepAlive: time.Second,
			DualStack: true,
		}).DialContext,
	}

	client := http.Client{
		Timeout:   time.Second * time.Duration(config.Timeout.HttpTimeout),
		Transport: tr,
	}

	req, err := http.NewRequest("GET", config.CheckSite, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Add("User-Agent", config.Headers.UserAgent)
	req.Header.Add("accept", config.Headers.Accept)

	res, err := client.Do(req)
	if err != nil {
		atomic.AddUint64(&proxyErr, 1)
		if strings.Contains(err.Error(), "timeout") {
			atomic.AddUint64(&timeoutErr, 1)
			return
		}
		return
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		atomic.AddUint64(&statusCodeErr, 1)
	} else {
		if config.PrintIps.Enabled {
			go PrintProxy(host, port)
		}
		atomic.AddUint64(&success, 1)
		exporter.Add(fmt.Sprintf("%s:%d", host, port))
	}
}

func (p *Proxy) CheckProxySocks4(host string, port int) {
	atomic.AddInt64(&p.openHttpThreads, 1)
	defer func() {
		atomic.AddInt64(&p.openHttpThreads, -1)
		atomic.AddUint64(&checked, 1)
		atomic.AddInt64(&jobsCompleted, 1)
	}()

	tr := &http.Transport{
		Dial: socks.Dial(fmt.Sprintf("socks4://%s:%d?timeout=%ds", host, port, config.Timeout.Socks4Timeout)),
	}

	client := http.Client{
		Timeout:   time.Second * time.Duration(config.Timeout.HttpTimeout),
		Transport: tr,
	}

	req, err := http.NewRequest("GET", config.CheckSite, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Add("User-Agent", config.Headers.UserAgent)
	req.Header.Add("accept", config.Headers.Accept)

	res, err := client.Do(req)
	if err != nil {
		atomic.AddUint64(&proxyErr, 1)
		if strings.Contains(err.Error(), "timeout") {
			atomic.AddUint64(&timeoutErr, 1)
			return
		}
		return
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		atomic.AddUint64(&statusCodeErr, 1)
	} else {
		if config.PrintIps.Enabled {
			go PrintProxy(host, port)
		}
		atomic.AddUint64(&success, 1)
		exporter.Add(fmt.Sprintf("%s:%d", host, port))
	}
}

func (p *Proxy) CheckProxySocks5(host string, port int) {
	atomic.AddInt64(&p.openHttpThreads, 1)
	defer func() {
		atomic.AddInt64(&p.openHttpThreads, -1)
		atomic.AddUint64(&checked, 1)
		atomic.AddInt64(&jobsCompleted, 1)
	}()

	tr := &http.Transport{
		Dial: socks.Dial(fmt.Sprintf("socks5://%s:%d?timeout=%ds", host, port, config.Timeout.Socks5Timeout)),
	}

	client := http.Client{
		Timeout:   time.Second * time.Duration(config.Timeout.HttpTimeout),
		Transport: tr,
	}

	req, err := http.NewRequest("GET", config.CheckSite, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Add("User-Agent", config.Headers.UserAgent)
	req.Header.Add("accept", config.Headers.Accept)

	res, err := client.Do(req)
	if err != nil {
		atomic.AddUint64(&proxyErr, 1)
		if strings.Contains(err.Error(), "timeout") {
			atomic.AddUint64(&timeoutErr, 1)
			return
		}
		return
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		atomic.AddUint64(&statusCodeErr, 1)
	} else {
		if config.PrintIps.Enabled {
			go PrintProxy(host, port)
		}
		atomic.AddUint64(&success, 1)
		exporter.Add(fmt.Sprintf("%s:%d", host, port))
	}
}
