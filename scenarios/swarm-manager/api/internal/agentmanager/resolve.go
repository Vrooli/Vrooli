package agentmanager

import (
	"fmt"
	"path/filepath"
	"strings"

	"swarm-manager/internal/projectroot"
)

// resolveScopeAndRoot returns the effective ScopePath and ProjectRoot for a
// spawn request. Caller-supplied values are honored unless they are empty or
// the legacy placeholder ".". In those cases the projectroot resolver fills
// the gap, deriving the target scenario from acceptance_allow.
//
// When the resolved ProjectRoot is absolute and acceptance_allow is non-empty,
// a fail-closed check confirms each glob's literal-prefix path exists under
// the root before the request is sent to agent-manager.
func resolveScopeAndRoot(reqScope, reqRoot string, acceptanceAllow []string) (scope, root string, err error) {
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
		if validateErr := projectroot.ValidateAcceptanceUnderRoot(root, acceptanceAllow); validateErr != nil {
			return "", "", fmt.Errorf("validate acceptance against project root %q: %w", root, validateErr)
		}
	}

	return scope, root, nil
}

func isEmptyOrDot(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || s == "."
}
