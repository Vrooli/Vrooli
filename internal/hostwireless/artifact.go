package hostwireless

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/artifactlock"
	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
	"github.com/vrooli/vrooli/packages/capacity"
)

func prepareArtifacts(ctx context.Context, r Receipt, source mt7927source.Profile, spec buildProfile) (_ Receipt, resultErr error) {
	ctx, cancel := context.WithTimeout(ctx, prepareLimit)
	defer cancel()

	if err := source.VerifyStaged(ctx, r.SourceDirectory); err != nil {
		return r, fmt.Errorf("verify governed source: %w", err)
	}
	artifactRoot := filepath.Dir(filepath.Dir(r.SourceDirectory))
	parent := filepath.Join(artifactRoot, "host-wireless", r.Profile)
	_, sourceBudget, err := source.StageBudget()
	if err != nil {
		return r, err
	}
	treeBudget, err := copiedTreeBudget(ctx, filepath.Join("/lib/modules", r.KernelRelease))
	if err != nil {
		return r, fmt.Errorf("size stock module tree: %w", err)
	}
	r.ReservedDiskBytes = sourceBudget + treeBudget + buildAllowance
	broker, err := capacity.NewBroker(ctx, "")
	if err != nil {
		return r, fmt.Errorf("open wireless build disk admission: %w", err)
	}
	defer broker.Close()
	lease, err := broker.ReserveDisk(ctx, "host-wireless", artifactRoot, r.ReservedDiskBytes, 0)
	if err != nil {
		return r, fmt.Errorf("reserve wireless build disk: %w", err)
	}

	// The four-minute prepare deadline is below the disk lease's normal
	// five-minute heartbeat interval.
	cleanupVerified := true
	defer func() {
		if !cleanupVerified {
			return
		}
		done, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		resultErr = errors.Join(resultErr, lease.Release(done))
	}()
	unlock, err := artifactlock.AcquireWithPathContext(ctx, r.Profile, func(string) (string, error) {
		return filepath.Join(parent, ".prepare.lock"), nil
	})
	if err != nil {
		return r, err
	}
	defer unlock()
	work, err := os.MkdirTemp(parent, ".prepare-")
	if err != nil {
		return r, err
	}
	defer func() {
		err := os.RemoveAll(work)
		cleanupVerified = err == nil
		resultErr = errors.Join(resultErr, err)
	}()

	r, err = buildCandidate(ctx, r, source, spec, work)
	if err != nil {
		return r, err
	}
	return publishCandidate(ctx, r, filepath.Join(work, "candidate"), filepath.Join(parent, strings.TrimPrefix(filepath.Base(work), ".prepare-")))
}

func buildCandidate(ctx context.Context, r Receipt, source mt7927source.Profile, spec buildProfile, work string) (Receipt, error) {
	inputs := filepath.Join(work, "inputs")
	if err := os.CopyFS(inputs, os.DirFS(r.SourceDirectory)); err != nil {
		return r, fmt.Errorf("copy governed source closure: %w", err)
	}
	if err := source.VerifyStaged(ctx, inputs); err != nil {
		return r, fmt.Errorf("verify private source closure: %w", err)
	}
	sourceHash, err := binaryfetch.FileDigestContext(ctx, filepath.Join(inputs, mt7927source.Manifest))
	if err != nil {
		return r, err
	}
	r.SourceSHA256 = sourceHash
	for path, bytes := range map[string][]byte{"adapted.patch": spec.patch, "build.sh": spec.recipe} {
		if err := os.WriteFile(filepath.Join(work, path), bytes, 0600); err != nil {
			return r, err
		}
	}
	headers := filepath.Join("/lib/modules", r.KernelRelease, "build", "Module.symvers")
	if err := runBuild(ctx, work, r.KernelRelease, "stock", nil); err != nil {
		return r, err
	}
	r.StockABI, err = verifyABI(headers, filepath.Join(work, "stock"), spec.modules, nil)
	if err != nil {
		return r, fmt.Errorf("verify stock ABI: %w", err)
	}
	if err := runBuild(ctx, work, r.KernelRelease, "candidate", r.Modaliases); err != nil {
		return r, err
	}
	bundle := filepath.Join(work, "candidate")
	r.CandidateABI, err = verifyABI(headers, bundle, spec.modules, spec.newExports)
	if err != nil {
		return r, fmt.Errorf("verify candidate ABI: %w", err)
	}
	if r.Profile == NobleBluetoothProfile {
		if err := verifyBluetoothQuirk(work, bundle); err != nil {
			return r, fmt.Errorf("verify Bluetooth quirk: %w", err)
		}
	}
	r.CopiedTreeRollback = true
	if err := ctx.Err(); err != nil {
		return r, err
	}
	return r, nil
}

func copiedTreeBudget(ctx context.Context, path string) (int64, error) {
	var bytes int64
	err := filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		bytes += 8192 // File allocation rounding and directory metadata allowance.
		if entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			bytes += info.Size()
		}
		return nil
	})
	return bytes, err
}

func publishCandidate(ctx context.Context, r Receipt, bundle, path string) (Receipt, error) {
	entries, err := os.ReadDir(bundle)
	if err != nil {
		return r, err
	}
	hashes := make(map[string]string, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return r, fmt.Errorf("unexpected candidate artifact %s", entry.Name())
		}
		hash, err := binaryfetch.FileDigestContext(ctx, filepath.Join(bundle, entry.Name()))
		if err != nil {
			return r, err
		}
		hashes[entry.Name()] = hash
	}
	published := r
	published.State, published.ArtifactDirectory = preparedArtifactState, path
	published.ArtifactSHA256 = hashes
	data, err := json.MarshalIndent(published, "", "  ")
	if err != nil {
		return r, err
	}
	if err := os.WriteFile(filepath.Join(bundle, "receipt.json"), data, 0600); err != nil {
		return r, err
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if err := os.Rename(bundle, path); err != nil {
		return r, err
	}
	return published, nil
}
