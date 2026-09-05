package scopecatalog

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestDerivedVerbsResolveToCLIInvocations proves that each run-eligible command
// produced by the authorization catalog is accepted by the owning CLI's real
// argument parser. CLIs run in parallel, while commands within one CLI run
// sequentially so stale-detection rebuilds never race each other.
func TestDerivedVerbsResolveToCLIInvocations(t *testing.T) {
	if testing.Short() {
		t.Skip("runtime CLI conformance is an integration-strength package check")
	}

	catalog, err := Build(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}

	commandsByCLI := make(map[string][]string)
	for _, scope := range catalog.Scopes {
		if !scope.RunEligible || strings.TrimSpace(scope.Command) == "" {
			continue
		}
		cli := scope.Scenario
		if cli == ProjectManifestIdentity {
			cli = "vrooli"
		}
		commandsByCLI[cli] = append(commandsByCLI[cli], scope.Command)
	}

	type unresolved struct {
		cli     string
		command string
		reason  string
	}
	var (
		mu              sync.Mutex
		missingBinaries []string
		failures        []unresolved
		wg              sync.WaitGroup
	)
	workers := make(chan struct{}, 8)
	for cli, commands := range commandsByCLI {
		cli, commands := cli, commands
		wg.Add(1)
		go func() {
			defer wg.Done()
			workers <- struct{}{}
			defer func() { <-workers }()
			binary, lookupErr := exec.LookPath(cli)
			if lookupErr != nil {
				mu.Lock()
				missingBinaries = append(missingBinaries, cli)
				mu.Unlock()
				return
			}
			for _, command := range commands {
				resolved, reason := invocationResolves(binary, command)
				if resolved {
					continue
				}
				mu.Lock()
				failures = append(failures, unresolved{cli: cli, command: command, reason: reason})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	sort.Strings(missingBinaries)
	if len(missingBinaries) > 0 {
		t.Logf("CLI binaries not installed; runtime resolution not attempted for their manifests: %s", strings.Join(missingBinaries, ", "))
	}
	sort.Slice(failures, func(i, j int) bool {
		return failures[i].cli+"\x00"+failures[i].command < failures[j].cli+"\x00"+failures[j].command
	})
	for _, failure := range failures {
		t.Errorf("derived invocation %q does not resolve in %s: %s", fmt.Sprintf("%s %s", failure.cli, strings.ReplaceAll(failure.command, "/", " ")), failure.cli, failure.reason)
	}
}

func invocationResolves(binary, command string) (bool, string) {
	parts := strings.Split(command, "/")
	output, runErr, contextErr := runCLIHelp(binary, append(parts, "--help")...)
	if runErr == nil || strings.Contains(output, "Usage:") {
		return true, ""
	}

	// Commands whose first positional is required can interpret a trailing
	// --help as that positional before their leaf parser runs. Prove those from
	// the owning parent's help surface instead.
	parentArgs := append(append([]string(nil), parts[:len(parts)-1]...), "--help")
	parentOutput, parentErr, _ := runCLIHelp(binary, parentArgs...)
	if (parentErr == nil || strings.Contains(parentOutput, "Usage:")) && helpNamesCommand(parentOutput, parts[len(parts)-1]) {
		return true, ""
	}

	reason := strings.TrimSpace(output)
	if errors.Is(contextErr, context.DeadlineExceeded) {
		reason = "timed out after 30s"
	} else if reason == "" {
		reason = runErr.Error()
	}
	return false, reason
}

func runCLIHelp(binary string, args ...string) (string, error, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, args...).CombinedOutput()
	return string(output), err, ctx.Err()
}

func helpNamesCommand(output, command string) bool {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == command {
			return true
		}
	}
	return false
}
