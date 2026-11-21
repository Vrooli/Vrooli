package journeytest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// Serves the real door and wallet for consumers' real-harness journeys. The
// upstream router alone is supplied by the consumer's deterministic fixture.
func TestJourneyDoorServer(t *testing.T) {
	if os.Getenv("VROOLI_JOURNEY_DOOR_SERVER") != "1" {
		t.Skip("consumer journey helper")
	}
	s := newSpine(t)
	done := make(chan struct{})
	var once sync.Once
	mux := http.NewServeMux()
	mux.HandleFunc("/__journey/ledger", func(w http.ResponseWriter, r *http.Request) {
		var charged, allocated, remaining int64
		for _, item := range []struct {
			query string
			dest  *int64
		}{
			{`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events`, &charged},
			{`SELECT COALESCE(SUM(credits),0) FROM credit_lot_allocations`, &allocated},
			{`SELECT COALESCE(SUM(remaining),0) FROM credit_funding_lots`, &remaining},
		} {
			if err := s.db.QueryRowContext(r.Context(), item.query).Scan(item.dest); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		json.NewEncoder(w).Encode(map[string]int64{"charged": charged, "allocated": allocated, "remaining": remaining})
	})
	mux.HandleFunc("/__journey/stop", func(w http.ResponseWriter, r *http.Request) { once.Do(func() { close(done) }) })
	if os.Getenv("VROOLI_JOURNEY_CONSUMER") == "1" {
		base, refresh := startConsumerDoor(t, s)
		u, _ := url.Parse(base)
		mux.Handle("/", httputil.NewSingleHostReverseProxy(u))
		fmt.Println("CONSUMER_REFRESH=" + refresh)
	} else {
		mux.Handle("/", s.server.Config.Handler)
	}
	server := httptest.NewServer(mux)
	defer server.Close()
	// These are isolated synthetic fixture credentials, never operator secrets.
	fmt.Println("DOOR_URL=" + server.URL)
	fmt.Println("DEVICE_KEY=" + s.key.Token)
	<-done
}

func startConsumerDoor(t *testing.T, s *spine) (string, string) {
	t.Helper()
	cmd := exec.Command("go", "test", ".", "-run", "^TestJourneyConsumerServer$", "-v", "-count=1", "-timeout", "10m")
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), "GOWORK=off", "VROOLI_JOURNEY_CONSUMER_DB="+s.dbURL, "VROOLI_JOURNEY_CONSUMER_GATEWAY="+s.gateway)
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var base, refresh string
	t.Cleanup(func() {
		if base != "" {
			resp, _ := http.Post(base+"/__consumer/stop", "text/plain", nil)
			if resp != nil {
				resp.Body.Close()
			}
		}
		_ = input.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("consumer helper: %v: %s", err, stderr.String())
		}
	})
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "CONSUMER_URL=") {
			base = strings.TrimPrefix(line, "CONSUMER_URL=")
		}
		if strings.HasPrefix(line, "CONSUMER_REFRESH=") {
			refresh = strings.TrimPrefix(line, "CONSUMER_REFRESH=")
		}
		if base != "" && refresh != "" {
			go io.Copy(io.Discard, output)
			return base, refresh
		}
	}
	t.Fatalf("consumer helper did not start: %s", stderr.String())
	return "", ""
}
