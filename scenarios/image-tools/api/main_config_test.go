package main

import (
	"encoding/json"
	"testing"
	"time"

	internalai "image-tools/internal/ai"
	"image-tools/internal/backends"
	internaljobs "image-tools/internal/jobs"
)

func TestLoadRuntimeConfigDefaults(t *testing.T) {
	t.Setenv(cpuWorkersEnv, "")
	t.Setenv(installMBPerSecondEnv, "")

	cfg, err := loadRuntimeConfig()
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if cfg.CPUWorkers != defaultCPUWorkers {
		t.Fatalf("CPUWorkers = %d, want %d", cfg.CPUWorkers, defaultCPUWorkers)
	}
	if cfg.InstallMBPerSecond != 15 {
		t.Fatalf("InstallMBPerSecond = %d, want 15", cfg.InstallMBPerSecond)
	}
}

func TestLoadRuntimeConfigFromEnv(t *testing.T) {
	t.Setenv(cpuWorkersEnv, "6")
	t.Setenv(installMBPerSecondEnv, "45")

	cfg, err := loadRuntimeConfig()
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if cfg.CPUWorkers != 6 || cfg.InstallMBPerSecond != 45 {
		t.Fatalf("config = %+v, want CPUWorkers=6 InstallMBPerSecond=45", cfg)
	}
}

func TestLoadRuntimeConfigRejectsInvalidEnv(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "cpu non integer", key: cpuWorkersEnv, value: "many"},
		{name: "cpu below min", key: cpuWorkersEnv, value: "0"},
		{name: "cpu above max", key: cpuWorkersEnv, value: "33"},
		{name: "throughput non integer", key: installMBPerSecondEnv, value: "fast"},
		{name: "throughput below min", key: installMBPerSecondEnv, value: "0"},
		{name: "throughput above max", key: installMBPerSecondEnv, value: "10001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(cpuWorkersEnv, "")
			t.Setenv(installMBPerSecondEnv, "")
			t.Setenv(tt.key, tt.value)
			if _, err := loadRuntimeConfig(); err == nil {
				t.Fatal("expected invalid runtime config to fail")
			}
		})
	}
}

func TestJobSampleAndTraceExtractsAIMetadata(t *testing.T) {
	payload, err := json.Marshal(internalai.Payload{
		Operation: "text_to_image",
		ModelID:   "sd-1.5",
		Backend:   "stable-diffusion.cpp",
		Tier:      backends.TierLocalCPU.String(),
		GPU:       true,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	created := time.Unix(100, 0)
	started := created.Add(250 * time.Millisecond)
	finished := started.Add(1500 * time.Millisecond)
	job := internaljobs.Job{
		ID:         "job-1",
		Operation:  "text_to_image",
		Lane:       internaljobs.LaneGPU,
		State:      internaljobs.StateSucceeded,
		ResultRef:  "out/1.png",
		Payload:    payload,
		CreatedAt:  created,
		StartedAt:  &started,
		FinishedAt: &finished,
	}

	sample, trace := jobSampleAndTrace(job)
	if sample.ModelID != "sd-1.5" || sample.Tier != "local-cpu" || !sample.FallbackUsed {
		t.Fatalf("sample metadata = %+v", sample)
	}
	if sample.DurationMS != 1500 || sample.QueueWaitMS != 250 {
		t.Fatalf("sample timing = %d/%d", sample.DurationMS, sample.QueueWaitMS)
	}
	if trace.JobID != "job-1" || trace.Backend != "stable-diffusion.cpp" || trace.ResultRef != "out/1.png" {
		t.Fatalf("trace metadata = %+v", trace)
	}
}
