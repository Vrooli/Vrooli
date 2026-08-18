package hostinventory

import (
	"strconv"
	"strings"

	"github.com/vrooli/binaryfetch"
)

// AcquisitionFacts exposes only facts that were actually observed. In
// particular, a missing NVIDIA probe is not represented as a zero-valued
// compute capability: absence must not satisfy a predicate.
func (s Snapshot) AcquisitionFacts() binaryfetch.Facts {
	facts := binaryfetch.Facts{}
	if value := strings.TrimSpace(s.OS); value != "" {
		facts["os"] = value
	}
	if value := strings.TrimSpace(s.Arch); value != "" {
		facts["arch"] = value
	}
	var highest float64
	var highestText string
	for _, gpu := range s.GPUs {
		if gpu.Source != "nvidia-smi" || strings.TrimSpace(gpu.CUDAComputeCapability) == "" {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(gpu.CUDAComputeCapability), 64)
		if err != nil || (highestText != "" && value <= highest) {
			continue
		}
		highest = value
		highestText = strings.TrimSpace(gpu.CUDAComputeCapability)
	}
	if highestText != "" {
		facts["gpu.cuda_compute"] = highestText
	}
	return facts
}
