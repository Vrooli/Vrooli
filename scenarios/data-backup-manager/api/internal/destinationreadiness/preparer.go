package destinationreadiness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// LocalPreparer performs the small set of local filesystem preparation actions
// that are safe enough for production. Destructive actions intentionally remain
// unsupported until a stricter platform adapter exists.
type LocalPreparer struct {
	MkdirAll    func(string, os.FileMode) error
	WriteFile   func(string, []byte, os.FileMode) error
	Remove      func(string) error
	ProbeSuffix string
}

var _ Preparer = (*LocalPreparer)(nil)

// NewLocalPreparer constructs the production local preparer.
func NewLocalPreparer() *LocalPreparer {
	return &LocalPreparer{
		MkdirAll:    os.MkdirAll,
		Remove:      os.Remove,
		ProbeSuffix: ".vrooli-write-probe",
	}
}

// Supported reports the production support matrix. Only create_subdir is live;
// format, relabel, and cleanup are explicit future work.
func (p *LocalPreparer) Supported(action PreparationAction) (bool, string) {
	if action == ActionCreateSubdir {
		return true, ""
	}
	return false, fmt.Sprintf("%s execution is not implemented", action)
}

// Execute creates the requested backup subdirectory and verifies it is writable
// with an exclusive, short-lived probe file. It refuses every destructive action.
func (p *LocalPreparer) Execute(ctx context.Context, plan Plan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if plan.Action != ActionCreateSubdir {
		return ErrPreparationRefused{Reason: fmt.Sprintf("%s execution is not implemented", plan.Action)}
	}
	target := cleanPath(plan.TargetPath)
	if target == "" {
		return ErrInvalidReadiness{Field: "target_path", Reason: "required"}
	}
	if !within(target, cleanPath(plan.Location)) || target == cleanPath(plan.Location) {
		return ErrPreparationRefused{Reason: "create_subdir target must be a child of the analyzed location"}
	}

	mkdirAll := p.MkdirAll
	if mkdirAll == nil {
		mkdirAll = os.MkdirAll
	}
	writeFile := p.WriteFile
	remove := p.Remove
	if remove == nil {
		remove = os.Remove
	}
	probeSuffix := p.ProbeSuffix
	if probeSuffix == "" {
		probeSuffix = ".vrooli-write-probe"
	}
	if filepath.Base(probeSuffix) != probeSuffix || probeSuffix == "." || probeSuffix == ".." {
		return ErrInvalidReadiness{Field: "probe_suffix", Reason: "must be a filename prefix"}
	}

	if err := mkdirAll(target, 0o700); err != nil {
		return fmt.Errorf("create preparation subdirectory: %w", err)
	}
	probeFile, err := os.CreateTemp(target, probeSuffix+"-*")
	if err != nil {
		return fmt.Errorf("create preparation probe: %w", err)
	}
	probe := probeFile.Name()
	probeContents := []byte("ok\n")
	var probeErr error
	if p.WriteFile != nil {
		if probeErr = probeFile.Close(); probeErr == nil {
			probeErr = writeFile(probe, probeContents, 0o600)
		}
	} else {
		var written int
		written, probeErr = probeFile.Write(probeContents)
		if probeErr == nil && written != len(probeContents) {
			probeErr = fmt.Errorf("short write: wrote %d of %d bytes", written, len(probeContents))
		}
		if probeErr == nil {
			probeErr = probeFile.Sync()
		}
		if closeErr := probeFile.Close(); probeErr == nil {
			probeErr = closeErr
		}
	}
	if probeErr != nil {
		if cleanupErr := remove(probe); cleanupErr != nil {
			return fmt.Errorf("write preparation probe: %v (cleanup failed: %w)", probeErr, cleanupErr)
		}
		return fmt.Errorf("write preparation probe: %w", probeErr)
	}
	if err := remove(probe); err != nil {
		return fmt.Errorf("remove preparation probe: %w", err)
	}
	return nil
}
