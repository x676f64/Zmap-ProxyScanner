/*
	(c) Yariya
*/

package main

import "sync/atomic"

var queueChan = make(chan string)

func Queue() {
	for {
		select {
		case ip := <-queueChan:
			{
				Proxies.mu.Lock()
				Proxies.ips[ip] = struct{}{}
				Proxies.mu.Unlock()
				atomic.AddUint64(&imported, 1)
				atomic.AddInt64(&totalJobs, 1)
			}
		}
	}
}