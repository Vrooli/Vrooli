package imageengine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeJobs is image-tools' job surface as Backdrop Studio sees it: a REST
// submit edge that queues one generation, a WaitJob that blocks server-side
// until the job is terminal or the caller leaves, and a CancelJob.
//
// It models the property the defect hinged on: a WaitJob whose caller goes
// away does NOT stop the job. Only CancelJob does.
type fakeJobs struct {
	t         *testing.T
	estimated int

	mu        sync.Mutex
	state     string
	waits     int
	cancelled chan string
	finished  chan struct{}
	failWaits int
}

func newFakeJobs(t *testing.T, estimatedSeconds int) *fakeJobs {
	return &fakeJobs{t: t, estimated: estimatedSeconds, state: "JOB_STATE_RUNNING", cancelled: make(chan string, 4), finished: make(chan struct{})}
}

// finish moves the job to SUCCEEDED, releasing every blocked WaitJob.
func (f *fakeJobs) finish() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state == "JOB_STATE_RUNNING" {
		f.state = "JOB_STATE_SUCCEEDED"
		close(f.finished)
	}
}

func (f *fakeJobs) waitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.waits
}

func (f *fakeJobs) jobJSON() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	job := map[string]any{"id": "job-1", "state": f.state, "estimatedSeconds": f.estimated}
	if f.state == "JOB_STATE_SUCCEEDED" {
		job["resultRef"] = "out/result.png"
	}
	body, _ := json.Marshal(map[string]any{"job": job})
	return body
}

func (f *fakeJobs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/api/v1/ai/text_to_image":
		body, _ := json.Marshal(map[string]any{"job_id": "job-1", "model_id": "sd-1.5", "tier": "local-cpu", "estimated_seconds": f.estimated})
		_, _ = w.Write(body)
	case "/vrooli.image_tools.v1.jobs.JobsService/WaitJob":
		// Reading the body is what lets net/http notice the caller hanging
		// up; a handler that never reads it never sees its context end.
		_, _ = io.Copy(io.Discard, r.Body)
		f.mu.Lock()
		f.waits++
		failing := f.failWaits > 0
		if failing {
			f.failWaits--
		}
		f.mu.Unlock()
		if failing {
			http.Error(w, `{"code":"unavailable","message":"restarting"}`, http.StatusServiceUnavailable)
			return
		}
		select {
		case <-f.finished:
			_, _ = w.Write(f.jobJSON())
		case <-r.Context().Done():
			// The caller left. image-tools logs this as 499 and the job
			// keeps running.
		}
	case "/vrooli.image_tools.v1.jobs.JobsService/GetJob":
		_, _ = w.Write(f.jobJSON())
	case "/vrooli.image_tools.v1.jobs.JobsService/CancelJob":
		var req struct {
			ID string `json:"id"`
		}
		require.NoError(f.t, json.NewDecoder(r.Body).Decode(&req))
		f.mu.Lock()
		f.state = "JOB_STATE_CANCELED"
		f.mu.Unlock()
		f.cancelled <- req.ID
		_, _ = w.Write(f.jobJSON())
	case "/api/v1/blobs/out/result.png":
		_, _ = w.Write([]byte{9, 8, 7})
	default:
		f.t.Errorf("unexpected path %s", r.URL.Path)
		http.NotFound(w, r)
	}
}

func clientFor(server *httptest.Server) *Client {
	return &Client{HTTPClient: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
}

func generationRequest() GenerationRequest {
	return GenerationRequest{Width: 512, Height: 320, Prompt: "quiet colonnade"}
}

// TestGenerateCancelsTheImageToolsJobWhenTheCallerAbandons is the regression
// for diffusion jobs running for nobody.
//
// The catalog's render requests were cut by a 15 s proxy timeout while
// image-tools was 18-26 minutes into a CPU Stable Diffusion run. Backdrop
// Studio's WaitJob was cancelled with them and it returned — but it never told
// image-tools, so every job ran to completion with nobody waiting, four at a
// time, holding the host at load 87. An abandoned wait must cancel the job it
// was waiting on, and the cancel must be sent on a context of its own: the
// caller's context is already cancelled, so a cancel sent on it never leaves
// the process.
func TestGenerateCancelsTheImageToolsJobWhenTheCallerAbandons(t *testing.T) {
	jobs := newFakeJobs(t, 1200)
	server := httptest.NewServer(jobs)
	defer server.Close()

	ctx, abandon := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := clientFor(server).Generate(ctx, generationRequest())
		done <- err
	}()

	require.Eventually(t, func() bool { return jobs.waitCount() > 0 }, 5*time.Second, 5*time.Millisecond, "Generate never waited on the job")
	abandon()

	select {
	case err := <-done:
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("Generate did not return after its caller abandoned it")
	}
	select {
	case id := <-jobs.cancelled:
		require.Equal(t, "job-1", id)
	case <-time.After(5 * time.Second):
		t.Fatal("the abandoned image-tools job was never cancelled; it would run to completion for nobody")
	}
}

// TestGenerateReWaitsAcrossChunksForALongJob proves a long render is waited
// for rather than abandoned: each WaitJob is bounded, and an unfinished job is
// waited on again instead of being given up on and cancelled.
func TestGenerateReWaitsAcrossChunksForALongJob(t *testing.T) {
	jobs := newFakeJobs(t, 1200)
	server := httptest.NewServer(jobs)
	defer server.Close()
	client := clientFor(server)
	client.Wait = WaitPolicy{Chunk: 20 * time.Millisecond}

	go func() {
		require.Eventually(t, func() bool { return jobs.waitCount() >= 4 }, 5*time.Second, 5*time.Millisecond)
		jobs.finish()
	}()

	out, err := client.Generate(context.Background(), generationRequest())
	require.NoError(t, err)
	require.Equal(t, []byte{9, 8, 7}, out.PNG)
	require.GreaterOrEqual(t, jobs.waitCount(), 4, "the job was not re-waited in chunks")
	require.Empty(t, jobs.cancelled, "a job still inside its budget must not be cancelled")
}

// TestGenerateRidesOutATransientWaitFailure keeps an image-tools restart or a
// dropped connection from abandoning a render that is still running.
func TestGenerateRidesOutATransientWaitFailure(t *testing.T) {
	jobs := newFakeJobs(t, 60)
	jobs.failWaits = 2
	server := httptest.NewServer(jobs)
	defer server.Close()
	client := clientFor(server)
	client.Wait = WaitPolicy{Chunk: 50 * time.Millisecond, RetryBackoff: time.Millisecond}

	go func() {
		require.Eventually(t, func() bool { return jobs.waitCount() >= 3 }, 5*time.Second, 5*time.Millisecond)
		jobs.finish()
	}()

	out, err := client.Generate(context.Background(), generationRequest())
	require.NoError(t, err)
	require.Equal(t, []byte{9, 8, 7}, out.PNG)
	require.Empty(t, jobs.cancelled)
}

// TestGenerateCancelsAJobThatOutrunsItsBudget bounds the wait from the other
// side: a job far past its own estimate is cancelled rather than waited on
// forever, so a wedged backend cannot pin a model and a request indefinitely.
func TestGenerateCancelsAJobThatOutrunsItsBudget(t *testing.T) {
	jobs := newFakeJobs(t, 1)
	server := httptest.NewServer(jobs)
	defer server.Close()
	client := clientFor(server)
	client.Wait = WaitPolicy{Chunk: 10 * time.Millisecond, MinRun: 50 * time.Millisecond, MaxWait: 200 * time.Millisecond}

	_, err := client.Generate(context.Background(), generationRequest())
	require.ErrorIs(t, err, ErrWaitBudgetExhausted)
	select {
	case id := <-jobs.cancelled:
		require.Equal(t, "job-1", id)
	case <-time.After(5 * time.Second):
		t.Fatal("a job past its wait budget was abandoned without being cancelled")
	}
}

// TestWaitBudgetHonoursTheJobsOwnEstimate pins the arithmetic: the budget
// scales with image-tools' estimate, so a 20-minute CPU render is not cut off
// at a constant chosen for a GPU.
func TestWaitBudgetHonoursTheJobsOwnEstimate(t *testing.T) {
	policy := WaitPolicy{}.withDefaults()
	require.Equal(t, policy.Unknown, policy.runBudget(0), "an unknown estimate gets the generous default")
	require.Equal(t, policy.MinRun, policy.runBudget(5), "a tiny estimate still gets the floor")
	require.Equal(t, 3*20*time.Minute, policy.runBudget(20*60), "a CPU-length estimate scales the budget")
	require.Equal(t, policy.MaxWait, policy.runBudget(24*60*60), "the budget never exceeds the ceiling")
}
