package ai

import (
	"strings"
	"testing"
)

func TestParseGPUHoldersReadsNvidiaSmiRowsAndSkipsNoise(t *testing.T) {
	output := "2114578, 5080\n3115860, 1380\n\nnot a row\n12, x\n"
	holders := parseGPUHolders(output, func(pid int) string {
		if pid == 2114578 {
			return "llama-server --model blob"
		}
		return ""
	})
	if len(holders) != 2 || holders[0].PID != 2114578 || holders[0].MiB != 5080 || holders[0].Detail != "llama-server --model blob" || holders[1].MiB != 1380 {
		t.Fatalf("holders = %+v", holders)
	}
}

func TestDescribeGPUHoldersNamesTheLargestFirst(t *testing.T) {
	text := describeGPUHolders([]gpuHolder{{PID: 1, MiB: 600, Detail: "small"}, {PID: 2, MiB: 6640, Detail: "big job"}, {PID: 3, MiB: 90}})
	want := "GPU memory held when it failed: pid 2 6640 MiB (big job); pid 1 600 MiB (small); pid 3 90 MiB"
	if text != want {
		t.Fatalf("got %q want %q", text, want)
	}
	if describeGPUHolders(nil) != "" {
		t.Fatal("no holders must render nothing")
	}
	if strings.Contains(describeGPUHolders([]gpuHolder{{PID: 9, MiB: 1}}), "()") {
		t.Fatal("a process that is gone must not render empty parentheses")
	}
}
