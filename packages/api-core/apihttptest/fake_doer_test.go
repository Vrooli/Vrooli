package apihttptest

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
)

func TestFakeDoerQueuesResponsesAndErrorsInOrder(t *testing.T) {
	var fake FakeDoer
	fake.AddResponse(http.StatusOK, []byte("first"))
	wantErr := errors.New("transport down")
	fake.AddError(wantErr)
	fake.AddResponse(http.StatusCreated, []byte("third"))

	req, _ := http.NewRequest(http.MethodGet, "https://example.invalid/", nil)
	first, err := fake.Do(req)
	if err != nil || first.StatusCode != http.StatusOK {
		t.Fatalf("first response = (%v, %v), want status 200", first, err)
	}
	body, _ := io.ReadAll(first.Body)
	_ = first.Body.Close()
	if string(body) != "first" {
		t.Fatalf("first body = %q, want first", body)
	}
	if _, err := fake.Do(req); !errors.Is(err, wantErr) {
		t.Fatalf("queued error = %v, want %v", err, wantErr)
	}
	third, err := fake.Do(req)
	if err != nil || third.StatusCode != http.StatusCreated {
		t.Fatalf("third response = (%v, %v), want status 201", third, err)
	}
}

func TestFakeDoerReportsUnconfiguredCalls(t *testing.T) {
	var fake FakeDoer
	req, _ := http.NewRequest(http.MethodGet, "https://example.invalid/", nil)
	if _, err := fake.Do(req); err == nil {
		t.Fatal("unconfigured request returned no error")
	}
}

func TestFakeDoerRecordsConcurrentRequests(t *testing.T) {
	const count = 50
	var fake FakeDoer
	for i := 0; i < count; i++ {
		fake.AddResponse(http.StatusOK, []byte("ok"))
	}
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, "https://example.invalid/", nil)
			resp, err := fake.Do(req)
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
	}
	wg.Wait()
	if got := fake.Calls.Load(); got != count {
		t.Fatalf("calls = %d, want %d", got, count)
	}
	if got := len(fake.Requests); got != count {
		t.Fatalf("recorded requests = %d, want %d", got, count)
	}
}
