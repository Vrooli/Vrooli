package quality

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image-tools/internal/modelenv"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"image-tools/internal/sidecar"
)

// MetricLPIPS is the learned perceptual distance between candidate and
// reference (lower is better). Unlike PSNR/SSIM it does not reward blur, so it
// is the pixel-reference metric that tracks how an upscaler's result looks.
const MetricLPIPS = "reference.lpips"

// lpipsTimeout bounds one pair; a 12-megapixel pair is scored in tiles on CPU.
const lpipsTimeout = 10 * time.Minute

// Perceptual scores the learned perceptual distance of one image pair. The pair
// must share one size; nothing is resized to make it comparable.
type Perceptual interface {
	Distance(ctx context.Context, reference, output []byte) (float64, error)
}

// SidecarLPIPS runs the in-repo sidecar's lpips_metric module inside Image
// Tools' private Python environment with the catalog-provisioned AlexNet
// backbone, so no weights are fetched at scoring time.
type SidecarLPIPS struct {
	// Python is the private environment's interpreter.
	Python string
	// Backbone is the pinned AlexNet ImageNet state dict the model catalog installs.
	Backbone string
}

func (s *SidecarLPIPS) Distance(ctx context.Context, reference, output []byte) (float64, error) {
	if s == nil || s.Python == "" || s.Backbone == "" {
		return 0, errors.New("perceptual metric is not configured")
	}
	if _, err := os.Stat(s.Backbone); err != nil {
		return 0, fmt.Errorf("LPIPS backbone is not installed (%v): install the lpips-alexnet model first", err)
	}
	dir, err := os.MkdirTemp("", "image-tools-lpips-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(dir)
	refPath, outPath := filepath.Join(dir, "reference"), filepath.Join(dir, "candidate")
	if err := os.WriteFile(refPath, reference, 0o600); err != nil {
		return 0, err
	}
	if err := os.WriteFile(outPath, output, 0o600); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, lpipsTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Python, "-m", "image_tools_sidecar.lpips_metric", "--reference", refPath, "--candidate", outPath, "--backbone", s.Backbone, "--device", "cpu")
	// The admitted claim is two CPU cores; keep torch inside it.
	cmd.Env = modelenv.Apply(append(os.Environ(), "OMP_NUM_THREADS=2", "MKL_NUM_THREADS=2", "IMAGE_TOOLS_CPU_THREADS=2", "CUDA_VISIBLE_DEVICES="), modelenv.ModeInference)
	// torch from the CUDA wheel loads the venv's own CUDA runtime at import, even
	// for CPU work; those libraries are not on the service's loader path.
	if libraries := sidecar.RuntimeLibraries(s.Python); len(libraries) > 0 {
		if current := os.Getenv("LD_LIBRARY_PATH"); current != "" {
			libraries = append(libraries, current)
		}
		cmd.Env = append(cmd.Env, "LD_LIBRARY_PATH="+strings.Join(libraries, string(os.PathListSeparator)))
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		tail := stderr.String()
		if len(tail) > 600 {
			tail = tail[len(tail)-600:]
		}
		return 0, fmt.Errorf("lpips sidecar failed: %w: %s", err, tail)
	}
	var out struct {
		LPIPS float64 `json:"lpips"`
		Net   string  `json:"net"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &out); err != nil {
		return 0, fmt.Errorf("lpips sidecar output: %w", err)
	}
	if out.Net != "alex" || math.IsNaN(out.LPIPS) || math.IsInf(out.LPIPS, 0) || out.LPIPS < 0 {
		return 0, fmt.Errorf("lpips sidecar returned an unusable score %v (net %q)", out.LPIPS, out.Net)
	}
	return out.LPIPS, nil
}

// wantsMetric reports whether any budget gates the named metric, so costly
// metrics are computed only for manifests that ask for them.
func wantsMetric(budgetMetrics []string, metric string) bool {
	for _, m := range budgetMetrics {
		if m == metric {
			return true
		}
	}
	return false
}

func budgetMetricNames(m *qualityv1.EvaluationManifest) []string {
	names := make([]string, 0, len(m.Budgets))
	for _, b := range m.Budgets {
		names = append(names, b.Metric)
	}
	return names
}
