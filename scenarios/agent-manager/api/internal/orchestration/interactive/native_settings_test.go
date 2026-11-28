package interactive

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
)

type settingsCaptureSessions struct {
	*fakeSessions
	created *webconsole.CreateSessionParams
}

func (s *settingsCaptureSessions) CreateSession(_ context.Context, p webconsole.CreateSessionParams) (string, error) {
	s.created = &p
	return "", errors.New("fixture stops at create")
}

func TestNativeCodexLaunchDescriptorCarriesAdmittedControls(t *testing.T) {
	fs := &settingsCaptureSessions{fakeSessions: newFakeSessions("unused")}
	info := codexGoalLaunchInfo{fakeLaunchInfo{rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/fixture/codex"}}
	sub := NewSubstrate(fs, fakeResolver(info), WithHomeDir(t.TempDir()))
	p := LaunchParams{RunnerType: domain.RunnerTypeCodex, WorkingDir: "/fixture/work", RunDir: t.TempDir(), NativeObjective: "fixture goal", Model: "ignored-legacy-model", Effort: domain.EffortHigh, Config: &domain.RunConfig{Model: "gpt-6-luna", Effort: domain.EffortLow, ContextLimitTokens: 12000, NetworkAccess: domain.NetworkAccessNone}}
	_, err := sub.Launch(context.Background(), p)
	if err == nil || fs.created == nil {
		t.Fatalf("launch did not reach fake create: %v", err)
	}
	var got struct {
		runner.CodexLaunchSettings
		Environment map[string]string `json:"environment"`
	}
	if err := json.Unmarshal([]byte(fs.created.LaunchDescriptorJSON), &got); err != nil {
		t.Fatal(err)
	}
	if got.Model != "gpt-6-luna" || got.ReasoningEffort != "low" || got.ControlsVersion != 1 || got.ApprovalPolicy != "never" || got.ContextLimitTokens != 12000 || got.Sandbox != "workspace-write" || !got.NetworkAccessDisabled {
		t.Fatalf("admitted controls lost: %+v", got)
	}
	if fs.created.LaunchCommand != "" || fs.created.LaunchMode != "codex_app_server" || got.Environment["CODEX_HOME"] == "" {
		t.Fatalf("native create = %+v", fs.created)
	}
}

func TestNativeCodexLaunchRejectsProtectedBeforeCreate(t *testing.T) {
	fs := &settingsCaptureSessions{fakeSessions: newFakeSessions("unused")}
	info := codexGoalLaunchInfo{fakeLaunchInfo{rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/fixture/codex"}}
	sub := NewSubstrate(fs, fakeResolver(info), WithHomeDir(t.TempDir()))
	_, err := sub.Launch(context.Background(), LaunchParams{RunnerType: domain.RunnerTypeCodex, WorkingDir: "/fixture/work", RunDir: t.TempDir(), NativeObjective: "fixture goal", Config: &domain.RunConfig{NetworkAccess: domain.NetworkAccessFull, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected}}})
	if err == nil || fs.created != nil {
		t.Fatalf("protected host launch: create=%v err=%v", fs.created, err)
	}
}

func TestNativeCodexLaunchRejectsUnenforceableNetworkBeforeCreate(t *testing.T) {
	for _, network := range []domain.NetworkAccess{"", domain.NetworkAccessLocalhost} {
		t.Run(string(network), func(t *testing.T) {
			fs := &settingsCaptureSessions{fakeSessions: newFakeSessions("unused")}
			info := codexGoalLaunchInfo{fakeLaunchInfo{rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/fixture/codex"}}
			sub := NewSubstrate(fs, fakeResolver(info), WithHomeDir(t.TempDir()))
			_, err := sub.Launch(context.Background(), LaunchParams{RunnerType: domain.RunnerTypeCodex, WorkingDir: "/fixture/work", RunDir: t.TempDir(), NativeObjective: "fixture goal", Config: &domain.RunConfig{Model: "gpt-6-luna", NetworkAccess: network}})
			if err == nil || fs.created != nil {
				t.Fatalf("unenforceable network reached host create: %v err=%v", fs.created, err)
			}
		})
	}
}
