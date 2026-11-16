package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CandidateCheck is the durable view exposed to clients. A running row is only
// a job handle; it is never evidence that the candidate passed.
type CandidateCheck struct {
	JobID           string              `json:"job_id"`
	State           string              `json:"state"`
	CandidateDigest string              `json:"candidate_digest"`
	ConfigDigest    string              `json:"config_digest"`
	Result          *PrecommitRunResult `json:"result,omitempty"`
}

type candidateCheckRun struct {
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	repoPath string
	config   PrecommitConfig
	state    string
	started  bool
}

func candidateDigest(ctx context.Context, git GitRunner, repoDir string) (string, error) {
	if git == nil {
		return "", fmt.Errorf("git runner is required for candidate identity")
	}
	tree, err := git.WriteTree(ctx, repoDir)
	if err != nil {
		return "", fmt.Errorf("identify staged candidate: %w", err)
	}
	return "tree:" + tree, nil
}

func precommitConfigDigest(ctx context.Context, repoDir string, cfg PrecommitConfig) (string, error) {
	// Arbitrary configured shell commands are opaque. Bind every declared input
	// GCT controls, the command environment, and resolved shell/leading executable
	// bytes when the command has a simple executable prefix.
	commandTool := ""
	if fields := strings.Fields(cfg.Command); len(fields) > 0 && !strings.Contains(fields[0], "=") {
		commandTool = executableIdentity(fields[0])
	}
	inputDigest, err := hashPrecommitInputs(ctx, repoDir, cfg.DeclaredInputs)
	if err != nil {
		return "", err
	}
	value := strings.Join([]string{
		"gct-precommit-config-v1", cfg.Command,
		filepath.Clean(cfg.WorkingDirectory), fmt.Sprint(cfg.TimeoutSeconds),
		os.Getenv("PATH"), executableIdentity("bash"), commandTool, inputDigest,
	}, "\x00")
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:]), nil
}

func executableIdentity(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return "unresolved:" + name
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	file, err := os.Open(path)
	if err != nil {
		return "unreadable:" + filepath.Clean(path)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "unstatable:" + filepath.Clean(path)
	}
	if info.Size() > 64<<20 {
		return fmt.Sprintf("large:%s:%d:%d", filepath.Clean(path), info.Size(), info.ModTime().UnixNano())
	}
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "unreadable:" + filepath.Clean(path)
	}
	return filepath.Clean(path) + ":" + hex.EncodeToString(h.Sum(nil))
}

func candidateJobID(repoPath, candidate, config string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{repoPath, candidate, config}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func (s *PrecommitService) StartCandidateCheck(ctx context.Context, repoDir string, git GitRunner, retryTerminal bool) (CandidateCheck, error) {
	cfg, err := s.Get(ctx, repoDir)
	if err != nil {
		return CandidateCheck{}, err
	}
	if !cfg.Enabled || !cfg.RunBeforeCommit {
		return CandidateCheck{State: "disabled"}, nil
	}
	candidate, err := candidateDigest(ctx, git, repoDir)
	if err != nil {
		return CandidateCheck{}, err
	}
	config, err := precommitConfigDigest(ctx, repoDir, cfg)
	if err != nil {
		return CandidateCheck{}, err
	}
	jobID := candidateJobID(repoDir, candidate, config)
	if s == nil || s.db == nil {
		return CandidateCheck{}, fmt.Errorf("durable precommit store not configured")
	}
	row, err := s.loadCandidateCheck(ctx, jobID)
	if err != nil && err != sql.ErrNoRows {
		return CandidateCheck{}, err
	}
	if err == nil && (row.State == "passed" || row.State == "failed" || row.State == "timeout" || row.State == "incomplete" || row.State == "cancelled") {
		if row.State == "passed" || !retryTerminal {
			return row, nil
		}
	}
	s.candidateMu.Lock()
	if s.candidateRuns == nil {
		s.candidateRuns = make(map[string]*candidateCheckRun)
	}
	if run := s.candidateRuns[jobID]; run != nil {
		state := run.state
		s.candidateMu.Unlock()
		return CandidateCheck{JobID: jobID, State: state, CandidateDigest: candidate, ConfigDigest: config}, nil
	}
	// Running rows without a local worker came from an owner that exited. A
	// fresh Start safely reclaims the exact identity instead of trusting stale
	// running state as proof of completion.
	workerCtx, cancelWorker := context.WithCancel(context.WithoutCancel(ctx))
	state := "running"
	if s.candidateMaxWork <= 0 {
		s.candidateMaxWork = defaultCandidateWorkers
	}
	if s.candidateActive >= s.candidateMaxWork {
		state = "queued"
	}
	run := &candidateCheckRun{ctx: workerCtx, cancel: cancelWorker, done: make(chan struct{}), repoPath: repoDir, config: cfg, state: state}
	s.candidateRuns[jobID] = run
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO git_precommit_candidate_checks (job_id, repo_path, candidate_digest, config_digest, state, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_path, candidate_digest, config_digest) DO UPDATE SET
			state=excluded.state, owner_id=excluded.owner_id, result_json=NULL, updated_at=excluded.updated_at
	`, jobID, repoDir, candidate, config, state, jobID, stamp, stamp)
	if err != nil {
		delete(s.candidateRuns, jobID)
		cancelWorker()
		s.candidateMu.Unlock()
		return CandidateCheck{}, fmt.Errorf("persist candidate check: %w", err)
	}
	if state == "queued" {
		s.candidateQueue = append(s.candidateQueue, jobID)
	} else {
		s.candidateActive++
		run.started = true
		go s.runCandidateCheck(workerCtx, repoDir, cfg, jobID)
	}
	s.candidateMu.Unlock()
	return CandidateCheck{JobID: jobID, State: state, CandidateDigest: candidate, ConfigDigest: config}, nil
}

func (s *PrecommitService) GetCandidateCheck(ctx context.Context, repoPath, jobID string) (CandidateCheck, error) {
	if s == nil || s.db == nil || strings.TrimSpace(jobID) == "" {
		return CandidateCheck{}, fmt.Errorf("candidate check identity is required")
	}
	return s.loadCandidateCheckForRepo(ctx, repoPath, jobID)
}

func (s *PrecommitService) loadCandidateCheck(ctx context.Context, jobID string) (CandidateCheck, error) {
	return scanCandidateCheck(s.db.QueryRowContext(ctx, `
		SELECT job_id, state, candidate_digest, config_digest, result_json
		FROM git_precommit_candidate_checks WHERE job_id = ?
	`, jobID))
}

func (s *PrecommitService) loadCandidateCheckForRepo(ctx context.Context, repoPath, jobID string) (CandidateCheck, error) {
	return scanCandidateCheck(s.db.QueryRowContext(ctx, `
		SELECT job_id, state, candidate_digest, config_digest, result_json
		FROM git_precommit_candidate_checks WHERE job_id = ? AND repo_path = ?
	`, jobID, repoPath))
}

type candidateCheckScanner interface {
	Scan(dest ...any) error
}

func scanCandidateCheck(scanner candidateCheckScanner) (CandidateCheck, error) {
	var row CandidateCheck
	var encoded sql.NullString
	err := scanner.Scan(&row.JobID, &row.State, &row.CandidateDigest, &row.ConfigDigest, &encoded)
	if err != nil {
		return CandidateCheck{}, err
	}
	if encoded.Valid && encoded.String != "" {
		var result PrecommitRunResult
		if err := json.Unmarshal([]byte(encoded.String), &result); err != nil {
			return CandidateCheck{}, fmt.Errorf("decode candidate check result: %w", err)
		}
		row.Result = &result
	}
	return row, nil
}

func (s *PrecommitService) runCandidateCheck(ctx context.Context, repoDir string, cfg PrecommitConfig, jobID string) {
	result, err := s.runConfig(ctx, repoDir, cfg)
	state := result.Status
	if err != nil {
		state = "incomplete"
		result = PrecommitRunResult{Status: state, Summary: "Precommit result unavailable: " + err.Error(), Timestamp: time.Now().UTC()}
	}
	if ctx.Err() != nil {
		state = "cancelled"
		result = PrecommitRunResult{Status: state, Summary: "Precommit check was cancelled", Timestamp: time.Now().UTC()}
	}
	encoded, _ := json.Marshal(result)
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, _ = s.db.ExecContext(writeCtx, `
		UPDATE git_precommit_candidate_checks SET state=?, result_json=?, updated_at=? WHERE job_id=?
	`, state, string(encoded), time.Now().UTC().Format(time.RFC3339Nano), jobID)
	s.finishCandidateCheck(jobID)
}

func (s *PrecommitService) finishCandidateCheck(jobID string) {
	s.candidateMu.Lock()
	if run := s.candidateRuns[jobID]; run != nil {
		delete(s.candidateRuns, jobID)
		if run.started && s.candidateActive > 0 {
			s.candidateActive--
		}
		run.cancel()
		close(run.done)
	}
	s.startQueuedCandidateChecksLocked()
	s.candidateMu.Unlock()
}

func (s *PrecommitService) startQueuedCandidateChecksLocked() {
	for s.candidateActive < s.candidateMaxWork && len(s.candidateQueue) > 0 {
		jobID := s.candidateQueue[0]
		s.candidateQueue = s.candidateQueue[1:]
		run := s.candidateRuns[jobID]
		if run == nil || run.ctx.Err() != nil {
			continue
		}
		run.state = "running"
		run.started = true
		s.candidateActive++
		result, err := s.db.ExecContext(context.WithoutCancel(run.ctx), `UPDATE git_precommit_candidate_checks SET state='running', updated_at=? WHERE job_id=? AND state='queued'`, time.Now().UTC().Format(time.RFC3339Nano), jobID)
		if err == nil {
			if changed, rowsErr := result.RowsAffected(); rowsErr == nil && changed == 1 {
				go s.runCandidateCheck(run.ctx, run.repoPath, run.config, jobID)
				continue
			}
		}
		// Never launch work whose durable identity could not be promoted to
		// running. A later admission must see a terminal incomplete receipt.
		encoded := encodeCandidateCheckResult(PrecommitRunResult{Status: "incomplete", Summary: "Queued candidate could not be claimed by the check owner", Timestamp: time.Now().UTC()})
		_, _ = s.db.ExecContext(context.WithoutCancel(run.ctx), `UPDATE git_precommit_candidate_checks SET state='incomplete', result_json=?, updated_at=? WHERE job_id=? AND state='queued'`, encoded, time.Now().UTC().Format(time.RFC3339Nano), jobID)
		delete(s.candidateRuns, jobID)
		run.cancel()
		close(run.done)
		s.candidateActive--
	}
}

func (s *PrecommitService) CancelCandidateCheck(ctx context.Context, repoPath, jobID string) (CandidateCheck, error) {
	check, err := s.GetCandidateCheck(ctx, repoPath, jobID)
	if err != nil {
		return CandidateCheck{}, err
	}
	if candidateCheckTerminal(check.State) {
		return check, nil
	}
	s.candidateMu.Lock()
	run := s.candidateRuns[jobID]
	if run == nil {
		// A durable running row with no local owner is stale. It cannot be
		// cancelled remotely or treated as current evidence.
		result := encodeCandidateCheckResult(PrecommitRunResult{Status: "incomplete", Summary: "Candidate check owner is no longer available", Timestamp: time.Now().UTC()})
		_, _ = s.db.ExecContext(ctx, `UPDATE git_precommit_candidate_checks SET state='incomplete', result_json=?, updated_at=? WHERE job_id=? AND state IN ('running','queued')`, result, time.Now().UTC().Format(time.RFC3339Nano), jobID)
		s.candidateMu.Unlock()
		return s.GetCandidateCheck(ctx, repoPath, jobID)
	}
	if !run.started {
		s.removeQueuedCandidateCheckLocked(jobID)
		run.cancel()
		result := encodeCandidateCheckResult(PrecommitRunResult{Status: "cancelled", Summary: "Precommit check was cancelled before execution", Timestamp: time.Now().UTC()})
		_, _ = s.db.ExecContext(ctx, `UPDATE git_precommit_candidate_checks SET state='cancelled', result_json=?, updated_at=? WHERE job_id=? AND state='queued'`, result, time.Now().UTC().Format(time.RFC3339Nano), jobID)
		delete(s.candidateRuns, jobID)
		close(run.done)
		s.startQueuedCandidateChecksLocked()
		s.candidateMu.Unlock()
		return s.GetCandidateCheck(ctx, repoPath, jobID)
	}
	run.state = "cancelling"
	_, _ = s.db.ExecContext(ctx, `UPDATE git_precommit_candidate_checks SET state='cancelling', updated_at=? WHERE job_id=? AND state='running'`, time.Now().UTC().Format(time.RFC3339Nano), jobID)
	run.cancel()
	done := run.done
	s.candidateMu.Unlock()
	select {
	case <-ctx.Done():
		return CandidateCheck{}, ctx.Err()
	case <-done:
		return s.GetCandidateCheck(ctx, repoPath, jobID)
	}
}

func (s *PrecommitService) removeQueuedCandidateCheckLocked(jobID string) {
	for index, queued := range s.candidateQueue {
		if queued == jobID {
			s.candidateQueue = append(s.candidateQueue[:index], s.candidateQueue[index+1:]...)
			return
		}
	}
}

// ShutdownCandidateChecks is called before the owner closes its database. It
// cancels all running process groups, marks never-started queue entries, and
// waits for terminal receipts to be persisted while their routed DB generation
// is still available.
func (s *PrecommitService) ShutdownCandidateChecks(ctx context.Context) error {
	s.candidateMu.Lock()
	s.candidateQueue = nil
	active := make([]*candidateCheckRun, 0, len(s.candidateRuns))
	for jobID, run := range s.candidateRuns {
		if run.started {
			run.state = "cancelling"
			_, _ = s.db.ExecContext(ctx, `UPDATE git_precommit_candidate_checks SET state='cancelling', updated_at=? WHERE job_id=? AND state='running'`, time.Now().UTC().Format(time.RFC3339Nano), jobID)
			active = append(active, run)
		} else {
			run.cancel()
			result := encodeCandidateCheckResult(PrecommitRunResult{Status: "cancelled", Summary: "Owner is shutting down before this check started", Timestamp: time.Now().UTC()})
			_, _ = s.db.ExecContext(ctx, `UPDATE git_precommit_candidate_checks SET state='cancelled', result_json=?, updated_at=? WHERE job_id=? AND state='queued'`, result, time.Now().UTC().Format(time.RFC3339Nano), jobID)
			delete(s.candidateRuns, jobID)
			close(run.done)
		}
	}
	s.candidateMu.Unlock()
	for _, run := range active {
		run.cancel()
	}
	for _, run := range active {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-run.done:
		}
	}
	return nil
}

func candidateCheckTerminal(state string) bool {
	switch state {
	case "passed", "failed", "timeout", "incomplete", "cancelled":
		return true
	default:
		return false
	}
}

func encodeCandidateCheckResult(result PrecommitRunResult) string {
	encoded, _ := json.Marshal(result)
	return string(encoded)
}

func (s *PrecommitService) AwaitCandidateCheck(ctx context.Context, repoDir string, git GitRunner) (PrecommitRunResult, error) {
	for attempts := 0; attempts < 3; attempts++ {
		check, err := s.StartCandidateCheck(ctx, repoDir, git, false)
		if err != nil {
			return PrecommitRunResult{}, err
		}
		if check.State == "disabled" {
			return PrecommitRunResult{}, nil
		}
		for check.State == "running" || check.State == "queued" {
			select {
			case <-ctx.Done():
				return PrecommitRunResult{}, ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
			check, err = s.GetCandidateCheck(ctx, repoDir, check.JobID)
			if err != nil {
				return PrecommitRunResult{}, err
			}
		}
		if check.State == "failed" || check.State == "timeout" {
			if check.Result != nil {
				return *check.Result, nil
			}
		}
		if check.State != "passed" || check.Result == nil {
			return PrecommitRunResult{Status: "incomplete", Summary: "candidate check did not complete successfully", Timestamp: time.Now().UTC()}, nil
		}
		current, err := candidateDigest(ctx, git, repoDir)
		if err != nil {
			return PrecommitRunResult{}, err
		}
		if current == check.CandidateDigest {
			result := *check.Result
			result.receiptTree = strings.TrimPrefix(check.CandidateDigest, "tree:")
			result.receiptConfig = check.ConfigDigest
			result.receiptID = check.JobID
			return result, nil
		}
	}
	return PrecommitRunResult{Status: "incomplete", Summary: "staged candidate kept changing during precommit checks", Timestamp: time.Now().UTC()}, nil
}
