package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func replayContract(t *testing.T, mode string) Contract {
	t.Helper()
	policy := map[string]any{
		"version": 1,
		"effects": []map[string]any{{
			"binding_id": "fixture-files/write/create",
			"owner":      "fixture-files",
			"mode":       mode,
			"key_input":  "request_key",
		}},
	}
	if mode == "idempotent" {
		policy["effects"].([]map[string]any)[0]["idempotency_guarantee"] = "owner_deduplicates_same_key"
	}
	if mode == "owner_receipt" {
		entry := policy["effects"].([]map[string]any)[0]
		entry["receipt_binding"] = "fixture-files/receipt/get"
		entry["receipt_id_field"] = "receipt_id"
		entry["outcome_field"] = "effect_state"
	}
	declaration, err := json.Marshal(map[string]any{"replay_safety": policy})
	require.NoError(t, err)
	bindings := []BindingRef{{ID: "fixture-files/write/create", Effect: "write", Via: "binding"}}
	if mode == "owner_receipt" {
		bindings = append(bindings, BindingRef{ID: "fixture-files/receipt/get", Effect: "read", Via: "binding"})
	}
	return Contract{
		ID: "fixture-files.create", Scenario: "fixture-files", Name: "create", Declaration: declaration,
		Bindings: bindings,
		Inputs:   map[string]InputSpec{"request_key": {Type: "string", Required: true}},
	}
}

func TestExistingEffectfulContractsDefaultUnsafe(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := NewIndex()
	require.NoError(t, index.Load(repoRoot))

	var effectful []string
	for _, contract := range index.List() {
		if !ContractMayHaveEffects(contract) {
			continue
		}
		effectful = append(effectful, contract.ID)
		assessment := AssessReplaySafety(contract, nil, nil)
		require.Equal(t, ReplayUnsafe, assessment.Disposition, "%s must not be replayable without complete owner proof: %s", contract.ID, assessment.Reason)
	}
	require.NotEmpty(t, effectful, "fixture must cover current effectful declarations")
	t.Logf("fail-closed effectful contract set (%d): %v", len(effectful), effectful)
}

func TestReplaySafetyAcceptsOwnerDeclaredIdempotencyKey(t *testing.T) {
	contract := replayContract(t, "idempotent")
	got := AssessReplaySafety(contract, map[string]any{"request_key": "run-42"}, nil)
	require.Equal(t, ReplayIdempotent, got.Disposition)
	require.Equal(t, []string{"fixture-files/write/create"}, got.EffectIDs)
}

func TestReplaySafetyClassifiesExplicitReadOnlyBindings(t *testing.T) {
	declaration := []byte(`{"name":"fixture-read.read","version":"1"}`)
	contract := Contract{
		ID: "fixture-read.read", Scenario: "fixture-read", Declaration: declaration,
		Bindings: []BindingRef{{ID: "fixture-read/item/list", Effect: "read", Via: "binding"}},
	}
	got := AssessReplaySafety(contract, nil, nil)
	require.Equal(t, ReplayReadOnly, got.Disposition)
	require.False(t, ContractMayHaveEffects(contract))
}

func TestReplaySafetyRequiresCompleteVerifiedOwnerReceipt(t *testing.T) {
	contract := replayContract(t, "owner_receipt")
	inputs := map[string]any{"request_key": "run-42"}
	base := EffectReceipt{BindingID: "fixture-files/write/create", ReceiptBinding: "fixture-files/receipt/get", Owner: "fixture-files", EffectKey: "run-42", ReceiptID: "receipt-1", Outcome: "applied", Verified: true}

	got := AssessReplaySafety(contract, inputs, []EffectReceipt{base})
	require.Equal(t, ReplayWithReceipts, got.Disposition)
	require.Equal(t, map[string]string{base.BindingID: "applied"}, got.EffectOutcomes)
	base.Outcome = "not_applied"
	got = AssessReplaySafety(contract, inputs, []EffectReceipt{base})
	require.Equal(t, ReplayWithReceipts, got.Disposition)
	require.Equal(t, map[string]string{base.BindingID: "not_applied"}, got.EffectOutcomes)
	base.Outcome = "applied"

	for name, receipt := range map[string]EffectReceipt{
		"unverified":         {BindingID: base.BindingID, ReceiptBinding: base.ReceiptBinding, Owner: base.Owner, EffectKey: base.EffectKey, ReceiptID: base.ReceiptID, Outcome: base.Outcome},
		"unknown outcome":    {BindingID: base.BindingID, ReceiptBinding: base.ReceiptBinding, Owner: base.Owner, EffectKey: base.EffectKey, ReceiptID: base.ReceiptID, Outcome: "unknown", Verified: true},
		"wrong owner":        {BindingID: base.BindingID, ReceiptBinding: base.ReceiptBinding, Owner: "other", EffectKey: base.EffectKey, ReceiptID: base.ReceiptID, Outcome: base.Outcome, Verified: true},
		"wrong effect key":   {BindingID: base.BindingID, ReceiptBinding: base.ReceiptBinding, Owner: base.Owner, EffectKey: "other-key", ReceiptID: base.ReceiptID, Outcome: base.Outcome, Verified: true},
		"missing receipt id": {BindingID: base.BindingID, ReceiptBinding: base.ReceiptBinding, Owner: base.Owner, EffectKey: base.EffectKey, Outcome: base.Outcome, Verified: true},
	} {
		t.Run(name, func(t *testing.T) {
			got := AssessReplaySafety(contract, inputs, []EffectReceipt{receipt})
			require.Equal(t, ReplayUnsafe, got.Disposition)
		})
	}
	require.Equal(t, ReplayUnsafe, AssessReplaySafety(contract, inputs, nil).Disposition)
}

func TestReplaySafetyRejectsMissingPolicyCoverageAndUnknownEffects(t *testing.T) {
	contract := replayContract(t, "idempotent")
	inputs := map[string]any{"request_key": "run-42"}
	require.Equal(t, ReplayUnsafe, AssessReplaySafety(Contract{Bindings: contract.Bindings}, inputs, nil).Disposition)

	contract.Bindings = append(contract.Bindings, BindingRef{ID: "fixture-files/write/delete", Effect: "destructive", Via: "binding"})
	require.Equal(t, ReplayUnsafe, AssessReplaySafety(contract, inputs, nil).Disposition)

	contract = replayContract(t, "idempotent")
	contract.Verbs = []string{"agent.run"}
	require.Equal(t, ReplayUnsafe, AssessReplaySafety(contract, inputs, nil).Disposition)

	contract = replayContract(t, "unsafe")
	require.Equal(t, ReplayUnsafe, AssessReplaySafety(contract, inputs, nil).Disposition)

	contract = replayContract(t, "idempotent")
	var declaration map[string]any
	require.NoError(t, json.Unmarshal(contract.Declaration, &declaration))
	declaration["replay_safety"].(map[string]any)["unrecognized_claim"] = true
	encoded, err := json.Marshal(declaration)
	require.NoError(t, err)
	contract.Declaration = encoded
	require.Equal(t, ReplayUnsafe, AssessReplaySafety(contract, inputs, nil).Disposition)
}

func TestReplaySafetySchemaAllowsExplicitPolicyAndRejectsUnknownMode(t *testing.T) {
	data := validContractData(t)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	raw["replay_safety"] = map[string]any{
		"version": 1,
		"effects": []map[string]any{{"binding_id": "sample/files/create", "owner": "sample", "mode": "idempotent", "key_input": "request_key", "idempotency_guarantee": "owner_deduplicates_same_key"}},
	}
	encoded, err := json.Marshal(raw)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "demo.json")
	require.NoError(t, os.WriteFile(path, encoded, 0o600))
	valid := readContract("demo", path, testSchema(t))
	require.Empty(t, valid.ValidationError)

	raw["replay_safety"].(map[string]any)["effects"].([]map[string]any)[0]["mode"] = "guess"
	encoded, err = json.Marshal(raw)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, encoded, 0o600))
	invalid := readContract("demo", path, testSchema(t))
	require.NotEmpty(t, invalid.ValidationError)
}
