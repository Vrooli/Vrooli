package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
	"time"
)

func TestFileMutationOutcome_CarriesFailureMessage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		result  any
		success bool
		message string
	}{
		{
			name:    "failed push carries the git error",
			result:  &PushResponse{Success: false, Error: "git push timed out before the transfer finished"},
			message: "git push timed out before the transfer finished",
		},
		{
			name:    "push that did not move the remote ref carries the verification error",
			result:  &PushResponse{Success: false, VerificationError: "unable to resolve remote ref after push"},
			message: "unable to resolve remote ref after push",
		},
		{
			name:    "failed pull carries the git error",
			result:  &PullResponse{Success: false, Error: "git pull failed: exit status 1"},
			message: "git pull failed: exit status 1",
		},
		{
			name:    "failed discard joins per-path errors",
			result:  &DiscardResponse{Success: false, Errors: []string{"a: denied", "b: denied"}},
			message: "a: denied; b: denied",
		},
		{
			name:    "successful push reports no failure",
			result:  &PushResponse{Success: true},
			success: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			success, message := fileMutationOutcome(tc.result)
			if success != tc.success {
				t.Fatalf("expected success=%v, got %v", tc.success, success)
			}
			if message != tc.message {
				t.Fatalf("expected message %q, got %q", tc.message, message)
			}
		})
	}
}

// A network transfer is bounded by payload size and link speed, so it cannot share the
// local write budget: a large push needs minutes, and being killed at 30s reports as a
// failed push with no work done.
func TestRemoteMutationTimeoutExceedsLocalBudget(t *testing.T) {
	t.Parallel()

	if remoteMutationTimeout <= localMutationTimeout {
		t.Fatalf("remote mutations must get a longer budget than local ones: remote=%s local=%s",
			remoteMutationTimeout, localMutationTimeout)
	}
	if remoteMutationTimeout < 10*time.Minute {
		t.Fatalf("remote budget %s is too short for a multi-hundred-megabyte push", remoteMutationTimeout)
	}
}

// SEAM: handlers that talk to a remote must run under the remote budget. Routing one
// back through runFileMutation silently caps it at the local write budget, which kills
// a large push mid-transfer.
func TestRemoteHandlersUseTheRemoteMutationBudget(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "repo_files_connect.go", nil, 0)
	if err != nil {
		t.Fatalf("parse repo_files_connect.go: %v", err)
	}

	remoteHandlers := map[string]bool{
		"pushToRemoteConnect":      false,
		"pullFromRemoteConnect":    false,
		"runUpstreamActionConnect": false,
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if _, tracked := remoteHandlers[fn.Name.Name]; !tracked {
			continue
		}
		ast.Inspect(fn, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch selector.Sel.Name {
			case "runRemoteMutation":
				remoteHandlers[fn.Name.Name] = true
			case "runFileMutation", "runMutation":
				t.Errorf("%s calls %s; a network transfer must not run under the local write budget",
					fn.Name.Name, selector.Sel.Name)
			}
			return true
		})
	}

	for name, usesRemoteBudget := range remoteHandlers {
		if !usesRemoteBudget {
			t.Errorf("%s does not run under the remote mutation budget", name)
		}
	}
}
