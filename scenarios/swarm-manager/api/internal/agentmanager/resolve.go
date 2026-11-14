package agentmanager

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"swarm-manager/internal/projectroot"
)

// MissingPath is one acceptance-allow glob whose literal-prefix path does
// not exist under the project root and is not declared by `creates`.
type MissingPath struct {
	Glob     string `json:"glob"`
	Resolved string `json:"resolved"`
	Reason   string `json:"reason"`
}

// StalePlanError is returned by resolveScopeAndRoot when one or more
// acceptance_allow globs reference paths that no longer exist on disk and
// are not declared by `creates`. Callers should re-classify this error
// to the structured "plan_stale" wire format and surface it to the UI as
// a re-workshop prompt rather than a raw failure.
type StalePlanError struct {
	ProjectRoot     string        `json:"project_root"`
	AcceptanceAllow []string      `json:"acceptance_allow,omitempty"`
	MissingPaths    []MissingPath `json:"missing_paths"`
}

func (e *StalePlanError) Error() string {
	if e == nil || len(e.MissingPaths) == 0 {
		return "plan stale: no missing paths recorded"
	}
	parts := make([]string, 0, len(e.MissingPaths))
	for _, mp := range e.MissingPaths {
		if mp.Resolved == "" {
			parts = append(parts, fmt.Sprintf("glob %q: %s", mp.Glob, mp.Reason))
		} else {
			parts = append(parts, fmt.Sprintf("glob %q: path %q: %s", mp.Glob, mp.Resolved, mp.Reason))
		}
	}
	return fmt.Sprintf("plan references paths that no longer exist under %q: %s", e.ProjectRoot, strings.Join(parts, "; "))
}

// AsStalePlanError returns the typed error if err wraps one, else nil.
func AsStalePlanError(err error) *StalePlanError {
	var spe *StalePlanError
	if errors.As(err, &spe) {
		return spe
	}
	return nil
}

// resolveScopeAndRoot returns the effective ScopePath and ProjectRoot for a
// spawn request. Caller-supplied values are honored unless they are empty or
// the legacy placeholder ".". In those cases the projectroot resolver fills
// the gap, deriving the target scenario from acceptance_allow.
//
// When the resolved ProjectRoot is absolute and acceptance_allow is non-empty,
// a fail-closed acceptance check confirms each glob's literal-prefix path
// exists under the root or is declared in `creates`. Failures surface as
// *StalePlanError so callers can render the structured "plan_stale" UX.
func resolveScopeAndRoot(reqScope, reqRoot string, acceptanceAllow, creates []string) (scope, root string, err error) {
	needScope := isEmptyOrDot(reqScope)
	needRoot := isEmptyOrDot(reqRoot)

	var res projectroot.Resolution
	if needScope || needRoot {
		res, err = projectroot.Resolve(projectroot.Options{AcceptanceAllow: acceptanceAllow})
		if err != nil {
			return "", "", fmt.Errorf("resolve project root: %w", err)
		}
	}

	scope = strings.TrimSpace(reqScope)
	if needScope {
		scope = res.ScopePath
	}
	root = strings.TrimSpace(reqRoot)
	if needRoot {
		root = res.ProjectRoot
	}

	if filepath.IsAbs(root) && len(acceptanceAllow) > 0 {
		report, validateErr := projectroot.ValidateAcceptance(root, acceptanceAllow, creates)
		if validateErr != nil {
			if errors.Is(validateErr, projectroot.ErrAcceptanceMismatch) {
				missing := make([]MissingPath, 0, len(report.Problems))
				for _, p := range report.Problems {
					missing = append(missing, MissingPath{
						Glob:     p.Glob,
						Resolved: p.ResolvedRel,
						Reason:   p.Reason,
					})
				}
				return "", "", &StalePlanError{
					ProjectRoot:     root,
					AcceptanceAllow: append([]string(nil), acceptanceAllow...),
					MissingPaths:    missing,
				}
			}
			return "", "", fmt.Errorf("validate acceptance against project root %q: %w", root, validateErr)
		}
	}

	return scope, root, nil
}

func isEmptyOrDot(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || s == "."
}
