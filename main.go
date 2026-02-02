package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

type Stat struct {
	avgResponseTime int64
	statusCodes     map[int]int
	respQty         int64
	mutex           sync.Mutex
}

func constructStatuses() *Stat {
	return &Stat{
		statusCodes: make(map[int]int),
	}
}

type GStat struct {
	items map[string]*Stat
	mu    sync.RWMutex
}

type Monitor struct {
	domains[] string
	gs *GStat
	done chan struct{}
}

func newMonitor(domains[] string) *Monitor {
	gs := &GStat{
		items: make(map[string]*Stat, len(domains)),
	}
	for _, name := range domains {
		gs.items[name] = constructStatuses()
	}
	done := make(chan struct{})

	return &Monitor{
		domains: domains,
		gs: gs,
		done: done,
	}
}

func (m *Monitor) displayDomains() {
	fmt.Println(strings.Repeat("=", 20) + "List of domains" + strings.Repeat("=", 20))
	for ind, name := range m.domains {
		fmt.Printf("%d --- %s\n", ind, name)
	}
	fmt.Println(strings.Repeat("=", 55))
}

func (m *Monitor) start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	wg := sync.WaitGroup{}
	go func() {
		for {
			select {
			case <-ticker.C:
				wg.Add(len(m.domains))
				for _, val := range m.domains {
					defer wg.Done()
					go repeatAction(val, m.gs)
				}
			case <-m.done:
				ticker.Stop()
				return
			}
		}
	}()
	wg.Wait()
}

func (m *Monitor) stop() {
	close(m.done)
}

func (m *Monitor) printSummary() {
	fmt.Println("\n" + strings.Repeat("=", 55))
	fmt.Printf("%-20s | %-12s | %-8s | %-15s\n", "DOMAIN", "AVG TIME", "REQS", "STATUS CODES")
	fmt.Println(strings.Repeat("-", 55))

	for domain, s := range m.gs.items {
		avg := s.avgResponseTime
		qty := s.respQty

		codes := ""
		for code, count := range s.statusCodes {
			codes += fmt.Sprintf("%d:%d ", code, count)
		}


		fmt.Printf("%-20s | %-10dms | %-8d | %-15s\n", domain, avg, qty, codes)
	}
	fmt.Println(strings.Repeat("=", 55))
}

type FileOpenError struct {
	fname string
}

func (foe FileOpenError) Error() string {
	return fmt.Sprintf("File open error on file - %s", foe.fname)
}

func loadDomains(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, &FileOpenError{fname: filename}
	}
	defer file.Close()
	var domains []string

	reader := bufio.NewScanner(file)

	for reader.Scan() {
		line := reader.Text()
		if line == "" {
			continue
		}
		domains = append(domains, line)
	}
	return domains, nil
}



func repeatAction(domain string, gstat *GStat) error {
	start := time.Now()
	resp, err := http.Get(domain)
	end := time.Now()
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	gstat.mu.RLock()
	s := gstat.items[domain]
	gstat.mu.RUnlock()

	s.mutex.Lock()
	rTime := end.Sub(start).Milliseconds()
	s.avgResponseTime = (s.avgResponseTime*s.respQty + rTime) / (s.respQty + 1)
	s.statusCodes[resp.StatusCode] += 1
	s.respQty++
	s.mutex.Unlock()


	fmt.Printf("domain request for %s, status code - %d\n", domain, resp.StatusCode)
	fmt.Printf("response time - %dms\n", rTime)
	return nil
}

func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
}

func main() {
	filename := "domains.txt"
	domains, err := loadDomains(filename)

	if (err != nil) {
		fmt.Printf("%s", err)
		return;
	}

	monitor := newMonitor(domains)
	monitor.displayDomains()
	interval := time.Second * 1

	monitor.start(interval)

	handleSignals()
	monitor.stop()
	monitor.printSummary()
}
