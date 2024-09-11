/*
	(c) Yariya
*/

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func Scanner() {
	if *fetch != "" {
		log.Printf("Detected URL Mode.\n")
		res, err := http.Get(*fetch)
		if err != nil {
			log.Fatalln("fetch error")
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalln("fetch body error")
		}
		res.Body.Close()

		scanner := bufio.NewScanner(bytes.NewReader(body))
		for scanner.Scan() {
			ip := strings.TrimSpace(scanner.Text())
			if ip != "" {
				queueChan <- ip
			}
		}
	} else if *input != "" {
		fmt.Printf("Detected FILE Mode.\n")
		file, err := os.Open(*input)
		if err != nil {
			log.Fatalln("open file err")
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				queueChan <- line
			}
		}
	} else {
		fmt.Printf("Detected ZMAP Mode.\n")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			ip := strings.TrimSpace(scanner.Text())
			if ip != "" {
				queueChan <- ip
			}
		}
	}
}
