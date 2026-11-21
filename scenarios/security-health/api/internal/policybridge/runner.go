package policybridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

const (
	PublishEnabledEnv        = "VROOLI_AGENT_POLICY_PUBLISH_ENABLED"
	ReviewedEffectsDigestEnv = "VROOLI_AGENT_POLICY_REVIEWED_EFFECTS_SHA256"
	runnerName               = "vrooli-policy-runner"
)

// RunnerSink writes only after an isolated candidate canary has proved that
// the runner preserves every evaluator identity and that no effective action
// changes without the separate rollout review switch.
type RunnerSink struct{ Binary string }

func PublicationEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(PublishEnabledEnv)), "true")
}

func (s RunnerSink) PublishProviderSnapshot(ctx context.Context, data []byte) error {
	binary, err := s.findRunner()
	if err != nil {
		return err
	}
	if err := s.preflight(ctx, binary, data); err != nil {
		return fmt.Errorf("policy snapshot rollout preflight: %w", err)
	}
	file, cleanup, err := stagedSnapshot(data)
	if err != nil {
		return err
	}
	defer cleanup()
	return s.run(ctx, binary, "publish", "--snapshot-file", file)
}

func (s RunnerSink) findRunner() (string, error) {
	if strings.TrimSpace(s.Binary) != "" {
		return s.Binary, nil
	}
	if path, err := exec.LookPath(runnerName); err == nil {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve runtime home for %s: %w", runnerName, err)
	}
	runtimeHome, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return "", fmt.Errorf("resolve Vrooli runtime home for %s: %w", runnerName, err)
	}
	name := runnerName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(runtimeHome, "bin", name)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return "", fmt.Errorf("%s is not on PATH or at %s", runnerName, path)
	}
	return path, nil
}

func (s RunnerSink) preflight(ctx context.Context, binary string, data []byte) error {
	var candidate struct {
		Rules []struct {
			EvaluatorID string `json:"evaluator_id"`
			Risk        string `json:"risk"`
			Action      string `json:"action"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(data, &candidate); err != nil {
		return fmt.Errorf("decode candidate snapshot: %w", err)
	}
	if len(candidate.Rules) == 0 {
		return errors.New("candidate snapshot has no evaluator rules")
	}
	for _, rule := range candidate.Rules {
		if strings.TrimSpace(rule.EvaluatorID) == "" {
			return fmt.Errorf("candidate %s rule has no evaluator identity", rule.Risk)
		}
	}
	temp, err := os.MkdirTemp("", "security-health-policy-preflight-")
	if err != nil {
		return fmt.Errorf("create isolated policy store: %w", err)
	}
	defer os.RemoveAll(temp)
	file := filepath.Join(temp, "candidate.json")
	if err := os.WriteFile(file, data, 0o600); err != nil {
		return fmt.Errorf("write isolated candidate: %w", err)
	}
	if err := s.run(ctx, binary, "publish", "--snapshot-dir", temp, "--snapshot-file", file); err != nil {
		return fmt.Errorf("publish isolated candidate: %w", err)
	}
	for _, rule := range candidate.Rules {
		event, err := eventForRisk(rule.Risk)
		if err != nil {
			return err
		}
		staged, err := s.canary(ctx, binary, temp, event)
		if err != nil {
			return fmt.Errorf("canary %s: %w", rule.EvaluatorID, err)
		}
		if staged.EvaluatorID != rule.EvaluatorID || staged.Action != rule.Action {
			return fmt.Errorf("runner did not preserve %s: got evaluator_id=%q action=%q, want %q/%q", rule.Risk, staged.EvaluatorID, staged.Action, rule.EvaluatorID, rule.Action)
		}
		live, err := s.canary(ctx, binary, "", event)
		if err != nil {
			return fmt.Errorf("read live %s decision: %w", rule.Risk, err)
		}
		if live.Action != staged.Action {
			digest, digestErr := EffectsDigest(data)
			if digestErr != nil {
				return digestErr
			}
			if !strings.EqualFold(strings.TrimSpace(os.Getenv(ReviewedEffectsDigestEnv)), digest) {
				return fmt.Errorf("%s would change live action %s to %s; review this exact policy effect digest before rollout: %s", rule.Risk, live.Action, staged.Action, digest)
			}
		}
	}
	return nil
}

type canaryDecision struct {
	EvaluatorID string `json:"evaluator_id"`
	Action      string `json:"action"`
}

func (s RunnerSink) canary(ctx context.Context, binary, snapshotDir string, event map[string]any) (canaryDecision, error) {
	args := []string{"canary", "--runner", "codex"}
	if snapshotDir != "" {
		args = append(args, "--snapshot-dir", snapshotDir)
	}
	input, err := json.Marshal(event)
	if err != nil {
		return canaryDecision{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		return canaryDecision{}, fmt.Errorf("%s %s: %w", binary, strings.Join(args, " "), err)
	}
	var decision canaryDecision
	if err := json.Unmarshal(output, &decision); err != nil {
		return canaryDecision{}, fmt.Errorf("decode canary response: %w", err)
	}
	return decision, nil
}

// EffectsDigest excludes refresh timestamps but binds review to the provider,
// readiness/evidence state, capability maturity, scope, and exact evaluator
// rules that determine runtime actions.
func EffectsDigest(data []byte) (string, error) {
	var effects struct {
		ProviderID string `json:"provider_id"`
		Scope      any    `json:"scope"`
		Health     struct {
			State string `json:"state"`
		} `json:"health"`
		Readiness struct {
			State string `json:"state"`
		} `json:"readiness"`
		Evidence     string `json:"evidence_state"`
		Capabilities []struct {
			ID       string `json:"id"`
			Maturity string `json:"declared_maturity"`
		} `json:"capabilities"`
		Rules []struct {
			EvaluatorID string `json:"evaluator_id"`
			Risk        string `json:"risk"`
			Action      string `json:"action"`
			Reason      string `json:"reason"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(data, &effects); err != nil {
		return "", fmt.Errorf("decode policy effects: %w", err)
	}
	canonical, err := json.Marshal(effects)
	if err != nil {
		return "", fmt.Errorf("encode policy effects: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func (s RunnerSink) run(ctx context.Context, binary string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", binary, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func stagedSnapshot(data []byte) (string, func(), error) {
	file, err := os.CreateTemp("", "sda-agent-policy-*.json")
	if err != nil {
		return "", func() {}, fmt.Errorf("stage policy snapshot: %w", err)
	}
	path := file.Name()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", func() {}, fmt.Errorf("stage policy snapshot: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", func() {}, fmt.Errorf("stage policy snapshot: %w", err)
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func eventForRisk(risk string) (map[string]any, error) {
	event := map[string]any{"contract_version": "agent-policy/v1", "runner": "codex"}
	switch risk {
	case "dependency_addition":
		event["tool"], event["arguments"] = "pnpm", []string{"add", "policy-preflight-fixture"}
	case "dependency_upgrade":
		event["tool"], event["arguments"] = "pnpm", []string{"update", "policy-preflight-fixture"}
	case "dependency_removal":
		event["tool"], event["arguments"] = "pnpm", []string{"remove", "policy-preflight-fixture"}
	case "lifecycle_execution":
		event["tool"], event["arguments"] = "pnpm", []string{"run", "prepare"}
	case "unknown":
		event["tool"], event["arguments"] = "node", []string{"fixture.js"}
	default:
		return nil, fmt.Errorf("cannot preflight unsupported policy risk %q", risk)
	}
	return event, nil
}
