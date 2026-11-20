package commands

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// maxRequestBytes mirrors the daemon's 8 MiB body ceiling.
const maxRequestBytes = 8 << 20

func GatewayCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "gateway",
		Description: "Answer typed decision questions (the surface ai-gateway calls)",
		Subcommands: []cliapp.Command{
			{
				Name:        "decide",
				Description: "Answer a System One request ({state, questions}) read from stdin on the role's upstream",
				Usage:       "resource-ollaya gateway decide --role <id> --input-stdin [--timeout 90s] [--json]",
				Run:         h.Decide,
			},
			{
				Name:        "health",
				Description: "Probe the local daemon (GET /api/version)",
				Usage:       "resource-ollaya gateway health [--json]",
				Run:         h.Health,
			},
		},
	}
}

func (h *Handlers) Decide(args []string) error {
	fs := flag.NewFlagSet("gateway decide", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "Decision role")
	expectedModel := fs.String("expected-model", "", "Expected local model identity from rights admission")
	expectedDigest := fs.String("expected-artifact-digest", "", "Expected local manifest digest from rights admission")
	expectedRightsSHA := fs.String("expected-rights-decision-sha256", "", "Expected canonical rights decision digest")
	expectedRightsEvaluatedAt := fs.String("expected-rights-evaluated-at", "", "Expected canonical rights evaluation timestamp")
	expectedRightsReceiptSHA := fs.String("expected-rights-receipt-sha256", "", "Expected canonical persisted rights receipt digest")
	expectedPolicySHA := fs.String("expected-policy-file-sha256", "", "Expected exact resource model-policy file digest")
	inputStdin := fs.Bool("input-stdin", false, "Read the {state, questions} JSON body from stdin")
	timeout := fs.Duration("timeout", 90*time.Second, "Upstream deadline, model load included")
	jsonOutput := fs.Bool("json", false, "Emit the normalized result as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*role) == "" || !*inputStdin {
		return errors.New("gateway decide requires --role and --input-stdin")
	}
	raw, err := io.ReadAll(io.LimitReader(h.Stdin, maxRequestBytes+1))
	if err != nil {
		return fmt.Errorf("read decision request: %w", err)
	}
	if len(raw) > maxRequestBytes {
		return fmt.Errorf("decision request exceeds %d bytes", maxRequestBytes)
	}
	var request upstream.Request
	if err := json.Unmarshal(raw, &request); err != nil {
		return fmt.Errorf("decode decision request: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	p, err := h.policy()
	if err != nil {
		return err
	}
	resolution, err := p.Resolve(*role, h.GetEnv)
	if err != nil {
		return err
	}
	if resolution.Locality == policy.LocalityLocal {
		resolution, err = h.authorizeLocalDecision(ctx, p, *role, resolution, strings.TrimSpace(*expectedModel), strings.TrimSpace(*expectedDigest), strings.TrimSpace(*expectedRightsSHA), strings.TrimSpace(*expectedRightsEvaluatedAt), strings.TrimSpace(*expectedRightsReceiptSHA), strings.TrimSpace(*expectedPolicySHA))
		if err != nil {
			return h.fail(err)
		}
		if err := h.requireModelPlacement(ctx, p, resolution); err != nil {
			return h.fail(err)
		}
	}
	result, err := h.client().Decide(ctx, resolution, request)
	if err != nil {
		return h.fail(err)
	}
	if *jsonOutput {
		return writeJSON(h.Stdout, result)
	}
	fmt.Fprintf(h.Stdout, "upstream=%s model=%s latency_ms=%d\n%s\n", result.Upstream, result.Model, result.LatencyMS, string(result.Answers))
	return nil
}

func (h *Handlers) authorizeLocalDecision(ctx context.Context, initialPolicy policy.Policy, role string, initial policy.Resolution, expectedModel, expectedDigest, expectedRightsSHA, expectedRightsEvaluatedAt, expectedReceiptSHA, expectedPolicySHA string) (policy.Resolution, error) {
	refuse := func(reason string) (policy.Resolution, error) {
		return policy.Resolution{}, &upstream.ProviderError{Code: upstream.CodeUnconfigured, Message: reason}
	}
	if initial.Rights == nil {
		return refuse("local decision rights are unknown: the model catalog has no artifact rights evidence")
	}
	if expectedModel == "" || expectedDigest == "" || expectedRightsSHA == "" || expectedRightsEvaluatedAt == "" || expectedReceiptSHA == "" || expectedPolicySHA == "" {
		return refuse("local decision requires expected model, manifest digest, policy digest, and current rights receipt proof")
	}
	if expectedModel != initial.Model || initialPolicy.LoadedSHA256 != expectedPolicySHA {
		return refuse("local decision model differs from the rights-admitted model")
	}
	evaluatedAt, err := time.Parse(time.RFC3339Nano, expectedRightsEvaluatedAt)
	if err != nil {
		return refuse("local decision rights evaluation timestamp is invalid")
	}
	currentContext, decision, err := h.rightsEvaluator().EvaluateWithContext(ctx, initial.Rights, sharedpolicy.RightsUseLocalInference, evaluatedAt)
	if err != nil {
		return refuse("trusted local decision rights context is unavailable: " + err.Error())
	}
	if strings.TrimSpace(currentContext.EntityScope) == "" {
		return refuse("local decision rights are not currently allowed (unknown: trusted context has no entity scope)")
	}
	if decision.Status != sharedpolicy.RightsAllowed {
		return refuse(fmt.Sprintf("local decision rights are not currently allowed (%s: %s)", decision.Status, decision.Reason))
	}
	if decision.DecisionSHA256 != expectedRightsSHA {
		return refuse(fmt.Sprintf("local decision rights receipt does not match the current trusted context (expected %s, got %s)", expectedRightsSHA, decision.DecisionSHA256))
	}
	initialIdentity, initialDigest, err := h.localModelIdentity(ctx, initialPolicy, initial.Model)
	if err != nil {
		return refuse("local decision manifest identity is unavailable: " + err.Error())
	}
	expectation := sharedpolicy.RightsDecisionReceiptExpectation{
		Role: role, Provider: sharedpolicy.RuntimeProviderOllaya, Model: initial.Model,
		Use: sharedpolicy.RightsUseLocalInference, PolicySchemaVersion: initialPolicy.SchemaVersion,
		ArtifactRights: initial.Rights, RuntimeIdentity: initialIdentity, Context: currentContext,
	}
	receipt := sharedpolicy.NewRightsDecisionReceipt(expectation, evaluatedAt)
	if receipt.Decision.DecisionSHA256 != expectedRightsSHA {
		return refuse("local decision rights receipt decision does not match the admitted decision")
	}
	if receipt.Decision.EvaluatedAt != expectedRightsEvaluatedAt {
		return refuse("local decision rights receipt timestamp does not match the admitted decision")
	}
	if receipt.ReceiptSHA256 != expectedReceiptSHA {
		return refuse(fmt.Sprintf("local decision rights receipt does not match the admitted receipt (expected %s, got %s)", expectedReceiptSHA, receipt.ReceiptSHA256))
	}
	if initialDigest != expectedDigest {
		return refuse("local decision manifest digest does not match the admitted artifact")
	}
	if err := sharedpolicy.RequireAllowedRightsDecisionReceipt(receipt); err != nil ||
		!sharedpolicy.RuntimeArtifactIdentityMatches(initialIdentity, initial.Rights, sharedpolicy.RuntimeProviderOllaya, initial.Model, initialDigest) {
		return refuse("local decision manifest identity differs from the rights-admitted artifact")
	}

	// Re-read the owner policy, current context, and actual installed manifest
	// immediately before invoking the local provider. This is the resource-side
	// boundary for direct CLI calls and the final Gateway check.
	freshPolicy, err := h.policy()
	if err != nil {
		return refuse("reload local decision policy before execution: " + err.Error())
	}
	fresh, err := freshPolicy.Resolve(role, h.GetEnv)
	if err != nil || freshPolicy.LoadedSHA256 != expectedPolicySHA || fresh.Locality != policy.LocalityLocal || fresh.Model != initial.Model || fresh.Rights == nil {
		return refuse("local decision policy changed or lost artifact rights before execution")
	}
	freshContext, err := h.rightsContext(ctx)
	if err != nil {
		return refuse("trusted local decision rights context is unavailable before execution: " + err.Error())
	}
	if err := h.rightsEvaluator().RequireCurrentWithContext(ctx, decision, fresh.Rights, sharedpolicy.RightsUseLocalInference, freshContext); err != nil {
		return refuse("local decision rights receipt changed before execution: " + err.Error())
	}
	freshIdentity, freshDigest, err := h.localModelIdentity(ctx, freshPolicy, fresh.Model)
	if err != nil || freshDigest != expectedDigest || !reflect.DeepEqual(freshIdentity, initialIdentity) ||
		!sharedpolicy.RuntimeArtifactIdentityMatches(freshIdentity, fresh.Rights, sharedpolicy.RuntimeProviderOllaya, fresh.Model, freshDigest) {
		return refuse("local decision manifest identity changed before execution")
	}
	freshExpectation := sharedpolicy.RightsDecisionReceiptExpectation{
		Role: role, Provider: sharedpolicy.RuntimeProviderOllaya, Model: fresh.Model,
		Use: sharedpolicy.RightsUseLocalInference, PolicySchemaVersion: freshPolicy.SchemaVersion,
		ArtifactRights: fresh.Rights, RuntimeIdentity: freshIdentity, Context: freshContext,
	}
	if err := sharedpolicy.VerifyRightsDecisionReceipt(receipt, freshExpectation); err != nil {
		return refuse("local decision rights receipt changed before execution: " + err.Error())
	}
	if err := sharedpolicy.RequireAllowedRightsDecisionReceipt(receipt); err != nil {
		return refuse("local decision rights are not allowed: " + err.Error())
	}
	fresh.RuntimeIdentity = freshIdentity
	fresh.ObservedRuntimeDigest = freshDigest
	fresh.ExpectedArtifactDigest = freshDigest
	return fresh, nil
}

func (h *Handlers) Health(args []string) error {
	fs := flag.NewFlagSet("gateway health", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOutput := fs.Bool("json", false, "Emit {healthy, version}")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	version, err := h.client().Version(ctx)
	if *jsonOutput {
		payload := map[string]any{"healthy": err == nil, "version": version}
		if err != nil {
			payload["error"] = err.Error()
		}
		if encodeErr := writeJSON(h.Stdout, payload); encodeErr != nil {
			return encodeErr
		}
		return err
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(h.Stdout, "healthy (ollaya %s)\n", version)
	return err
}
