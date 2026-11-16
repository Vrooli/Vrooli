//go:build linux

package integration

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"
)

const (
	treasuryCF1CrossBudgetEvidencePathEnv = "TREASURY_CF1_CROSS_BUDGET_IDENTITIES_PATH"
	cf1CrossBudgetEvidenceLimit           = 64 << 10
	cf1CrossBudgetIdentityCapMinor        = 1000
)

// cf1AgentCrossBudgetIdentity is the per ordinary-run join between Agent
// Manager identity, the Treasury budget wallet and its first settled spend.
// It contains public identifiers and receipt hashes only; session private keys
// are never serialized here.
type cf1AgentCrossBudgetIdentity struct {
	RunID                     string `json:"run_id"`
	ProfileID                 string `json:"profile_id"`
	BudgetID                  string `json:"budget_id"`
	MandateID                 string `json:"mandate_id"`
	WalletID                  string `json:"wallet_id"`
	WalletAddress             string `json:"wallet_address"`
	OwnAuthorizationID        string `json:"own_authorization_id"`
	OwnSettlementID           string `json:"own_settlement_id"`
	OwnTxHash                 string `json:"own_tx_hash"`
	OwnHeadroomMinor          int64  `json:"own_headroom_minor"`
	ForeignWalletProbeStatus  string `json:"foreign_wallet_probe_status"`
	ForeignMandateProbeStatus string `json:"foreign_mandate_probe_status"`
}

type cf1CrossBudgetIdentityEvidence struct {
	Agents []cf1AgentCrossBudgetIdentity `json:"agents"`
}

type cf1AgentWalletObservation struct {
	WalletID string
	Address  string
}

func parseCF1AgentWalletOutput(data []byte, budgetID, mandateID, runID string) (cf1AgentWalletObservation, error) {
	var envelope struct {
		Wallet struct {
			WalletID       string `json:"wallet_id"`
			WalletIDCamel  string `json:"walletId"`
			BudgetID       string `json:"budget_id"`
			BudgetIDCamel  string `json:"budgetId"`
			MandateID      string `json:"mandate_id"`
			MandateIDCamel string `json:"mandateId"`
			AgentRunID     string `json:"agent_run_id"`
			AgentRunCamel  string `json:"agentRunId"`
			Address        string `json:"address"`
		} `json:"wallet"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return cf1AgentWalletObservation{}, fmt.Errorf("decode Treasury agent wallet: %w", err)
	}
	walletID := firstCF1Nonempty(envelope.Wallet.WalletID, envelope.Wallet.WalletIDCamel)
	observedBudget := firstCF1Nonempty(envelope.Wallet.BudgetID, envelope.Wallet.BudgetIDCamel)
	observedMandate := firstCF1Nonempty(envelope.Wallet.MandateID, envelope.Wallet.MandateIDCamel)
	observedRun := firstCF1Nonempty(envelope.Wallet.AgentRunID, envelope.Wallet.AgentRunCamel)
	if walletID == "" || envelope.Wallet.Address == "" {
		return cf1AgentWalletObservation{}, errors.New("Treasury agent wallet omitted wallet ID or address")
	}
	if observedBudget != budgetID || observedMandate != mandateID || observedRun != runID {
		return cf1AgentWalletObservation{}, fmt.Errorf("Treasury wallet binding does not match run/budget/mandate: wallet=%+v", envelope.Wallet)
	}
	if !cf1HexIdentity(envelope.Wallet.Address, 20) {
		return cf1AgentWalletObservation{}, errors.New("Treasury agent wallet returned a malformed EVM address")
	}
	return cf1AgentWalletObservation{WalletID: walletID, Address: strings.ToLower(envelope.Wallet.Address)}, nil
}

func firstCF1Nonempty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func validateCF1AgentCrossBudgetIdentity(agent cf1AgentCrossBudgetIdentity) error {
	for name, value := range map[string]string{
		"run_id": agent.RunID, "profile_id": agent.ProfileID, "budget_id": agent.BudgetID,
		"mandate_id": agent.MandateID, "wallet_id": agent.WalletID,
		"wallet_address": agent.WalletAddress, "own_authorization_id": agent.OwnAuthorizationID,
		"own_settlement_id": agent.OwnSettlementID, "own_tx_hash": agent.OwnTxHash,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("CF1 cross-budget identity omitted %s", name)
		}
	}
	if agent.OwnHeadroomMinor <= 0 || agent.OwnHeadroomMinor >= cf1CrossBudgetIdentityCapMinor {
		return errors.New("CF1 own post-spend headroom must prove a positive in-cap debit below the full budget cap")
	}
	if !cf1HexIdentity(agent.WalletAddress, 20) || !cf1HexIdentity(agent.OwnTxHash, 32) {
		return errors.New("CF1 wallet address or transaction hash has an invalid EVM identity shape")
	}
	if agent.ForeignWalletProbeStatus != "permission_denied" || agent.ForeignMandateProbeStatus != "permission_denied" {
		return errors.New("CF1 foreign wallet and foreign mandate probes must both be permission_denied")
	}
	return nil
}

type cf1PairProbeMarker struct {
	RunID                     string `json:"run_id"`
	CounterpartRunID          string `json:"counterpart_run_id"`
	TargetBudgetID            string `json:"target_budget_id"`
	TargetMandateID           string `json:"target_mandate_id"`
	CounterpartMandateStatus  string `json:"counterpart_mandate_status"`
	CounterpartGeneration     int64  `json:"counterpart_session_generation"`
	ForeignWalletProbeStatus  string `json:"foreign_wallet_probe_status"`
	ForeignMandateProbeStatus string `json:"foreign_mandate_probe_status"`
}

type cf1PairProbeRelease struct {
	RunID                        string `json:"run_id"`
	CounterpartRunID             string `json:"counterpart_run_id"`
	BothMandatesLiveDuringProbes bool   `json:"both_mandates_live_during_probes"`
}

type cf1PairIdentityReady struct {
	RunID            string `json:"run_id"`
	CounterpartRunID string `json:"counterpart_run_id"`
	BudgetID         string `json:"budget_id"`
	MandateID        string `json:"mandate_id"`
	WalletID         string `json:"wallet_id"`
	WalletAddress    string `json:"wallet_address"`
}

type cf1PairIdentityRelease struct {
	RunID                          string `json:"run_id"`
	CounterpartRunID               string `json:"counterpart_run_id"`
	BothIdentityRowsRecorded       bool   `json:"both_identity_rows_recorded"`
	BothMandatesLiveAtIdentityGate bool   `json:"both_mandates_live_at_identity_gate"`
}

func cf1PairBarrierPath(dir, stage, runID string) string {
	return filepath.Join(dir, stage+"-"+runID+".json")
}

func writeCF1PairBarrierMarker(dir, stage, runID string, value any) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect CF1 paired barrier directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return errors.New("CF1 paired barrier directory must be a private real directory")
	}
	if stage == "" || runID == "" || strings.ContainsAny(stage+runID, "/\\") {
		return errors.New("CF1 paired barrier marker identity is invalid")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode CF1 paired barrier marker: %w", err)
	}
	if len(data) > 16<<10 {
		return errors.New("CF1 paired barrier marker exceeds its size limit")
	}
	path := cf1PairBarrierPath(dir, stage, runID)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if errors.Is(err, os.ErrExist) {
		priorFile, openErr := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
		if openErr != nil {
			return fmt.Errorf("open existing CF1 paired barrier marker: %w", openErr)
		}
		defer priorFile.Close()
		prior, readErr := io.ReadAll(io.LimitReader(priorFile, (16<<10)+1))
		if readErr != nil || len(prior) > 16<<10 || !reflect.DeepEqual(prior, data) {
			return errors.New("existing CF1 paired barrier marker conflicts with this run")
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("create CF1 paired barrier marker: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write CF1 paired barrier marker: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("sync CF1 paired barrier marker: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close CF1 paired barrier marker: %w", err)
	}
	return nil
}

func validateCF1PairProbeRelease(release cf1PairProbeRelease, runID, counterpartRunID string) error {
	if release.RunID != runID || release.CounterpartRunID != counterpartRunID || !release.BothMandatesLiveDuringProbes {
		return errors.New("paired probe release lacks current run IDs or live-mandate verification")
	}
	return nil
}

func validateCF1PairIdentityRelease(release cf1PairIdentityRelease, runID, counterpartRunID string) error {
	if release.RunID != runID || release.CounterpartRunID != counterpartRunID || !release.BothIdentityRowsRecorded || !release.BothMandatesLiveAtIdentityGate {
		return errors.New("paired identity release lacks two-row or live-mandate verification")
	}
	return nil
}

func validateCF1PairedIdentityEvidence(path string, own cf1AgentCrossBudgetIdentity, counterpart cf1TreasuryCounterpartIdentity) error {
	evidence, err := readCF1CrossBudgetIdentityEvidence(path)
	if err != nil {
		return err
	}
	if len(evidence.Agents) != 2 {
		return fmt.Errorf("paired evidence has %d rows; want exactly two", len(evidence.Agents))
	}
	var ownRow, counterpartRow *cf1AgentCrossBudgetIdentity
	for index := range evidence.Agents {
		switch evidence.Agents[index].RunID {
		case own.RunID:
			ownRow = &evidence.Agents[index]
		case counterpart.RunID:
			counterpartRow = &evidence.Agents[index]
		}
	}
	if ownRow == nil || counterpartRow == nil || !reflect.DeepEqual(*ownRow, own) {
		return errors.New("paired evidence does not contain the exact current-run identity row and counterpart")
	}
	if counterpartRow.BudgetID != counterpart.BudgetID || counterpartRow.MandateID != counterpart.MandateID || counterpartRow.WalletID != counterpart.WalletID || !strings.EqualFold(counterpartRow.WalletAddress, counterpart.WalletAddress) {
		return errors.New("paired evidence counterpart row does not match the attested wallet tuple")
	}
	if ownRow.RunID == counterpartRow.RunID || ownRow.BudgetID == counterpartRow.BudgetID || ownRow.MandateID == counterpartRow.MandateID || ownRow.WalletID == counterpartRow.WalletID || strings.EqualFold(ownRow.WalletAddress, counterpartRow.WalletAddress) {
		return errors.New("paired evidence reused the same run, budget, mandate, wallet, or address")
	}
	return nil
}

func cf1HexIdentity(value string, size int) bool {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "0x")
	if len(value) != size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// upsertCF1CrossBudgetIdentityEvidence atomically joins one ordinary run's
// evidence into the shared two-agent artifact. Replaying the exact same row is
// idempotent; reusing a run ID with changed identity or settlement facts fails.
func upsertCF1CrossBudgetIdentityEvidence(path string, agent cf1AgentCrossBudgetIdentity) error {
	if err := validateCF1AgentCrossBudgetIdentity(agent); err != nil {
		return err
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Base(path) == "." {
		return errors.New("CF1 cross-budget evidence path must be absolute and clean")
	}
	dir := filepath.Dir(path)
	dirInfo, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect CF1 cross-budget evidence directory: %w", err)
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 || dirInfo.Mode().Perm()&0o077 != 0 {
		return errors.New("CF1 cross-budget evidence directory must be a private real directory")
	}
	lockFD, err := syscall.Open(dir, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
	if err != nil {
		return fmt.Errorf("open CF1 evidence directory lock without following links: %w", err)
	}
	lock := os.NewFile(uintptr(lockFD), dir)
	if lock == nil {
		_ = syscall.Close(lockFD)
		return errors.New("open CF1 evidence directory lock returned an invalid handle")
	}
	defer lock.Close()
	var lockStat syscall.Stat_t
	if err := syscall.Fstat(lockFD, &lockStat); err != nil || lockStat.Mode&syscall.S_IFMT != syscall.S_IFDIR || lockStat.Mode&0o077 != 0 {
		return errors.New("CF1 evidence lock is not a private directory")
	}
	if err := syscall.Flock(lockFD, syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock CF1 evidence directory: %w", err)
	}
	defer syscall.Flock(lockFD, syscall.LOCK_UN)

	evidence, err := readCF1CrossBudgetIdentityEvidence(path)
	if err != nil {
		return err
	}
	for _, prior := range evidence.Agents {
		if prior.RunID != agent.RunID {
			continue
		}
		if !reflect.DeepEqual(prior, agent) {
			return fmt.Errorf("CF1 cross-budget evidence conflicts with prior row for run %s", agent.RunID)
		}
		return nil
	}
	if len(evidence.Agents) >= 2 {
		return errors.New("CF1 cross-budget evidence already contains two ordinary runs")
	}
	for _, prior := range evidence.Agents {
		if prior.ProfileID == agent.ProfileID || prior.BudgetID == agent.BudgetID || prior.MandateID == agent.MandateID || prior.WalletID == agent.WalletID || strings.EqualFold(prior.WalletAddress, agent.WalletAddress) || prior.OwnAuthorizationID == agent.OwnAuthorizationID || prior.OwnSettlementID == agent.OwnSettlementID || strings.EqualFold(prior.OwnTxHash, agent.OwnTxHash) {
			return errors.New("CF1 ordinary runs reused a profile, budget, mandate, wallet, address, authorization, settlement or transaction identity")
		}
	}
	evidence.Agents = append(evidence.Agents, agent)
	sort.Slice(evidence.Agents, func(i, j int) bool { return evidence.Agents[i].RunID < evidence.Agents[j].RunID })
	encoded, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return fmt.Errorf("encode CF1 cross-budget identity evidence: %w", err)
	}
	if len(encoded) > cf1CrossBudgetEvidenceLimit {
		return errors.New("CF1 cross-budget identity evidence exceeded its size limit")
	}
	tmp, err := os.CreateTemp(dir, ".cf1-cross-budget-identities-*.tmp")
	if err != nil {
		return fmt.Errorf("create atomic CF1 evidence file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set CF1 evidence mode: %w", err)
	}
	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write CF1 evidence: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync CF1 evidence: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close CF1 evidence: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("publish CF1 evidence atomically: %w", err)
	}
	dirFile, err := os.Open(dir)
	if err == nil {
		err = dirFile.Sync()
		_ = dirFile.Close()
	}
	if err != nil {
		return fmt.Errorf("sync CF1 evidence directory: %w", err)
	}
	return nil
}

func readCF1CrossBudgetIdentityEvidence(path string) (cf1CrossBudgetIdentityEvidence, error) {
	evidence := cf1CrossBudgetIdentityEvidence{Agents: []cf1AgentCrossBudgetIdentity{}}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		return evidence, nil
	}
	if err != nil {
		return evidence, fmt.Errorf("open prior CF1 evidence without following links: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return evidence, errors.New("open prior CF1 evidence returned an invalid handle")
	}
	defer file.Close()
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil || stat.Mode&syscall.S_IFMT != syscall.S_IFREG || stat.Mode&0o077 != 0 || stat.Size > cf1CrossBudgetEvidenceLimit {
		return evidence, errors.New("prior CF1 evidence must be a private bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, cf1CrossBudgetEvidenceLimit+1))
	if err != nil || len(data) > cf1CrossBudgetEvidenceLimit {
		return evidence, errors.New("read prior CF1 evidence failed or exceeded the size limit")
	}
	if err := json.Unmarshal(data, &evidence); err != nil {
		return evidence, fmt.Errorf("decode prior CF1 evidence: %w", err)
	}
	if len(evidence.Agents) > 2 {
		return evidence, errors.New("prior CF1 evidence has more than two ordinary runs")
	}
	for _, agent := range evidence.Agents {
		if err := validateCF1AgentCrossBudgetIdentity(agent); err != nil {
			return evidence, fmt.Errorf("prior CF1 agent row is invalid: %w", err)
		}
	}
	return evidence, nil
}

func cf1IdentityFixture(runID, profileID, budgetID, mandateID, walletID, address, authorizationID, settlementID, tx string) cf1AgentCrossBudgetIdentity {
	return cf1AgentCrossBudgetIdentity{
		RunID: runID, ProfileID: profileID, BudgetID: budgetID, MandateID: mandateID,
		WalletID: walletID, WalletAddress: address, OwnAuthorizationID: authorizationID,
		OwnSettlementID: settlementID, OwnTxHash: tx, OwnHeadroomMinor: 970,
		ForeignWalletProbeStatus: "permission_denied", ForeignMandateProbeStatus: "permission_denied",
	}
}

func TestCF1CrossBudgetIdentityEvidenceUpsertsDistinctOrdinaryRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-manager-cross-budget-identities.json")
	first := cf1IdentityFixture("run-b", "profile-b", "budget-b", "mandate-b", "wallet-b", "0x0000000000000000000000000000000000000002", "auth-b", "settle-b", "0x"+strings.Repeat("b", 64))
	second := cf1IdentityFixture("run-a", "profile-a", "budget-a", "mandate-a", "wallet-a", "0x0000000000000000000000000000000000000001", "auth-a", "settle-a", "0x"+strings.Repeat("a", 64))
	if err := upsertCF1CrossBudgetIdentityEvidence(path, first); err != nil {
		t.Fatal(err)
	}
	if err := upsertCF1CrossBudgetIdentityEvidence(path, first); err != nil {
		t.Fatalf("exact row replay should be idempotent: %v", err)
	}
	if err := upsertCF1CrossBudgetIdentityEvidence(path, second); err != nil {
		t.Fatal(err)
	}
	evidence, err := readCF1CrossBudgetIdentityEvidence(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Agents) != 2 || evidence.Agents[0].RunID != "run-a" || evidence.Agents[1].RunID != "run-b" {
		t.Fatalf("shared artifact must contain two stable run-ID-sorted rows: %+v", evidence.Agents)
	}
	changed := first
	changed.WalletAddress = "0x0000000000000000000000000000000000000003"
	if err := upsertCF1CrossBudgetIdentityEvidence(path, changed); err == nil {
		t.Fatal("changed duplicate run row must be rejected")
	}
	duplicateWallet := second
	duplicateWallet.WalletID = first.WalletID
	if err := upsertCF1CrossBudgetIdentityEvidence(path, duplicateWallet); err == nil {
		t.Fatal("two ordinary runs must not reuse a wallet identity")
	}
	noDebit := cf1IdentityFixture("run-no-debit", "profile-no-debit", "budget-no-debit", "mandate-no-debit", "wallet-no-debit", "0x0000000000000000000000000000000000000004", "auth-no-debit", "settle-no-debit", "0x"+strings.Repeat("c", 64))
	noDebit.OwnHeadroomMinor = cf1CrossBudgetIdentityCapMinor
	if err := upsertCF1CrossBudgetIdentityEvidence(filepath.Join(t.TempDir(), "no-debit.json"), noDebit); err == nil {
		t.Fatal("an identity row with full, unspent headroom was accepted as an in-cap spend")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("identity evidence file mode must be 0600: info=%v err=%v", info, err)
	}
}

func TestCF1CrossBudgetIdentityEvidenceConcurrentUpsertsDoNotLoseRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-manager-cross-budget-identities.json")
	rows := []cf1AgentCrossBudgetIdentity{
		cf1IdentityFixture("run-a", "profile-a", "budget-a", "mandate-a", "wallet-a", "0x0000000000000000000000000000000000000001", "auth-a", "settle-a", "0x"+strings.Repeat("a", 64)),
		cf1IdentityFixture("run-b", "profile-b", "budget-b", "mandate-b", "wallet-b", "0x0000000000000000000000000000000000000002", "auth-b", "settle-b", "0x"+strings.Repeat("b", 64)),
	}
	start := make(chan struct{})
	results := make(chan error, len(rows))
	for _, row := range rows {
		row := row
		go func() {
			<-start
			results <- upsertCF1CrossBudgetIdentityEvidence(path, row)
		}()
	}
	close(start)
	for range rows {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	evidence, err := readCF1CrossBudgetIdentityEvidence(path)
	if err != nil || len(evidence.Agents) != 2 {
		t.Fatalf("concurrent ordinary-run evidence upserts lost a row: rows=%+v err=%v", evidence.Agents, err)
	}
}

func TestCF1AgentWalletOutputRequiresExpectedOrdinaryRunBinding(t *testing.T) {
	valid := []byte(`{"wallet":{"budget_id":"budget-a","mandate_id":"mandate-a","wallet_id":"wallet-a","address":"0x0000000000000000000000000000000000000001","agent_run_id":"run-a"}}`)
	if _, err := parseCF1AgentWalletOutput(valid, "budget-a", "mandate-a", "run-a"); err != nil {
		t.Fatal(err)
	}
	foreign := []byte(`{"wallet":{"budget_id":"budget-b","mandate_id":"mandate-a","wallet_id":"wallet-a","address":"0x0000000000000000000000000000000000000001","agent_run_id":"run-a"}}`)
	if _, err := parseCF1AgentWalletOutput(foreign, "budget-a", "mandate-a", "run-a"); err == nil {
		t.Fatal("agent wallet response bound to a different budget was accepted")
	}
	omitted := []byte(`{"wallet":{"budget_id":"budget-a","mandate_id":"mandate-a","wallet_id":"wallet-a","address":"0x0000000000000000000000000000000000000001"}}`)
	if _, err := parseCF1AgentWalletOutput(omitted, "budget-a", "mandate-a", "run-a"); err == nil {
		t.Fatal("wallet response without a run binding was accepted")
	}
}

func TestCF1ForeignProbeIDsPreferCounterpartPairAndKeepIsolationFallback(t *testing.T) {
	config := cf1TreasuryRunConfig{
		ForeignBudgetID: "counterpart-budget", ForeignMandateID: "counterpart-mandate",
		IsolationBudgetID: "isolation-budget", IsolationMandateID: "isolation-mandate",
	}
	budgetID, mandateID, err := config.foreignProbeIDs()
	if err != nil {
		t.Fatal(err)
	}
	if budgetID != "counterpart-budget" || mandateID != "counterpart-mandate" {
		t.Fatalf("counterpart IDs not selected: budget=%q mandate=%q", budgetID, mandateID)
	}
	config.BudgetID, config.MandateID = "counterpart-budget", "counterpart-mandate"
	if _, _, err := config.foreignProbeIDs(); err == nil {
		t.Fatal("counterpart IDs matching the current run should fail closed")
	}
	config.BudgetID, config.MandateID = "current-budget", "current-mandate"
	config.ForeignBudgetID, config.ForeignMandateID = "", ""
	budgetID, mandateID, err = config.foreignProbeIDs()
	if err != nil {
		t.Fatal(err)
	}
	if budgetID != "isolation-budget" || mandateID != "isolation-mandate" {
		t.Fatalf("isolation fallback not selected: budget=%q mandate=%q", budgetID, mandateID)
	}
	config.ForeignBudgetID = "partial-counterpart"
	if _, _, err := config.foreignProbeIDs(); err == nil {
		t.Fatal("partial counterpart configuration should fail closed")
	}
}

func TestCF1TreasuryCounterpartIdentityRequiresActiveRunBoundTuple(t *testing.T) {
	barrierDir := t.TempDir()
	if err := os.Chmod(barrierDir, 0o700); err != nil {
		t.Fatal(err)
	}
	evidenceDir := t.TempDir()
	if err := os.Chmod(evidenceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(evidenceDir, "agent-manager-cross-budget-identities.json")
	t.Setenv(treasuryCF1CrossBudgetEvidencePathEnv, evidencePath)
	config := cf1TreasuryRunConfig{
		BudgetID: "budget-own", MandateID: "mandate-own", ForeignBudgetID: "budget-other", ForeignMandateID: "mandate-other",
		PairedProbeBarrierDir: barrierDir,
		CounterpartIdentity: &cf1TreasuryCounterpartIdentity{
			RunID: "run-other", BudgetID: "budget-other", MandateID: "mandate-other", WalletID: "wallet-other",
			WalletAddress: "0x0000000000000000000000000000000000000002", WalletRunID: "run-other",
			WalletBudgetID: "budget-other", WalletMandateID: "mandate-other", MandateStatus: "LIVE", SessionGeneration: 1,
		},
	}
	if err := validateCF1TreasuryCounterpartIdentity(config, "run-own"); err != nil {
		t.Fatalf("complete active counterpart tuple was refused: %v", err)
	}
	cases := map[string]func(*cf1TreasuryRunConfig){
		"revoked mandate":             func(value *cf1TreasuryRunConfig) { value.CounterpartIdentity.MandateStatus = "REVOKED" },
		"stale session":               func(value *cf1TreasuryRunConfig) { value.CounterpartIdentity.SessionGeneration = 0 },
		"wallet from another run":     func(value *cf1TreasuryRunConfig) { value.CounterpartIdentity.WalletRunID = "run-stale" },
		"wallet from another budget":  func(value *cf1TreasuryRunConfig) { value.CounterpartIdentity.WalletBudgetID = "budget-stale" },
		"wallet from another mandate": func(value *cf1TreasuryRunConfig) { value.CounterpartIdentity.WalletMandateID = "mandate-stale" },
		"mismatched probe target":     func(value *cf1TreasuryRunConfig) { value.ForeignBudgetID = "budget-stale" },
		"self run": func(value *cf1TreasuryRunConfig) {
			value.CounterpartIdentity.RunID = "run-own"
			value.CounterpartIdentity.WalletRunID = "run-own"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := config
			counterpart := *config.CounterpartIdentity
			changed.CounterpartIdentity = &counterpart
			mutate(&changed)
			if err := validateCF1TreasuryCounterpartIdentity(changed, "run-own"); err == nil {
				t.Fatal("stale, revoked, self, or mismatched counterpart identity was accepted")
			}
		})
	}
	t.Setenv(treasuryCF1CrossBudgetEvidencePathEnv, "")
	if err := validateCF1TreasuryCounterpartIdentity(config, "run-own"); err == nil {
		t.Fatal("paired counterpart was accepted without the shared identity evidence path")
	}
}

func TestCF1PairedBarrierMarkersAndIdentityEvidenceRequireBothActiveRows(t *testing.T) {
	barrierDir := t.TempDir()
	if err := os.Chmod(barrierDir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := cf1PairProbeMarker{
		RunID: "run-a", CounterpartRunID: "run-b", TargetBudgetID: "budget-b", TargetMandateID: "mandate-b",
		CounterpartMandateStatus: "LIVE", CounterpartGeneration: 1,
		ForeignWalletProbeStatus: "permission_denied", ForeignMandateProbeStatus: "permission_denied",
	}
	if err := writeCF1PairBarrierMarker(barrierDir, "probe-complete", marker.RunID, marker); err != nil {
		t.Fatal(err)
	}
	if err := writeCF1PairBarrierMarker(barrierDir, "probe-complete", marker.RunID, marker); err != nil {
		t.Fatalf("exact marker replay should be idempotent: %v", err)
	}
	changedMarker := marker
	changedMarker.CounterpartGeneration++
	if err := writeCF1PairBarrierMarker(barrierDir, "probe-complete", marker.RunID, changedMarker); err == nil {
		t.Fatal("conflicting probe marker replay was accepted")
	}
	if err := validateCF1PairProbeRelease(cf1PairProbeRelease{RunID: "run-a", CounterpartRunID: "run-b", BothMandatesLiveDuringProbes: true}, "run-a", "run-b"); err != nil {
		t.Fatal(err)
	}
	if err := validateCF1PairProbeRelease(cf1PairProbeRelease{RunID: "run-a", CounterpartRunID: "run-b"}, "run-a", "run-b"); err == nil {
		t.Fatal("probe release without both live mandates was accepted")
	}
	if err := validateCF1PairIdentityRelease(cf1PairIdentityRelease{RunID: "run-a", CounterpartRunID: "run-b", BothIdentityRowsRecorded: true, BothMandatesLiveAtIdentityGate: true}, "run-a", "run-b"); err != nil {
		t.Fatal(err)
	}
	if err := validateCF1PairIdentityRelease(cf1PairIdentityRelease{RunID: "run-a", CounterpartRunID: "run-b", BothIdentityRowsRecorded: true}, "run-a", "run-b"); err == nil {
		t.Fatal("identity release without both active mandates was accepted")
	}

	evidenceDir := t.TempDir()
	if err := os.Chmod(evidenceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(evidenceDir, "agent-manager-cross-budget-identities.json")
	own := cf1IdentityFixture("run-a", "profile-a", "budget-a", "mandate-a", "wallet-a", "0x0000000000000000000000000000000000000001", "auth-a", "settle-a", "0x"+strings.Repeat("a", 64))
	counterpartRow := cf1IdentityFixture("run-b", "profile-b", "budget-b", "mandate-b", "wallet-b", "0x0000000000000000000000000000000000000002", "auth-b", "settle-b", "0x"+strings.Repeat("b", 64))
	if err := upsertCF1CrossBudgetIdentityEvidence(evidencePath, own); err != nil {
		t.Fatal(err)
	}
	if err := upsertCF1CrossBudgetIdentityEvidence(evidencePath, counterpartRow); err != nil {
		t.Fatal(err)
	}
	counterpart := cf1TreasuryCounterpartIdentity{
		RunID: "run-b", BudgetID: "budget-b", MandateID: "mandate-b", WalletID: "wallet-b", WalletAddress: "0x0000000000000000000000000000000000000002",
	}
	if err := validateCF1PairedIdentityEvidence(evidencePath, own, counterpart); err != nil {
		t.Fatalf("complete two-row artifact was refused: %v", err)
	}
	if err := validateCF1PairedIdentityEvidence(filepath.Join(t.TempDir(), "missing.json"), own, counterpart); err == nil {
		t.Fatal("paired release was allowed without both shared identity rows")
	}
}

func TestCF1CrossBudgetEvidenceRejectsSymlinkAndOversizedPriorFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-manager-cross-budget-identities.json")
	fixture := cf1IdentityFixture("run-a", "profile-a", "budget-a", "mandate-a", "wallet-a", "0x0000000000000000000000000000000000000001", "auth-a", "settle-a", "0x"+strings.Repeat("a", 64))
	outside := filepath.Join(dir, "outside.json")
	if err := os.WriteFile(outside, []byte(`{"agents":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if err := upsertCF1CrossBudgetIdentityEvidence(path, fixture); err == nil {
		t.Fatal("symlinked prior identity evidence was accepted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", cf1CrossBudgetEvidenceLimit+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := upsertCF1CrossBudgetIdentityEvidence(path, fixture); err == nil {
		t.Fatal("oversized prior identity evidence was accepted")
	}
}
