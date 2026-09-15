package mutation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"unit-health/internal/executor"
	"unit-health/internal/testquality"
)

const maxMutants = 50

type PilotRequest struct {
	WorkspaceRoot string
	Package       string
	Operators     []string
	MaxMutants    uint32
	Seed          string
	CacheRoot     string
	Executor      executor.Runner
}

type PilotResponse struct {
	RunID         string
	WorkspacePath string
	Receipts      []Receipt
	Summary       Summary
	Limitations   []string
}

type Receipt struct {
	ID          string
	Operator    string
	File        string
	Line        uint32
	Disposition testquality.MutationDisposition
	Detail      string
	OwningTest  string
}

type Summary struct {
	Generated             uint32
	Killed                uint32
	Survived              uint32
	Invalid               uint32
	Equivalent            uint32
	OutOfContract         uint32
	InfrastructureFailure uint32
	Unknown               uint32
	KillRate              float64
}

func Run(ctx context.Context, request PilotRequest) (PilotResponse, error) {
	if request.WorkspaceRoot == "" || request.CacheRoot == "" {
		return PilotResponse{}, fmt.Errorf("workspace and cache roots are required")
	}
	if request.Seed == "" {
		return PilotResponse{}, fmt.Errorf("seed is required")
	}
	if request.Package == "" {
		return PilotResponse{}, fmt.Errorf("package is required")
	}
	packagePath, err := safePackagePath(request.WorkspaceRoot, request.Package)
	if err != nil {
		return PilotResponse{}, err
	}
	if info, statErr := os.Stat(packagePath); statErr != nil || !info.IsDir() {
		return PilotResponse{}, fmt.Errorf("package %q is not a directory", request.Package)
	}
	operators := normalizeOperatorList(request.Operators)
	if len(operators) == 0 {
		operators = []string{OperatorBoundary, OperatorNegateCondition, OperatorReturnConstant}
	}
	candidates, err := Discover(packagePath, request.WorkspaceRoot, operators)
	if err != nil {
		return PilotResponse{}, err
	}
	candidates = CandidateOrder(request.Seed, candidates)
	limit := int(request.MaxMutants)
	if limit <= 0 {
		limit = 20
	}
	if limit > maxMutants {
		limit = maxMutants
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	if len(candidates) == 0 {
		return PilotResponse{}, fmt.Errorf("no mutation sites found for operators %v", operators)
	}

	runID := newRunID()
	runRoot := filepath.Join(request.CacheRoot, "mutation", runID)
	if err := os.MkdirAll(runRoot, 0o755); err != nil {
		return PilotResponse{}, fmt.Errorf("create mutation cache prefix: %w", err)
	}
	response := PilotResponse{RunID: runID, WorkspacePath: runRoot, Receipts: make([]Receipt, 0, len(candidates))}
	runner := request.Executor
	if runner == nil {
		runner = executor.Bounded{}
	}
	for index, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			response.Limitations = append(response.Limitations, "pilot context ended before all mutants were observed: "+err.Error())
			break
		}
		copyRoot := filepath.Join(runRoot, fmt.Sprintf("mutant-%03d", index+1))
		if err := copyWorkspace(request.WorkspaceRoot, copyRoot); err != nil {
			response.Receipts = append(response.Receipts, Receipt{ID: candidate.ID, Operator: candidate.Operator, File: candidate.File, Line: uint32(candidate.Line), Disposition: testquality.MutationInfrastructureFailure, Detail: "copy workspace: " + err.Error(), OwningTest: candidate.OwningTest})
			continue
		}
		receipt := Receipt{ID: candidate.ID, Operator: candidate.Operator, File: candidate.File, Line: uint32(candidate.Line), OwningTest: candidate.OwningTest}
		mutantFile := filepath.Join(copyRoot, filepath.FromSlash(candidate.File))
		if err := Apply(mutantFile, candidate); err != nil {
			receipt.Disposition = testquality.MutationInfrastructureFailure
			receipt.Detail = "apply mutation: " + err.Error()
			response.Receipts = append(response.Receipts, receipt)
			_ = os.RemoveAll(copyRoot)
			continue
		}
		args := []string{"test", "-mod=readonly", "-modfile=" + filepath.Join(request.WorkspaceRoot, "go.mod"), "-count=1"}
		if candidate.OwningTest != "" {
			args = append(args, "-run=^"+candidate.OwningTest+"$")
		}
		args = append(args, request.Package)
		result := runner.Run(ctx, executor.Command{WorkspaceID: "mutation", Name: candidate.ID, Executable: "go", Args: args, Dir: copyRoot, TimeoutSeconds: 60, NoOutputTimeout: 60 * time.Second, Env: map[string]string{"GOWORK": "off", "CI": "1"}})
		receipt.Disposition, receipt.Detail = classifyResult(result)
		response.Receipts = append(response.Receipts, receipt)
		if err := os.RemoveAll(copyRoot); err != nil {
			response.Limitations = append(response.Limitations, "remove mutant workspace failed: "+err.Error())
		}
	}
	response.Summary = summarize(response.Receipts)
	return response, nil
}

func safePackagePath(workspaceRoot, packageArg string) (string, error) {
	clean := filepath.Clean(packageArg)
	clean = strings.TrimPrefix(clean, "."+string(filepath.Separator))
	clean = strings.TrimSuffix(clean, string(filepath.Separator)+"...")
	clean = strings.TrimSuffix(clean, "...")
	if filepath.IsAbs(packageArg) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("package must stay inside workspace: %q", packageArg)
	}
	root, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, clean)
	resolved, err := filepath.Abs(path)
	if err != nil || (resolved != root && !strings.HasPrefix(resolved, root+string(filepath.Separator))) {
		return "", fmt.Errorf("package escapes workspace: %q", packageArg)
	}
	return resolved, nil
}

func normalizeOperatorList(operators []string) []string {
	seen := map[string]bool{}
	for _, operator := range operators {
		for normalized := range normalizeOperators([]string{operator}) {
			seen[normalized] = true
		}
	}
	out := make([]string, 0, len(seen))
	for operator := range seen {
		out = append(out, operator)
	}
	sort.Strings(out)
	return out
}

func newRunID() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return "mutation-" + hex.EncodeToString(bytes[:])
	}
	return fmt.Sprintf("mutation-%d", time.Now().UnixNano())
}

func copyWorkspace(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(destination, 0o755)
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == ".cache" || entry.Name() == "coverage" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(destination, relative), 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := entry.Type().Perm()
		if mode == 0 {
			mode = 0o644
		}
		return os.WriteFile(filepath.Join(destination, relative), data, mode)
	})
}

func classifyResult(result executor.Result) (testquality.MutationDisposition, string) {
	if result.Status == executor.StatusPassed {
		return testquality.MutationSurvived, "owning test passed"
	}
	if result.Status == executor.StatusTimeout || result.FailureClass == executor.ClassTimeoutHang || result.FailureClass == executor.ClassNoOutputStall || result.FailureClass == executor.ClassSystem || result.FailureClass == executor.ClassMissingDependency {
		return testquality.MutationInfrastructureFailure, strings.TrimSpace(result.FailureReason)
	}
	if result.Status == executor.StatusFailed && compileFailure(result.Stdout, result.Stderr) {
		return testquality.MutationInvalid, "mutated source did not compile"
	}
	if result.Status == executor.StatusFailed {
		return testquality.MutationKilled, "owning test failed"
	}
	return testquality.MutationUnknown, "executor returned no classified evidence"
}

func compileFailure(stdout, stderr string) bool {
	combined := strings.ToLower(stdout + "\n" + stderr)
	for _, marker := range []string{"build failed", "undefined:", "syntax error", "cannot use", "too many errors", "expected ';'", "declared and not used"} {
		if strings.Contains(combined, marker) {
			return true
		}
	}
	return false
}

func summarize(receipts []Receipt) Summary {
	var summary Summary
	summary.Generated = uint32(len(receipts))
	for _, receipt := range receipts {
		switch receipt.Disposition {
		case testquality.MutationKilled:
			summary.Killed++
		case testquality.MutationSurvived:
			summary.Survived++
		case testquality.MutationInvalid:
			summary.Invalid++
		case testquality.MutationEquivalent:
			summary.Equivalent++
		case testquality.MutationOutOfContract:
			summary.OutOfContract++
		case testquality.MutationInfrastructureFailure:
			summary.InfrastructureFailure++
		default:
			summary.Unknown++
		}
	}
	denominator := summary.Killed + summary.Survived
	if denominator > 0 {
		summary.KillRate = float64(summary.Killed) / float64(denominator)
	}
	return summary
}
