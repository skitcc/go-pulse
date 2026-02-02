package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

type Stat struct {
	avgResponseTime int64
	statusCodes map[int]int
	respQty int64
	mutex sync.Mutex
}

func repeatAction(domain string, m_map map[string]Stat) error {
	start := time.Now()
	resp, err := http.Get(domain)
	end := time.Now()
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	m_map[domain].mutex.Lock()
	rTime := end.Sub(start).Milliseconds();
	m_map[domain].avgResponseTime = (m_map[domain].avgResponseTime * m_map[domain].respQty + rTime) / (m_map[domain].respQty + 1) 
	m_map[domain].statusCodes[resp.StatusCode] += 1
	m_map[domain].respQty++
	m_map[domain].mutex.Unlock()

	fmt.Printf("domain request for %s, status code - %d\n", domain, resp.StatusCode)
	fmt.Printf("response time - %dms\n", rTime)
	return nil
}


func summary(stats []Stat) {

}

func main() {
	filename := "domains.txt"
	file, err := os.Open(filename)

	if err != nil {
		fmt.Println("err open file")
		return;
	}
	defer file.Close()
	var domains []string
	
	reader := bufio.NewScanner(file)

	for reader.Scan() {
		line := reader.Text()
		if (line == "") {
			continue
		}
		domains = append(domains, line)
	}
	
	var statsPerDomain map[string]Stat = make(map[string] Stat, len(domains))

	for ind, name := range(domains) {
		fmt.Printf("%d --- %s\n", ind, name)
	}

	interval := time.Second * 1;
	ticker := time.NewTicker(interval)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				for _, val := range(domains) {
					go repeatAction(val, statsPerDomain)
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	done<-true
}
