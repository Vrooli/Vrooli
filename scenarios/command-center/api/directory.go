package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// portDirectory answers "where is scenario X listening right now" by reading
// the control plane's own scenario listing. Ports are assigned by the
// lifecycle manager at start, so a fixed port is wrong the first time a
// source restarts; the directory is what makes a source findable again.
type portDirectory struct {
	coreBaseURL string
	http        *http.Client

	mu        sync.Mutex
	ports     map[string]int
	refreshed time.Time
}

const portDirectoryTTL = 60 * time.Second

func newPortDirectory(coreBaseURL string) *portDirectory {
	return &portDirectory{coreBaseURL: coreBaseURL, http: &http.Client{Timeout: 3 * time.Second}, ports: map[string]int{}}
}

// resolver returns a base-URL function for one scenario. An explicit base URL
// wins, then an explicit port, then the live directory. An unknown scenario
// resolves to "" so the client reports ErrNotAvailable rather than guessing.
func (d *portDirectory) resolver(scenario, baseURLEnv, portEnv string) func() string {
	return func() string {
		if v := os.Getenv(baseURLEnv); v != "" {
			return v
		}
		if v := os.Getenv(portEnv); v != "" {
			return "http://localhost:" + v
		}
		if port := d.lookup(scenario); port > 0 {
			return "http://localhost:" + strconv.Itoa(port)
		}
		return ""
	}
}

func (d *portDirectory) lookup(scenario string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if time.Since(d.refreshed) > portDirectoryTTL {
		d.refreshLocked()
	}
	return d.ports[scenario]
}

func (d *portDirectory) refreshLocked() {
	d.refreshed = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.coreBaseURL+"/scenarios", nil)
	if err != nil {
		return
	}
	resp, err := d.http.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var body struct {
		Data []struct {
			Name   string         `json:"name"`
			Status string         `json:"status"`
			Ports  map[string]int `json:"ports"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&body) != nil {
		return
	}
	next := map[string]int{}
	for _, sc := range body.Data {
		if sc.Status == "running" && sc.Ports["API_PORT"] > 0 {
			next[sc.Name] = sc.Ports["API_PORT"]
		}
	}
	d.ports = next
}
