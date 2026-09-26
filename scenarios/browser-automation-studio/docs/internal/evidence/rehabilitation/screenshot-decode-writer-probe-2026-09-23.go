// Command screenshot-decode-writer-probe measures Go heap and Linux PSS while
// concurrent real FileWriter calls validate the retained full-page PNG.
// Run from scenarios/browser-automation-studio/api with a writer count from 1–100.
package main

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	writer "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/storage"
)

type blockingStore struct {
	*storage.MemoryStorage
	arrived chan struct{}
	release chan struct{}
}

func (s *blockingStore) StoreScreenshot(ctx context.Context, id uuid.UUID, name string, data []byte, contentType string) (*storage.ScreenshotInfo, error) {
	s.arrived <- struct{}{}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.release:
	}
	return s.MemoryStorage.StoreScreenshot(ctx, id, name, data, contentType)
}

func pssKiB() uint64 {
	data, err := os.ReadFile("/proc/self/smaps_rollup")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Pss:") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				value, _ := strconv.ParseUint(fields[1], 10, 64)
				return value
			}
		}
	}
	return 0
}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		panic("usage: screenshot-decode-writer-probe <concurrent-writers-1..100> [--gc-reclaim]")
	}
	gcReclaim := len(os.Args) == 3 && os.Args[2] == "--gc-reclaim"
	if len(os.Args) == 3 && !gcReclaim {
		panic("the only optional argument is --gc-reclaim")
	}
	count, err := strconv.Atoi(os.Args[1])
	if err != nil || count < 1 || count > 100 {
		panic("concurrent writer count must be in 1..100")
	}

	imagePath := "automation/execution-writer/testdata/full-page-1280x12800.png"
	file, err := os.Open(imagePath)
	if err != nil {
		panic(err)
	}
	imageConfig, err := png.DecodeConfig(file)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		panic(err)
	}
	data, err := os.ReadFile(imagePath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("fixture=%s width=%d height=%d encoded_bytes=%d\n", imagePath, imageConfig.Width, imageConfig.Height, len(data))

	dir, err := os.MkdirTemp("", "bas-screenshot-decode-probe-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	arrived := make(chan struct{}, count)
	release := make(chan struct{})
	store := &blockingStore{MemoryStorage: storage.NewMemoryStorage(), arrived: arrived, release: release}
	fileWriter := writer.NewFileWriter(nil, store, nil, writer.NewStaticRoot(dir))
	start := make(chan struct{})
	var ready, done sync.WaitGroup
	ready.Add(count)
	done.Add(count)
	errors := make(chan error, count)
	for i := 0; i < count; i++ {
		go func(index int) {
			defer done.Done()
			ready.Done()
			<-start
			id := uuid.New()
			plan := contracts.ExecutionPlan{ExecutionID: id, WorkflowID: uuid.New()}
			outcome := contracts.StepOutcome{
				ExecutionID: id,
				StepIndex:   index,
				StepType:    "screenshot",
				Success:     true,
				Screenshot:  &contracts.Screenshot{Data: data, MediaType: "image/png", Width: imageConfig.Width, Height: imageConfig.Height},
			}
			_, err := fileWriter.RecordStepOutcome(context.Background(), plan, outcome)
			errors <- err
		}(i)
	}
	ready.Wait()
	runtime.GC()
	basePSS := pssKiB()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	var peakPSS atomic.Uint64
	peakPSS.Store(basePSS)
	stopSample := make(chan struct{})
	sampleDone := make(chan struct{})
	go func() {
		defer close(sampleDone)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				current := pssKiB()
				for peak := peakPSS.Load(); current > peak && !peakPSS.CompareAndSwap(peak, current); peak = peakPSS.Load() {
				}
			case <-stopSample:
				return
			}
		}
	}()

	started := time.Now()
	close(start)
	for i := 0; i < count; i++ {
		select {
		case <-arrived:
		case <-time.After(10 * time.Second):
			close(release)
			panic(fmt.Sprintf("%d writers did not reach storage", count))
		}
	}
	var atBarrier runtime.MemStats
	runtime.ReadMemStats(&atBarrier)
	close(release)
	done.Wait()
	close(stopSample)
	<-sampleDone
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	for i := 0; i < count; i++ {
		if err := <-errors; err != nil {
			panic(err)
		}
	}
	fmt.Printf("concurrent_writers=%d all_decodes_reached_storage=true barrier_ms=%d baseline_pss_kib=%d peak_pss_kib=%d live_heap_delta_at_barrier=%d allocated_bytes=%d\n",
		count, time.Since(started).Milliseconds(), basePSS, peakPSS.Load(), atBarrier.HeapAlloc-before.HeapAlloc, after.TotalAlloc-before.TotalAlloc)
	if gcReclaim {
		beforeGC := pssKiB()
		gcStarted := time.Now()
		runtime.GC()
		gcElapsed := time.Since(gcStarted)
		afterGC := pssKiB()
		var afterGCStats runtime.MemStats
		runtime.ReadMemStats(&afterGCStats)
		freeStarted := time.Now()
		debug.FreeOSMemory()
		freeElapsed := time.Since(freeStarted)
		fmt.Printf("reclaim_experiment=true before_gc_pss_kib=%d after_gc_pss_kib=%d after_free_os_memory_pss_kib=%d heap_alloc_after_gc=%d heap_inuse_after_gc=%d heap_released_after_gc=%d gc_elapsed_ms=%d free_os_memory_elapsed_ms=%d\n",
			beforeGC, afterGC, pssKiB(), afterGCStats.HeapAlloc, afterGCStats.HeapInuse, afterGCStats.HeapReleased, gcElapsed.Milliseconds(), freeElapsed.Milliseconds())
	}
}
