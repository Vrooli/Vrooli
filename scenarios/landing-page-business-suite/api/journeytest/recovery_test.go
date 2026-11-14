package journeytest

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/commerce"
)

func TestJourneyRecoveryServer(t *testing.T) {
	url := os.Getenv("VROOLI_JOURNEY_RECOVERY_DB")
	if url == "" {
		t.Skip("crash recovery helper")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	live := false
	door := &aidoor.Door{Keys: aidoor.KeyStore{DB: db}, Wallet: commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: db, Limits: commerce.NewLimitsService(db, "postgres", nil), Dialect: "postgres", LiveMode: &live}), GatewayURL: os.Getenv("VROOLI_JOURNEY_RECOVERY_GATEWAY"), ObserveTransition: func(row aidoor.RequestTransition) {
		data, _ := json.Marshal(row)
		fmt.Println("DOOR_TRACE=" + string(data))
	}}
	if err := door.RecoverPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/__journey/crash", func(http.ResponseWriter, *http.Request) { os.Exit(91) })
	mux.HandleFunc("/__journey/stop", func(http.ResponseWriter, *http.Request) { close(done) })
	mux.Handle("/", door)
	server := httptest.NewServer(mux)
	defer server.Close()
	fmt.Println("RECOVERY_URL=" + server.URL)
	<-done
}

func startRecoveryDoor(t *testing.T, s *spine) (string, func()) {
	t.Helper()
	cmd := exec.Command("go", "test", ".", "-run", "^TestJourneyRecoveryServer$", "-count=1", "-v", "-timeout", "2m")
	cmd.Env = append(os.Environ(), "GOWORK=off", "VROOLI_JOURNEY_RECOVERY_DB="+s.dbURL, "VROOLI_JOURNEY_RECOVERY_GATEWAY="+s.gateway)
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	observe := func(line string) {
		if strings.HasPrefix(line, "DOOR_TRACE=") {
			var row aidoor.RequestTransition
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "DOOR_TRACE=")), &row); err != nil {
				t.Errorf("invalid trace: %v", err)
			} else {
				s.trace.observe(row)
			}
		}
	}
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		observe(line)
		if strings.HasPrefix(line, "RECOVERY_URL=") {
			url := strings.TrimPrefix(line, "RECOVERY_URL=")
			drained := make(chan struct{})
			go func() {
				defer close(drained)
				for scanner.Scan() {
					observe(scanner.Text())
				}
			}()
			crashed := false
			t.Cleanup(func() {
				if !crashed {
					resp, err := http.Post(url+"/__journey/stop", "text/plain", nil)
					if err == nil {
						resp.Body.Close()
					}
				}
				if err := cmd.Wait(); err != nil && !crashed {
					t.Errorf("recovery helper: %v %s", err, stderr.String())
				}
				<-drained
			})
			return url, func() {
				crashed = true
				resp, err := http.Post(url+"/__journey/crash", "text/plain", nil)
				if err == nil {
					resp.Body.Close()
				}
			}
		}
	}
	t.Fatalf("recovery helper did not start: %s", stderr.String())
	return "", nil
}

func TestJ2_CrashedStreamSettlesFromReceipt(t *testing.T) {
	s := newSpine(t)
	url, crash := startRecoveryDoor(t, s)
	req, _ := http.NewRequest("POST", url+"/api/v1/ai/chat/completions", strings.NewReader(`{"model":"fixture-model","stream":true,"messages":[{"role":"user","content":"crash"}]}`))
	req.Header.Set("Authorization", "Bearer "+s.key.Token)
	req.Header.Set("Idempotency-Key", "crash")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	buffer := make([]byte, 10)
	if _, err := resp.Body.Read(buffer); err != nil {
		t.Fatal(err)
	}
	crash()
	resp.Body.Close()
	assertCharged(t, s, 0)
	// The real Gateway process finishes independently after LPBS exits.
	time.Sleep(2300 * time.Millisecond)
	_, _ = startRecoveryDoor(t, s)
	s.trace.require(t, "reconciling", "receive", "commit")
	assertCharged(t, s, 200)
	_, _ = startRecoveryDoor(t, s)
	s.trace.require(t, "reconciling", "receive", "commit")
	assertCharged(t, s, 200)
}

// A real process dies after the cap participant writes but before the wallet
// INSERT completes. A trigger-held advisory lock injects that interruption in
// isolated real Postgres; no caller operation ID can resume an abandoned claim.
func TestJ2_PreDispatchCrashDoesNotStrandKeyCaps(t *testing.T) {
	for _, kind := range []string{"device", "session"} {
		t.Run(kind, func(t *testing.T) {
			var dispatched atomic.Int32
			s := newSpine(t, func([]byte) { dispatched.Add(1) })
			if _, err := s.db.Exec(`UPDATE ai_access_keys SET max_credits=1000`); err != nil {
				t.Fatal(err)
			}
			key := s.key
			if kind == "session" {
				var err error
				key, err = (aidoor.KeyStore{DB: s.db}).IssueSession(context.Background(), s.key, "codex", "fixture-model", 1000, time.Hour)
				if err != nil {
					t.Fatal(err)
				}
			}
			url, crash := startRecoveryDoor(t, s)
			if _, err := s.db.Exec(`CREATE FUNCTION block_journey_admission() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(48251920); RETURN NEW; END $$; CREATE TRIGGER block_journey_admission BEFORE INSERT ON business_account_credit_reservations FOR EACH ROW EXECUTE FUNCTION block_journey_admission()`); err != nil {
				t.Fatal(err)
			}
			lock, err := s.db.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Rollback()
			if _, err = lock.Exec(`SELECT pg_advisory_xact_lock(48251920)`); err != nil {
				t.Fatal(err)
			}
			request := func(base string) (*http.Response, error) {
				req, _ := http.NewRequest("POST", base+"/api/v1/ai/chat/completions", strings.NewReader(`{"model":"fixture-model","stream":true,"messages":[{"role":"user","content":"hello"}]}`))
				req.Header.Set("Authorization", "Bearer "+key.Token)
				return http.DefaultClient.Do(req)
			}
			done := make(chan struct{})
			go func() {
				defer close(done)
				resp, _ := request(url)
				if resp != nil {
					resp.Body.Close()
				}
			}()
			deadline := time.Now().Add(10 * time.Second)
			for {
				var blocked bool
				if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'INSERT INTO business_account_credit_reservations%' AND pid<>pg_backend_pid())`).Scan(&blocked); err != nil {
					t.Fatal(err)
				}
				if blocked {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("real wallet admission did not reach blocked INSERT")
				}
				time.Sleep(10 * time.Millisecond)
			}
			crash()
			if err := lock.Rollback(); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("crashed request did not exit")
			}
			recovered, _ := startRecoveryDoor(t, s)
			var held, reservations int64
			if err := s.db.QueryRow(`SELECT COALESCE(SUM(COALESCE(actual_credits,reserved_credits)),0) FROM ai_key_charges`).Scan(&held); err != nil {
				t.Fatal(err)
			}
			if err := s.db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_reservations`).Scan(&reservations); err != nil {
				t.Fatal(err)
			}
			if held != 0 || reservations != 0 || dispatched.Load() != 0 {
				t.Fatalf("pre-dispatch crash stranded state: cap=%d reservations=%d provider_calls=%d", held, reservations, dispatched.Load())
			}
			assertCharged(t, s, 0)
			resp, err := request(recovered)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("fresh operation cannot use recovered cap: %d %s", resp.StatusCode, body)
			}
			assertCharged(t, s, 200)
		})
	}
}
