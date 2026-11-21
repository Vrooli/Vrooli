package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

const attachedIdentityProofMaxBytes = 4 << 10

func (o *ControlPlaneObserver) readAttachedBinding(ctx context.Context, request attachedBindingRequest) (cliutil.AgentSessionBindingProof, error) {
	if o.readBinding != nil {
		return o.readBinding(ctx, request)
	}
	return readAttachedSessionBindingProof(ctx, request)
}

// readAttachedSessionBindingProof asks the control-plane owner to resolve a
// live attached session when the legacy Agent Manager row has no persisted PID.
// The proof is only a locator for the subsequent executor-scope observation;
// it is never persisted or treated as liveness by itself.
func readAttachedSessionBindingProof(parent context.Context, request attachedBindingRequest) (cliutil.AgentSessionBindingProof, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	args := []string{
		"agent", "identity-binding", "--json",
		"--run-id", request.RunID,
		"--harness-session-id", request.HarnessSessionID,
		"--web-console-session-id", request.WebConsoleSessionID,
		"--harness-kind", request.HarnessKind,
	}
	cmd := exec.CommandContext(ctx, "vrooli", args...)
	var out limitedAttachedIdentityProof
	cmd.Stdout = &out
	cmd.WaitDelay = 100 * time.Millisecond
	if err := cmd.Run(); err != nil {
		return cliutil.AgentSessionBindingProof{}, errors.New("control-plane attached identity proof unavailable")
	}
	dec := json.NewDecoder(strings.NewReader(out.String()))
	dec.DisallowUnknownFields()
	var proof cliutil.AgentSessionBindingProof
	if err := dec.Decode(&proof); err != nil {
		return cliutil.AgentSessionBindingProof{}, errors.New("control-plane attached identity proof invalid")
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return cliutil.AgentSessionBindingProof{}, errors.New("control-plane attached identity proof invalid")
	}
	if !validAttachedBindingProof(proof, request) {
		return cliutil.AgentSessionBindingProof{}, errors.New("control-plane attached identity proof mismatched")
	}
	return proof, nil
}

type limitedAttachedIdentityProof struct{ strings.Builder }

func (b *limitedAttachedIdentityProof) Write(p []byte) (int, error) {
	if b.Len()+len(p) > attachedIdentityProofMaxBytes {
		return 0, fmt.Errorf("attached identity proof exceeds %d bytes", attachedIdentityProofMaxBytes)
	}
	return b.Builder.Write(p)
}

func validAttachedBindingProof(proof cliutil.AgentSessionBindingProof, request attachedBindingRequest) bool {
	if proof.ProofVersion != cliutil.AgentSessionBindingProofVersion ||
		proof.RunID != request.RunID || proof.HarnessSessionID != request.HarnessSessionID ||
		proof.WebConsoleSessionID != request.WebConsoleSessionID || proof.HarnessKind != request.HarnessKind ||
		proof.OwnerKind != "web-console-session" || strings.TrimSpace(proof.LeaseSessionID) == "" {
		return false
	}
	binding := proof.Binding
	if binding.PID <= 1 || binding.UID < 0 || len(binding.Cgroup) > 1024 ||
		!strings.HasPrefix(binding.Cgroup, "0::/user.slice/user-") ||
		!strings.Contains(binding.Cgroup, "/vrooli.slice/vrooli-agents.slice/") ||
		strings.ContainsAny(binding.Cgroup, "\r\n\t ") {
		return false
	}
	start, err := strconv.ParseUint(binding.Start, 10, 64)
	if err != nil || start == 0 {
		return false
	}
	boot, err := uuid.Parse(binding.Boot)
	return err == nil && boot != uuid.Nil
}
