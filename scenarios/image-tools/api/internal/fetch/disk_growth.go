package fetch

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vrooli/vrooli/packages/capacity"
)

// DiskGrowthGuard turns streaming progress into a lease extension before the
// next chunk is written. The admitted path peaks include staging/final-copy
// headroom; if that bounded headroom is exhausted, it refuses further writes.
type DiskGrowthGuard struct {
	mu        sync.Mutex
	lease     capacity.DiskJobLease
	paths     []capacity.DiskJobPath
	perFile   map[string]int64
	threshold int64
	expansion int64
	ceiling   int64
}

func NewDiskGrowthGuard(lease capacity.DiskJobLease, request capacity.DiskJobRequest, expansion int, ceiling int64) (*DiskGrowthGuard, error) {
	if lease == nil || len(request.Paths) == 0 {
		return nil, errors.New("disk growth guard requires a live lease and complete path plan")
	}
	if expansion < 1 {
		expansion = 1
	}
	if ceiling <= 0 {
		ceiling = InstallOwnerCeilingBytes
	}
	var rootPeak int64
	for _, path := range request.Paths {
		if diskJobPathHasRole(path.Role, "model-adapter-staging-and-final") {
			rootPeak = path.PeakBytes
			break
		}
	}
	if rootPeak <= 0 {
		return nil, errors.New("disk growth guard requires the model/adapter staging path")
	}
	threshold := rootPeak
	if threshold <= 0 {
		return nil, errors.New("disk growth guard has no positive initial estimate")
	}
	return &DiskGrowthGuard{lease: lease, paths: append([]capacity.DiskJobPath(nil), request.Paths...), perFile: map[string]int64{}, threshold: threshold, expansion: int64(expansion), ceiling: ceiling}, nil
}

func (g *DiskGrowthGuard) Observe(ctx context.Context, path string, done, _ int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if done < 0 {
		return fmt.Errorf("disk progress reported a negative byte count")
	}
	key, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if done < g.perFile[key] {
		done = g.perFile[key]
	}
	g.perFile[key] = done
	var total int64
	for _, size := range g.perFile {
		if total > math.MaxInt64-size {
			return errors.New("disk progress size overflow")
		}
		total += size
	}
	if total <= g.threshold {
		return nil
	}
	if total > math.MaxInt64/4/g.expansion {
		return fmt.Errorf("observed model acquisition size overflows its bounded lease")
	}
	newThreshold := total * 2
	newRootPeak := newThreshold * 2 * g.expansion
	updated := append([]capacity.DiskJobPath(nil), g.paths...)
	var estimate int64
	for i := range updated {
		var desiredPeak int64
		if diskJobPathHasRole(updated[i].Role, "model-adapter-staging-and-final") {
			desiredPeak = newRootPeak
		}
		if diskJobPathHasRole(updated[i].Role, "installer-and-model-tool-temporary") {
			tempPeak := max(int64(128<<20), newThreshold/4)
			if desiredPeak > math.MaxInt64-tempPeak {
				return errors.New("extended model install peak overflow")
			}
			desiredPeak += tempPeak
		}
		if diskJobPathHasRole(updated[i].Role, "huggingface-hub-cache") || diskJobPathHasRole(updated[i].Role, "huggingface-transformers-cache") {
			if desiredPeak > math.MaxInt64-newThreshold {
				return errors.New("extended model install peak overflow")
			}
			desiredPeak += newThreshold
		} else if diskJobPathHasRole(updated[i].Role, "huggingface-home-cache") && desiredPeak == 0 {
			desiredPeak = 1
		}
		if desiredPeak > 0 {
			updated[i].PeakBytes = max(updated[i].PeakBytes, desiredPeak)
		}
		if estimate > math.MaxInt64-updated[i].PeakBytes {
			return errors.New("extended model install peak overflow")
		}
		estimate += updated[i].PeakBytes
	}
	if estimate > g.ceiling {
		return fmt.Errorf("observed install growth requires %d bytes, above the %d-byte planning ceiling", estimate, g.ceiling)
	}
	if _, err := g.lease.Extend(ctx, updated); err != nil {
		return fmt.Errorf("shared lease extension denied before further writes: %w", err)
	}
	g.paths = updated
	g.threshold = newThreshold
	return nil
}

func diskJobPathHasRole(roles, wanted string) bool {
	for _, role := range strings.Split(roles, ",") {
		if strings.TrimSpace(role) == wanted {
			return true
		}
	}
	return false
}

func DiskGrowthObserver(guard *DiskGrowthGuard) func(context.Context, string, int64, int64) error {
	if guard == nil {
		return nil
	}
	return guard.Observe
}
