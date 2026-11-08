package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOperationBudgetDeliversReceiptPastOrdinaryWriteDeadline(t *testing.T) {
	handler, err := WithOperationBudget(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := time.NewTimer(150 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
			_, _ = io.WriteString(w, "validated")
		case <-r.Context().Done():
			http.Error(w, r.Context().Err().Error(), http.StatusRequestTimeout)
		}
	}), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Config.WriteTimeout = 50 * time.Millisecond
	server.Start()
	defer server.Close()
	client := server.Client()
	client.Timeout = 2 * time.Second
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("operation receipt lost: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusOK || string(body) != "validated" {
		t.Fatalf("operation receipt: status=%d body=%q err=%v", response.StatusCode, body, err)
	}
}

func TestOperationBudgetEnforcesDeadlineAndPreservesParentCancellation(t *testing.T) {
	for _, cancelParent := range []bool{false, true} {
		t.Run(map[bool]string{false: "budget", true: "caller"}[cancelParent], func(t *testing.T) {
			entered, stopped := make(chan struct{}), make(chan error, 1)
			budget := 20 * time.Millisecond
			if cancelParent {
				budget = time.Second
			}
			handler, err := WithOperationBudget(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(entered)
				<-r.Context().Done()
				stopped <- r.Context().Err()
				http.Error(w, r.Context().Err().Error(), http.StatusRequestTimeout)
			}), budget)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(handler)
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			done := make(chan error, 1)
			go func() {
				resp, err := server.Client().Do(req)
				if resp != nil {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					if !cancelParent && (resp.StatusCode != http.StatusRequestTimeout || !strings.Contains(string(body), "deadline exceeded")) {
						done <- context.Canceled
						return
					}
				}
				done <- err
			}()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("operation never started")
			}
			if cancelParent {
				cancel()
			}
			select {
			case err := <-stopped:
				want := context.DeadlineExceeded
				if cancelParent {
					want = context.Canceled
				}
				if err != want {
					t.Fatalf("operation cancellation=%v want=%v", err, want)
				}
			case <-time.After(time.Second):
				t.Fatal("operation ignored cancellation")
			}
			select {
			case err := <-done:
				if (err != nil) != cancelParent {
					t.Fatalf("client outcome err=%v callerCanceled=%v", err, cancelParent)
				}
			case <-time.After(time.Second):
				t.Fatal("operation did not return")
			}
		})
	}
}

func TestOperationBudgetRejectsUnboundedOrMissingHandlers(t *testing.T) {
	if _, err := WithOperationBudget(nil, time.Second); err == nil {
		t.Fatal("missing handler accepted")
	}
	for _, budget := range []time.Duration{0, -time.Second} {
		if _, err := WithOperationBudget(http.NotFoundHandler(), budget); err == nil {
			t.Fatal("non-positive budget accepted")
		}
	}
}
