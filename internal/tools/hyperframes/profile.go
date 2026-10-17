package hyperframes

import "strconv"

// RenderProfile records fixed owner policy; callers cannot supply arbitrary
// flags or relax resource limits. The full value participates in reuse identity.
type RenderProfile struct {
	ID                   string `json:"id"`
	CaptureWorkers       int    `json:"capture_workers"`
	LowMemoryMode        bool   `json:"low_memory_mode"`
	BrowserGPU           bool   `json:"browser_gpu"`
	BestEffort           bool   `json:"best_effort"`
	FrameFormat          string `json:"frame_format"`
	FramesCache          string `json:"frames_cache"`
	Quality              string `json:"quality"`
	Slice                string `json:"slice"`
	TasksMax             int    `json:"tasks_max"`
	CPUWeight            int    `json:"cpu_weight"`
	DecodeThreads        int    `json:"decode_threads"`
	OMPThreads           int    `json:"omp_threads"`
	BLASThreads          int    `json:"blas_threads"`
	TemporaryDiskBytes   int64  `json:"temporary_disk_bytes"`
	LogBytes             int64  `json:"log_bytes"`
	TaskPressureSampleMS int    `json:"task_pressure_sample_ms"`
}

func fixedRenderProfile() RenderProfile {
	return RenderProfile{
		ID: "strict-software-one-worker-png-no-cache-v2", CaptureWorkers: 1,
		LowMemoryMode: true, FrameFormat: "png", FramesCache: "off", Quality: "high",
		Slice: "vrooli-test.slice", TasksMax: 512, CPUWeight: 50, DecodeThreads: 2,
		OMPThreads: 2, BLASThreads: 2, TemporaryDiskBytes: 4 << 30,
		LogBytes: 8 << 20, TaskPressureSampleMS: 250,
	}
}

func (p RenderProfile) renderArgs(entry, composition, output string) []string {
	args := []string{entry, "render", composition, "--workers", strconv.Itoa(p.CaptureWorkers)}
	if p.LowMemoryMode {
		args = append(args, "--low-memory-mode")
	}
	if !p.BrowserGPU {
		args = append(args, "--no-browser-gpu")
	}
	if !p.BestEffort {
		args = append(args, "--no-best-effort")
	}
	return append(args, "--video-frame-format", p.FrameFormat, "--frames-cache-dir", p.FramesCache, "--quality", p.Quality, "--output", output)
}
