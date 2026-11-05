package opsalert

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

type fakeHTTP struct {
	calls  int
	status []int
}

func (f *fakeHTTP) Do(*http.Request) (*http.Response, error) {
	f.calls++
	if len(f.status) == 0 {
		return nil, errors.New("down")
	}
	code := f.status[0]
	f.status = f.status[1:]
	return &http.Response{StatusCode: code, Body: http.NoBody}, nil
}

func TestTransportRetriesAndStopsOnSuccess(t *testing.T) {
	f := &fakeHTTP{status: []int{503, 202}}
	tr := New()
	tr.UseHTTPClient(f)
	tr.backoff = []time.Duration{0, 0}
	tr.attempts = 3
	if err := tr.Send(context.Background(), "http://example.test", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if f.calls != 2 {
		t.Fatalf("calls = %d, want 2", f.calls)
	}
}

func TestTransportTokenBucket(t *testing.T) {
	tr := New()
	now := time.Unix(100, 0)
	if !tr.Allow("ip", 2, time.Minute, now) || !tr.Allow("ip", 2, time.Minute, now) || tr.Allow("ip", 2, time.Minute, now) {
		t.Fatal("bucket did not enforce burst")
	}
	if !tr.Allow("ip", 2, time.Minute, now.Add(time.Minute)) {
		t.Fatal("bucket did not refill")
	}
}
