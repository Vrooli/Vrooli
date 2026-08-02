package mocks

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFakeDoer_AddResponseRoundTrips(t *testing.T) {
	var f FakeDoer
	f.AddResponse(http.StatusOK, []byte("hello"))

	req, _ := http.NewRequest(http.MethodGet, "https://example.invalid/x", nil)
	resp, err := f.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, "hello", string(body))
	require.Equal(t, int64(1), f.Calls.Load())
	require.Len(t, f.Requests, 1)
	require.Equal(t, http.MethodGet, f.Requests[0].Method)
}

func TestFakeDoer_RespondsInOrder(t *testing.T) {
	var f FakeDoer
	f.AddResponse(http.StatusOK, []byte("first"))
	f.AddResponse(http.StatusCreated, []byte("second"))

	req, _ := http.NewRequest(http.MethodGet, "https://example.invalid/", nil)
	first, err := f.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, first.StatusCode)

	second, err := f.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, second.StatusCode)
}

func TestFakeDoer_AddErrorSurfaces(t *testing.T) {
	var f FakeDoer
	want := errors.New("transport down")
	f.AddError(want)

	req, _ := http.NewRequest(http.MethodGet, "https://example.invalid/", nil)
	_, err := f.Do(req)
	require.ErrorIs(t, err, want)
}

func TestFakeDoer_OutOfRangeError(t *testing.T) {
	var f FakeDoer
	req, _ := http.NewRequest(http.MethodGet, "https://example.invalid/", nil)
	_, err := f.Do(req)
	require.Error(t, err, "no canned response should produce a clear error")
	require.Contains(t, err.Error(), "no canned response")
}

// TestFakeDoer_RaceCleanWhenSharedAcrossGoroutines is the load-bearing
// regression test for the mutex + atomic counter. Run with
// `go test -race`; without the synchronisation the slice append inside
// Do races with itself.
func TestFakeDoer_RaceCleanWhenSharedAcrossGoroutines(t *testing.T) {
	t.Parallel()
	const goroutines = 50
	var f FakeDoer
	for i := 0; i < goroutines; i++ {
		f.AddResponse(http.StatusOK, []byte("ok"))
	}
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, "https://example.invalid/", nil)
			resp, err := f.Do(req)
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int64(goroutines), f.Calls.Load())
	require.Len(t, f.Requests, goroutines)
}
