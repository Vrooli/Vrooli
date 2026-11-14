package ai

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/storage"
	"image-tools/internal/technique"
)

// TestProcessRSSHelper is re-executed by TestConcurrentBackendExecutionsKeepRSSPerChild.
func TestProcessRSSHelper(t *testing.T) {
	text := os.Getenv("IMAGE_TOOLS_RSS_HELPER_BYTES")
	if text == "" {
		return
	}
	size, err := strconv.Atoi(text)
	if err != nil || size <= 0 {
		t.Fatalf("invalid helper size %q", text)
	}
	allocation := make([]byte, size)
	for i := 0; i < len(allocation); i += 4096 {
		allocation[i] = byte(i)
	}
	runtime.KeepAlive(allocation)
	time.Sleep(100 * time.Millisecond)
}

func TestConcurrentBackendExecutionsKeepRSSPerChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test uses a POSIX shell; Windows child RSS remains unavailable")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	process := &backendProcess{}
	sizes := []int{8 << 20, 64 << 20}
	results := make([]int64, len(sizes))
	errs := make([]error, len(sizes))
	var wg sync.WaitGroup
	for i, size := range sizes {
		wg.Add(1)
		go func(i, size int) {
			defer wg.Done()
			script := fmt.Sprintf("export IMAGE_TOOLS_RSS_HELPER_BYTES=%d; exec \"$1\" -test.run=^TestProcessRSSHelper$", size)
			_, peakRSS, measured, err := process.execute(context.Background(), "/bin/sh", []string{"-c", script, "rss-helper", executable}, nil)
			if err == nil && !measured {
				err = fmt.Errorf("child wait result did not expose peak RSS")
			}
			results[i], errs[i] = peakRSS, err
		}(i, size)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("execution %d: %v", i, err)
		}
	}
	if results[1] <= results[0]+(16<<20) {
		t.Fatalf("concurrent children were not attributed independently: small=%d big=%d bytes", results[0], results[1])
	}
}

func TestWarmAndInjectedRunnersDoNotClaimPerJobRSS(t *testing.T) {
	warm := &backendProcess{warm: &fakeWarmRunner{}}
	_, warmRSS, warmMeasured, err := warm.execute(context.Background(), "unused", nil, nil)
	if err != nil || warmMeasured || warmRSS != 0 {
		t.Fatalf("warm execution RSS = %d, %v, %v", warmRSS, warmMeasured, err)
	}
	injected := &backendProcess{run: func(context.Context, string, []string) error { return nil }}
	_, injectedRSS, injectedMeasured, err := injected.execute(context.Background(), "unused", nil, nil)
	if err != nil || injectedMeasured || injectedRSS != 0 {
		t.Fatalf("injected runner RSS = %d, %v, %v", injectedRSS, injectedMeasured, err)
	}
}

func TestExecProviderEmitsExplicitChildRSSMetadata(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test uses a POSIX shell; Windows child RSS remains unavailable")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := "export IMAGE_TOOLS_RSS_HELPER_BYTES=16777216; exec \"$1\" -test.run=^TestProcessRSSHelper$"
	provider := &execProvider{
		backendProcess: backendProcess{name: "test-process", program: "/bin/sh"},
		techniques: technique.Single("caption", func(backends.Request, string) ([]string, error) {
			return []string{"-c", script, "rss-helper", executable}, nil
		}),
	}
	result, err := provider.Execute(context.Background(), backends.Request{
		Operation: "caption",
		Output:    storage.OutputTarget{LocalPath: "/tmp/unused-output"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta["process_peak_rss_source"] != "child_wait4_rusage_maxrss" {
		t.Fatalf("RSS source metadata = %#v", result.Meta)
	}
	if bytes, err := strconv.ParseInt(result.Meta["process_peak_rss_bytes"], 10, 64); err != nil || bytes <= 0 {
		t.Fatalf("RSS byte metadata = %#v (parse error %v)", result.Meta, err)
	}
}
