package main

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

type hostFactsResponse struct {
	Available            bool     `json:"available"`
	Reason               string   `json:"reason,omitempty"`
	CPUCount             *int     `json:"cpu_count,omitempty"`
	MemoryTotalBytes     *uint64  `json:"memory_total_bytes,omitempty"`
	MemoryAvailableBytes *uint64  `json:"memory_available_bytes,omitempty"`
	DiskFreeBytes        *uint64  `json:"disk_free_bytes,omitempty"`
	GPUs                 []string `json:"gpus,omitempty"`
	Platform             string   `json:"platform,omitempty"`
}

var (
	hostFactsMu       sync.Mutex
	cachedHostFacts   hostFactsResponse
	cachedHostFactsAt time.Time
	hostFactsProbe    = collectHostFacts
)

func collectHostFacts(ctx context.Context) (hostFactsResponse, error) {
	snapshot, err := hostinventory.Collect(ctx)
	response := hostFactsResponse{GPUs: []string{}, Platform: runtime.GOOS + "/" + runtime.GOARCH}
	if snapshot.CPU.Cores > 0 {
		value := snapshot.CPU.Cores
		response.CPUCount = &value
	}
	if snapshot.Memory.TotalBytes > 0 {
		value := snapshot.Memory.TotalBytes
		response.MemoryTotalBytes = &value
	}
	if snapshot.Memory.AvailableBytes > 0 {
		value := snapshot.Memory.AvailableBytes
		response.MemoryAvailableBytes = &value
	}
	for _, gpu := range snapshot.GPUs {
		if gpu.Name != "" {
			response.GPUs = append(response.GPUs, gpu.Name)
		}
	}
	var stat syscall.Statfs_t
	if statErr := syscall.Statfs("/", &stat); statErr == nil {
		free := stat.Bavail * uint64(stat.Bsize)
		response.DiskFreeBytes = &free
	}
	response.Available = response.CPUCount != nil || response.MemoryTotalBytes != nil || response.DiskFreeBytes != nil || len(response.GPUs) > 0
	return response, err
}

func (s *Server) handleV2HostFacts(w http.ResponseWriter, r *http.Request) {
	hostFactsMu.Lock()
	if time.Since(cachedHostFactsAt) < time.Minute {
		response := cachedHostFacts
		hostFactsMu.Unlock()
		writeJSON(w, http.StatusOK, response)
		return
	}
	hostFactsMu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	result := make(chan struct {
		response hostFactsResponse
		err      error
	}, 1)
	go func() {
		response, err := hostFactsProbe(ctx)
		result <- struct {
			response hostFactsResponse
			err      error
		}{response, err}
	}()

	select {
	case outcome := <-result:
		response := outcome.response
		if outcome.err != nil && !response.Available {
			response.Reason = outcome.err.Error()
		}
		if response.Available {
			hostFactsMu.Lock()
			cachedHostFacts, cachedHostFactsAt = response, time.Now()
			hostFactsMu.Unlock()
		}
		writeJSON(w, http.StatusOK, response)
	case <-ctx.Done():
		writeJSON(w, http.StatusOK, hostFactsResponse{Available: false, Reason: "host inventory probe exceeded the 2 second budget"})
	}
}
