package teams

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompt-manager/internal/teamconfig"
)

func finiteCLIConfig() HeartbeatConfig {
	return HeartbeatConfig{TeamID: "new-team", AgentID: "leader", ProfileKey: "qualified", Schedule: "*/5 * * * *",
		FiniteLeader:      &teamconfig.FiniteLeader{EffortRef: "owner:any-effort", AcceptedRevision: "accepted:r3", CoordinatorPromptRef: "pm:coordinator", SourceRefs: []string{"owner:source"}},
		FiniteLeaderState: json.RawMessage(`{"id":"reservation","runId":"existing-run","status":"parked"}`)}
}

func finiteCLIRequest(t *testing.T, payload string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "binding.json")
	if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFiniteLeaderCLIProvisioningIsDisabledAndExact(t *testing.T) {
	for _, update := range []bool{false, true} {
		config := finiteCLIConfig()
		input, _ := json.Marshal(finiteLeaderBindingInput{Schedule: config.Schedule, ProfileKey: config.ProfileKey, FiniteLeader: config.FiniteLeader})
		ctx := &fakeContext{response: config}
		args := []string{"new-team", "leader", "--request-file", finiteCLIRequest(t, string(input)), "--json"}
		method := "POST"
		if update {
			args = append(args, "--update")
			method = "PUT"
		}
		if err := route(ctx, append([]string{"heartbeat-bind-effort"}, args...)); err != nil {
			t.Fatal(err)
		}
		ctx.assertMethodPath(t, method, "/teams/new-team/heartbeats/leader")
		var sent CreateHeartbeatRequest
		if err := json.Unmarshal(ctx.gotPayload, &sent); err != nil {
			t.Fatal(err)
		}
		if sent.Enabled == nil || *sent.Enabled || sent.FiniteLeader == nil || sent.FiniteLeader.AcceptedRevision != "accepted:r3" || sent.ProfileKey != "qualified" {
			t.Fatalf("provisioning changed identity or enabled work: %s", ctx.gotPayload)
		}
	}
}

func TestFiniteLeaderCLIRejectsAmbiguousInputBeforeMutation(t *testing.T) {
	for _, payload := range []string{
		`{"enabled":true}`, `{"finiteLeader":null}`, `{} {}`, strings.Repeat(" ", 16385),
		`{"schedule":"@hourly","profileKey":"qualified","finiteLeader":{"effortRef":"x","acceptedRevision":"","coordinatorPromptRef":"pm:x","sourceRefs":["source:x"]}}`,
	} {
		ctx := &fakeContext{}
		if err := cmdHeartbeatBindEffort(ctx, []string{"team", "leader", "--request-file", finiteCLIRequest(t, payload)}); err == nil {
			t.Fatalf("invalid binding accepted: %q", payload)
		}
		if ctx.gotMethod != "" {
			t.Fatal("invalid input reached API")
		}
	}
}

func TestFiniteLeaderCLIRequiresOwnerConfirmation(t *testing.T) {
	inputConfig := finiteCLIConfig()
	input, _ := json.Marshal(finiteLeaderBindingInput{Schedule: inputConfig.Schedule, ProfileKey: inputConfig.ProfileKey, FiniteLeader: inputConfig.FiniteLeader})
	file := finiteCLIRequest(t, string(input))
	for _, field := range []string{"missing", "enabled", "different-effort", "wrong-member"} {
		config := finiteCLIConfig()
		switch field {
		case "missing":
			config.FiniteLeader = nil
		case "enabled":
			config.Enabled = true
		case "different-effort":
			config.FiniteLeader.EffortRef = "wrong-effort"
		case "wrong-member":
			config.AgentID = "wrong-member"
		}
		if err := cmdHeartbeatBindEffort(&fakeContext{response: config}, []string{"new-team", "leader", "--request-file", file}); err == nil {
			t.Fatalf("owner response %s was reported as confirmed", field)
		}
	}
}

func TestFiniteLeaderCLIRetirementPreservesBinding(t *testing.T) {
	current, retired := finiteCLIConfig(), finiteCLIConfig()
	retired.FiniteLeader.Retired = true
	ctx := &fakeContext{getResponse: current, response: retired}
	if err := route(ctx, []string{"heartbeat-retire-effort", "new-team", "leader", "--json"}); err != nil {
		t.Fatal(err)
	}
	var sent UpdateHeartbeatRequest
	if err := json.Unmarshal(ctx.gotPayload, &sent); err != nil {
		t.Fatal(err)
	}
	if ctx.gotMethod != "PUT" || sent.FiniteLeader == nil || !sent.FiniteLeader.Retired || sent.FiniteLeader.EffortRef != current.FiniteLeader.EffortRef || sent.Enabled == nil || *sent.Enabled {
		t.Fatalf("retirement did not preserve exact disabled binding: %s", ctx.gotPayload)
	}
	ctx = &fakeContext{err: errors.New("owner unavailable")}
	if err := cmdHeartbeatRetireEffort(ctx, []string{"new-team", "leader"}); err == nil || ctx.gotMethod != "GET" {
		t.Fatal("owner read failure performed retirement")
	}
}

func TestFiniteLeaderCLIReadRetainsReservation(t *testing.T) {
	config := finiteCLIConfig()
	ctx := &fakeContext{getResponse: config}
	output, err := captureTeamStdout(t, func() error {
		return route(ctx, []string{"heartbeat", "new-team", "leader", "--json"})
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded HeartbeatConfig
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatal(err)
	}
	var reservation struct {
		RunID  string `json:"runId"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(decoded.FiniteLeaderState, &reservation); err != nil {
		t.Fatal(err)
	}
	if decoded.FiniteLeader == nil || reservation.RunID != "existing-run" || reservation.Status != "parked" {
		t.Fatal("CLI decoding dropped finite leader identity")
	}
	output, err = captureTeamStdout(t, func() error {
		return route(ctx, []string{"heartbeat", "new-team", "leader"})
	})
	if err != nil || !strings.Contains(output, "owner:any-effort") || !strings.Contains(output, "existing-run") || !strings.Contains(output, "parked") {
		t.Fatalf("human heartbeat read omitted owner standing: %s %v", output, err)
	}
}
