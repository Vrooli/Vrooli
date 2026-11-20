//go:build linux

package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	treasuryCF1MeasuredX402Env = "TREASURY_CF1_X402_MEASURED_SECOND_CALL"
	treasuryPaidBodyDirDevEnv  = "TREASURY_X402_PAID_BODY_EVIDENCE_DIR_DEV"
	treasuryPaidBodyDirInoEnv  = "TREASURY_X402_PAID_BODY_EVIDENCE_DIR_INO"
)

func hashCF1X402ReceiptReference(reference string) (string, int, error) {
	if reference == "" {
		return "", 0, errors.New("Treasury x402 receipt reference is empty")
	}
	digest := sha256.Sum256([]byte(reference))
	return hex.EncodeToString(digest[:]), len(reference), nil
}

func compareCF1X402ReceiptReferences(before, after string) (string, int, error) {
	beforeDigest, beforeLength, err := hashCF1X402ReceiptReference(before)
	if err != nil {
		return "", 0, err
	}
	afterDigest, afterLength, err := hashCF1X402ReceiptReference(after)
	if err != nil {
		return "", 0, err
	}
	if beforeLength != afterLength || beforeDigest != afterDigest || before != after {
		return "", 0, errors.New("Treasury x402 receipt reference changed across restart/retry")
	}
	return afterDigest, afterLength, nil
}

func treasuryCF1MeasuredX402Enabled() bool {
	return strings.TrimSpace(os.Getenv(treasuryCF1MeasuredX402Env)) == "1"
}

func cf1EvidenceDirectoryIdentity(path string) (uint64, uint64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, 0, err
	}
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return 0, 0, errors.New("trusted x402 evidence directory must be a private real directory")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Ino == 0 {
		return 0, 0, errors.New("trusted x402 evidence directory has no stable device/inode identity")
	}
	return uint64(stat.Dev), stat.Ino, nil
}

func setCF1EvidenceDirectoryIdentity(t *testing.T, evidenceDir string) {
	t.Helper()
	dev, ino, err := cf1EvidenceDirectoryIdentity(evidenceDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(treasuryPaidBodyDirDevEnv, strconv.FormatUint(dev, 10))
	t.Setenv(treasuryPaidBodyDirInoEnv, strconv.FormatUint(ino, 10))
}

func executeCF1MeasuredX402Call(
	ctx context.Context,
	request runner.ExecuteRequest,
	config cf1TreasuryRunConfig,
	coord string,
	runCLI func(string, ...string) ([]byte, error),
	write func(string, any) error,
) error {
	if !treasuryCF1MeasuredX402Enabled() {
		return nil
	}
	if config.X402CatalogURL == "" || config.X402AmountMinor != 1 || config.X402AuthorizationID == "" || config.X402SettlementID == "" || config.X402IdempotencyKey == "" || config.X402InstrumentID == "" {
		return errors.New("measured CF1 x402 call requires the existing one-minor-unit priced call configuration")
	}
	evidenceDir := strings.TrimSpace(os.Getenv("TREASURY_X402_PAID_BODY_EVIDENCE_DIR"))
	evidenceRoot := strings.TrimSpace(os.Getenv("TREASURY_X402_PAID_BODY_EVIDENCE_ROOT"))
	evidenceDirDev, devErr := strconv.ParseUint(strings.TrimSpace(os.Getenv(treasuryPaidBodyDirDevEnv)), 10, 64)
	evidenceDirIno, inoErr := strconv.ParseUint(strings.TrimSpace(os.Getenv(treasuryPaidBodyDirInoEnv)), 10, 64)
	if !filepath.IsAbs(evidenceDir) || filepath.Clean(evidenceDir) != evidenceDir || !filepath.IsAbs(evidenceRoot) || filepath.Clean(evidenceRoot) != evidenceRoot {
		return errors.New("measured CF1 x402 call requires absolute, clean paid-response evidence directory and run root")
	}
	if devErr != nil || inoErr != nil || evidenceDirDev == 0 || evidenceDirIno == 0 {
		return errors.New("measured CF1 x402 call requires trusted paid-response evidence directory device and inode")
	}

	settlementID := config.X402SettlementID + "-measured"
	authorizationID := config.X402AuthorizationID + "-measured"
	idempotencyKey := config.X402IdempotencyKey + "-measured"
	settlementIdempotencyKey := idempotencyKey + "-settlement"
	target, settlementTarget := (cliutil.TreasurySpendIntent{
		AuthorizationID: authorizationID, MandateID: config.MandateID, AmountMinor: config.X402AmountMinor, Currency: "USD",
		Counterparty: config.X402Counterparty, AuthorizationIdempotencyKey: idempotencyKey,
		SettlementID: settlementID, InstrumentID: config.X402InstrumentID,
		SettlementIdempotencyKey: settlementIdempotencyKey,
	}).Targets()
	if err := write("x402-measured-operation-grant-requested.json", map[string]any{
		"run_id": request.RunID.String(), "authorization_id": authorizationID, "settlement_id": settlementID,
		"amount_minor": config.X402AmountMinor, "price_id": config.X402PriceID, "call_id": config.X402CallID,
		"catalog_url": config.X402CatalogURL, "instrument_id": config.X402InstrumentID,
		"required_target": target, "required_settlement_target": settlementTarget,
	}); err != nil {
		return err
	}
	var approval struct {
		RunID        string   `json:"run_id"`
		GrantID      string   `json:"grant_id"`
		SpendTargets []string `json:"spend_targets"`
	}
	if err := waitForFile(ctx, filepath.Join(coord, "x402-measured-operation-approved.json"), &approval); err != nil {
		return err
	}
	if approval.RunID != request.RunID.String() || approval.GrantID == "" || len(approval.SpendTargets) != 2 || approval.SpendTargets[0] != target || approval.SpendTargets[1] != settlementTarget {
		return errors.New("measured CF1 x402 call is missing its own exact run-bound Agent Manager grant")
	}
	if _, err := runCLI("authorization-propose-x402-measured", "--api-base", config.APIBaseURL, "authorization", "propose",
		"--id", authorizationID, "--idempotency-key", idempotencyKey,
		"--mandate-id", config.MandateID, "--amount-minor", "1", "--currency", "USD",
		"--counterparty", config.X402Counterparty, "--settlement-id", settlementID,
		"--instrument-id", config.X402InstrumentID, "--settlement-idempotency-key", settlementIdempotencyKey, "--json"); err != nil {
		return err
	}
	if err := write("x402-measured-authorization-ready.json", map[string]any{
		"run_id": request.RunID.String(), "authorization_id": authorizationID, "settlement_id": settlementID,
		"grant_id": approval.GrantID, "price_id": config.X402PriceID, "call_id": config.X402CallID,
	}); err != nil {
		return err
	}
	if err := waitForFile(ctx, filepath.Join(coord, "x402-measured-call-release"), nil); err != nil {
		return err
	}
	settledOutput, err := runCLI("settlement-report-x402-measured", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
		"--authorization-id", authorizationID, "--settlement-id", settlementID,
		"--instrument-id", config.X402InstrumentID, "--idempotency-key", settlementIdempotencyKey, "--json")
	if err != nil {
		return fmt.Errorf("no-drop measured x402 settlement did not settle: %w", err)
	}
	var response struct {
		Settlement struct {
			ID              string `json:"id"`
			Outcome         string `json:"outcome"`
			ExternalID      string `json:"external_id"`
			AmountMinor     int64  `json:"amount_minor"`
			Detail          string `json:"detail"`
			AuthorizationID string `json:"authorization_id"`
		} `json:"settlement"`
	}
	if err := json.Unmarshal(settledOutput, &response); err != nil {
		return fmt.Errorf("decode measured x402 settlement response: %w", err)
	}
	if response.Settlement.Outcome != "CHARGE_OUTCOME_SETTLED" || response.Settlement.ID != settlementID || response.Settlement.AuthorizationID != authorizationID || response.Settlement.AmountMinor != 1 || strings.TrimSpace(response.Settlement.ExternalID) == "" {
		return fmt.Errorf("measured x402 call returned an unexpected settlement: %+v", response.Settlement)
	}
	bodyName := "x402-paid-response-" + settlementID + ".body"
	body, err := readCF1PaidResponseBody(evidenceRoot, evidenceDir, bodyName, evidenceDirDev, evidenceDirIno)
	if err != nil {
		return err
	}
	if len(body) == 0 || len(body) > 1<<20 || !json.Valid(body) {
		return errors.New("measured x402 paid response body is empty, oversized or invalid JSON")
	}
	digest := sha256.Sum256(body)
	digestHex := hex.EncodeToString(digest[:])
	if !strings.Contains(response.Settlement.Detail, fmt.Sprintf("response_bytes=%d", len(body))) || !strings.Contains(response.Settlement.Detail, "response_sha256="+digestHex) {
		return fmt.Errorf("measured x402 response body does not match Treasury settlement evidence: detail=%q bytes=%d sha256=%s", response.Settlement.Detail, len(body), digestHex)
	}
	var catalog struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(body, &catalog); err != nil || len(catalog.Nodes) == 0 {
		return fmt.Errorf("measured x402 paid body has no usable Offer Desk catalog result: nodes=%d decode=%v", len(catalog.Nodes), err)
	}
	nodeIDs := make([]string, 0, len(catalog.Nodes))
	for _, node := range catalog.Nodes {
		if node.ID != "" {
			nodeIDs = append(nodeIDs, node.ID)
		}
	}
	if len(nodeIDs) == 0 {
		return errors.New("measured x402 paid catalog result omitted every node identity")
	}
	if err := write("x402-measured-result.json", map[string]any{
		"status": "settled", "run_id": request.RunID.String(), "authorization_id": authorizationID,
		"settlement_id": settlementID, "receipt_id": response.Settlement.ExternalID,
		"ledger_emission_id": settlementID + ":ledger", "amount_minor": response.Settlement.AmountMinor,
		"response_bytes": len(body), "response_sha256": digestHex, "body_path": bodyName,
		"price_id": config.X402PriceID, "call_id": config.X402CallID,
		"returned_node_ids": nodeIDs, "benefit_state": "measured", "operation_grant_id": approval.GrantID,
	}); err != nil {
		return err
	}
	return nil
}

func readCF1PaidResponseBody(evidenceRoot, evidenceDir, filename string, expectedDev, expectedIno uint64) ([]byte, error) {
	dirFD, err := openCF1EvidenceDirectory(evidenceRoot, evidenceDir, expectedDev, expectedIno)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(dirFD)
	return readCF1PaidResponseBodyAt(dirFD, filename)
}

func openCF1EvidenceDirectory(evidenceRoot, evidenceDir string, expectedDev, expectedIno uint64) (int, error) {
	if !filepath.IsAbs(evidenceRoot) || filepath.Clean(evidenceRoot) != evidenceRoot || !filepath.IsAbs(evidenceDir) || filepath.Clean(evidenceDir) != evidenceDir {
		return -1, errors.New("measured x402 evidence paths must be absolute and clean")
	}
	relativeDir, err := filepath.Rel(evidenceRoot, evidenceDir)
	if err != nil || relativeDir == "" || relativeDir == "." || relativeDir == ".." || strings.HasPrefix(relativeDir, ".."+string(filepath.Separator)) || filepath.Dir(relativeDir) != "." {
		return -1, errors.New("measured x402 evidence directory is not a direct child of its run root")
	}
	rootFD, err := openCF1AbsoluteDirectoryNoFollow(evidenceRoot)
	if err != nil {
		return -1, fmt.Errorf("open measured x402 evidence root by pinned components: %w", err)
	}
	var rootStat syscall.Stat_t
	if err := syscall.Fstat(rootFD, &rootStat); err != nil {
		_ = syscall.Close(rootFD)
		return -1, fmt.Errorf("inspect pinned measured x402 evidence root: %w", err)
	}
	if rootStat.Mode&syscall.S_IFMT != syscall.S_IFDIR || rootStat.Mode&0o077 != 0 {
		_ = syscall.Close(rootFD)
		return -1, errors.New("measured x402 evidence root is not a private directory")
	}
	dirFD, err := syscall.Openat(rootFD, relativeDir, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
	_ = syscall.Close(rootFD)
	if err != nil {
		return -1, fmt.Errorf("open measured x402 evidence directory relative to pinned run root: %w", err)
	}
	var dirStat syscall.Stat_t
	if err := syscall.Fstat(dirFD, &dirStat); err != nil {
		_ = syscall.Close(dirFD)
		return -1, fmt.Errorf("inspect pinned measured x402 evidence directory: %w", err)
	}
	if dirStat.Mode&syscall.S_IFMT != syscall.S_IFDIR || dirStat.Mode&0o077 != 0 || uint64(dirStat.Dev) != expectedDev || dirStat.Ino != expectedIno {
		_ = syscall.Close(dirFD)
		return -1, errors.New("measured x402 evidence directory identity or privacy differs from trusted setup")
	}
	return dirFD, nil
}

func openCF1AbsoluteDirectoryNoFollow(path string) (int, error) {
	fd, err := syscall.Open(string(filepath.Separator), syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
	if err != nil {
		return -1, err
	}
	components := strings.Split(strings.TrimPrefix(filepath.ToSlash(path), "/"), "/")
	for _, component := range components {
		if component == "" {
			continue
		}
		if component == "." || component == ".." {
			_ = syscall.Close(fd)
			return -1, errors.New("path component traversal is not allowed")
		}
		next, openErr := syscall.Openat(fd, component, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
		_ = syscall.Close(fd)
		if openErr != nil {
			return -1, openErr
		}
		fd = next
	}
	return fd, nil
}

func readCF1PaidResponseBodyAt(dirFD int, filename string) ([]byte, error) {
	if filename == "" || filepath.Base(filename) != filename {
		return nil, errors.New("measured x402 body filename must not contain path components")
	}
	bodyFD, err := syscall.Openat(dirFD, filename, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("open Treasury paid response body without following links: %w", err)
	}
	bodyFile := os.NewFile(uintptr(bodyFD), filename)
	if bodyFile == nil {
		_ = syscall.Close(bodyFD)
		return nil, errors.New("open Treasury paid response body returned an invalid file handle")
	}
	defer bodyFile.Close()
	var bodyInfo syscall.Stat_t
	if err := syscall.Fstat(bodyFD, &bodyInfo); err != nil || bodyInfo.Mode&syscall.S_IFMT != syscall.S_IFREG || bodyInfo.Mode&0o077 != 0 || bodyInfo.Size > 1<<20 {
		return nil, errors.New("Treasury paid response body is not a private bounded regular file")
	}
	body, err := io.ReadAll(io.LimitReader(bodyFile, (1<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("read bounded Treasury paid response body: %w", err)
	}
	if len(body) > 1<<20 {
		return nil, errors.New("Treasury paid response body exceeded the evidence read limit")
	}
	return body, nil
}

func TestExecuteCF1MeasuredX402CallProducesBodyBoundBenefitManifest(t *testing.T) {
	root := privateEvidenceDir(t)
	coord := filepath.Join(root, "agent-manager-cli")
	if err := os.MkdirAll(coord, 0o700); err != nil {
		t.Fatal(err)
	}
	setCF1EvidenceDirectoryIdentity(t, coord)
	t.Setenv(treasuryCF1MeasuredX402Env, "1")
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_DIR", coord)
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_ROOT", root)
	runID := uuid.New()
	config := cf1TreasuryRunConfig{
		APIBaseURL: "http://127.0.0.1:18500", BudgetID: "budget", MandateID: "mandate",
		X402CatalogURL: "http://offers.test/catalog", X402PriceID: "price-1", X402CallID: "call-1",
		X402Counterparty: "offers.test", X402InstrumentID: "instrument", X402AmountMinor: 1,
		X402AuthorizationID: "auth", X402SettlementID: "settlement", X402IdempotencyKey: "idem",
	}
	settlementID := config.X402SettlementID + "-measured"
	proposalTarget, settlementTarget := cf1MeasuredX402Targets(config)
	body := []byte(`{"nodes":[{"id":"offer-1"},{"id":"offer-2"}]}`)
	bodyPath := filepath.Join(coord, "x402-paid-response-"+settlementID+".body")
	if err := os.WriteFile(bodyPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	grantID := uuid.NewString()
	approval, err := json.Marshal(map[string]any{
		"run_id": runID.String(), "grant_id": grantID,
		"spend_targets": []string{proposalTarget, settlementTarget},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coord, "x402-measured-operation-approved.json"), approval, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coord, "x402-measured-call-release"), []byte("released"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := runner.ExecuteRequest{RunID: runID, Profile: &domain.AgentProfile{}, Task: &domain.Task{}}
	var commands []string
	runCLI := func(label string, args ...string) ([]byte, error) {
		commands = append(commands, label)
		if label == "settlement-report-x402-measured" {
			payload, marshalErr := json.Marshal(map[string]any{"settlement": map[string]any{
				"id": settlementID, "outcome": "CHARGE_OUTCOME_SETTLED", "external_id": "0xabc123",
				"amount_minor": 1, "authorization_id": config.X402AuthorizationID + "-measured",
				"detail": fmt.Sprintf("response_bytes=%d; response_sha256=%s", len(body), hex.EncodeToString(digest[:])),
			}})
			return payload, marshalErr
		}
		return []byte(`{}`), nil
	}
	var manifest map[string]any
	write := func(name string, value any) error {
		if name != "x402-measured-result.json" {
			return nil
		}
		encoded, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return marshalErr
		}
		return json.Unmarshal(encoded, &manifest)
	}
	if err := executeCF1MeasuredX402Call(context.Background(), request, config, coord, runCLI, write); err != nil {
		t.Fatal(err)
	}
	if len(commands) != 2 || manifest["status"] != "settled" || manifest["run_id"] != runID.String() || manifest["settlement_id"] != settlementID || manifest["receipt_id"] != "0xabc123" || manifest["ledger_emission_id"] != settlementID+":ledger" || manifest["response_bytes"] != float64(len(body)) || manifest["response_sha256"] != hex.EncodeToString(digest[:]) || manifest["body_path"] != filepath.Base(bodyPath) || manifest["benefit_state"] != "measured" {
		t.Fatalf("measured x402 manifest or command sequence incomplete: commands=%v manifest=%+v", commands, manifest)
	}
}

func TestExecuteCF1MeasuredX402CallRejectsBodyThatDiffersFromTreasuryReceipt(t *testing.T) {
	root := privateEvidenceDir(t)
	coord := filepath.Join(root, "agent-manager-cli")
	if err := os.MkdirAll(coord, 0o700); err != nil {
		t.Fatal(err)
	}
	setCF1EvidenceDirectoryIdentity(t, coord)
	t.Setenv(treasuryCF1MeasuredX402Env, "1")
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_DIR", coord)
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_ROOT", root)
	runID := uuid.New()
	config := cf1TreasuryRunConfig{
		APIBaseURL: "http://127.0.0.1:18500", BudgetID: "budget", MandateID: "mandate",
		X402CatalogURL: "http://offers.test/catalog", X402PriceID: "price-1", X402CallID: "call-1",
		X402Counterparty: "offers.test", X402InstrumentID: "instrument", X402AmountMinor: 1,
		X402AuthorizationID: "auth", X402SettlementID: "settlement", X402IdempotencyKey: "idem",
	}
	settlementID := config.X402SettlementID + "-measured"
	proposalTarget, settlementTarget := cf1MeasuredX402Targets(config)
	if err := os.WriteFile(filepath.Join(coord, "x402-paid-response-"+settlementID+".body"), []byte(`{"nodes":[{"id":"changed"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	approval, _ := json.Marshal(map[string]any{
		"run_id": runID.String(), "grant_id": uuid.NewString(),
		"spend_targets": []string{proposalTarget, settlementTarget},
	})
	if err := os.WriteFile(filepath.Join(coord, "x402-measured-operation-approved.json"), approval, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coord, "x402-measured-call-release"), []byte("released"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := runner.ExecuteRequest{RunID: runID, Profile: &domain.AgentProfile{}, Task: &domain.Task{}}
	runCLI := func(label string, _ ...string) ([]byte, error) {
		if label == "settlement-report-x402-measured" {
			return []byte(`{"settlement":{"id":"settlement-measured","outcome":"CHARGE_OUTCOME_SETTLED","external_id":"0xabc123","amount_minor":1,"authorization_id":"auth-measured","detail":"response_bytes=32; response_sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`), nil
		}
		return []byte(`{}`), nil
	}
	write := func(string, any) error { return nil }
	err := executeCF1MeasuredX402Call(context.Background(), request, config, coord, runCLI, write)
	if err == nil || !strings.Contains(err.Error(), "does not match Treasury settlement evidence") {
		t.Fatalf("mismatched paid body should be rejected, got %v", err)
	}
}

func TestReadCF1PaidResponseBodyRejectsSymlinksAndOversize(t *testing.T) {
	root := privateEvidenceDir(t)
	dir := filepath.Join(root, "agent-manager-cli")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	dev, ino, err := cf1EvidenceDirectoryIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("symlink", func(t *testing.T) {
		outside := filepath.Join(root, "outside.body")
		if err := os.WriteFile(outside, []byte(`{"nodes":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(dir, "body.bin")); err != nil {
			t.Fatal(err)
		}
		if _, err := readCF1PaidResponseBody(root, dir, "body.bin", dev, ino); err == nil {
			t.Fatal("paid response symlink was accepted")
		}
	})
	t.Run("oversized", func(t *testing.T) {
		path := filepath.Join(dir, "oversized.body")
		if err := os.WriteFile(path, []byte(strings.Repeat("x", (1<<20)+1)), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readCF1PaidResponseBody(root, dir, "oversized.body", dev, ino); err == nil {
			t.Fatal("oversized paid response body was accepted")
		}
	})
}

func TestReadCF1PaidResponseBodyRejectsEvidenceDirectoryReplacementBeforePin(t *testing.T) {
	root := privateEvidenceDir(t)
	dir := filepath.Join(root, "agent-manager-cli")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	dev, ino, err := cf1EvidenceDirectoryIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(dir, dir+".expected"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := openCF1EvidenceDirectory(root, dir, dev, ino); err == nil {
		t.Fatal("substituted evidence directory was accepted before descriptor pinning")
	}
}

func TestReadCF1PaidResponseBodyStaysOnPinnedRootAfterAncestorReplacement(t *testing.T) {
	parent := t.TempDir()
	ancestor := filepath.Join(parent, "run-parent")
	root := filepath.Join(ancestor, "current-run")
	dir := filepath.Join(root, "agent-manager-cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside")
	outsideDir := filepath.Join(outside, "agent-manager-cli")
	if err := os.MkdirAll(outsideDir, 0o700); err != nil {
		t.Fatal(err)
	}
	const filename = "x402-paid-response-settlement.body"
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(`{"nodes":[{"id":"original"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideDir, filename), []byte(`{"nodes":[{"id":"attacker"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	dev, ino, err := cf1EvidenceDirectoryIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	pinnedDirFD, err := openCF1EvidenceDirectory(root, dir, dev, ino)
	if err != nil {
		t.Fatalf("pin evidence path before the simulated replacement: %v", err)
	}
	defer syscall.Close(pinnedDirFD)
	renamedAncestor := ancestor + ".pinned"
	if err := os.Rename(ancestor, renamedAncestor); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, ancestor); err != nil {
		t.Fatal(err)
	}
	body, err := readCF1PaidResponseBodyAt(pinnedDirFD, filename)
	if err != nil {
		t.Fatalf("read through pinned descriptors after ancestor replacement: %v", err)
	}
	if strings.Contains(string(body), "attacker") || !strings.Contains(string(body), "original") {
		t.Fatalf("ancestor replacement redirected the artifact read: %s", body)
	}
}

func TestCF1X402ReceiptReferenceContinuityAcrossRestart(t *testing.T) {
	const receipt = `{"x402Version":2,"payload":"signed-local-fork-test-payment"}`
	digest, length, err := hashCF1X402ReceiptReference(receipt)
	if err != nil {
		t.Fatal(err)
	}
	retryDigest, retryLength, err := compareCF1X402ReceiptReferences(receipt, receipt)
	if err != nil || retryDigest != digest || retryLength != length {
		t.Fatalf("exact retry receipt reference was not continuous: digest=%s/%s length=%d/%d err=%v", retryDigest, digest, retryLength, length, err)
	}
	if _, _, err := compareCF1X402ReceiptReferences(receipt, receipt+"changed"); err == nil {
		t.Fatal("changed receipt reference was accepted as an exact restart retry")
	}
	if _, _, err := hashCF1X402ReceiptReference(""); err == nil {
		t.Fatal("empty receipt reference was accepted")
	}
}

// cf1MeasuredX402Targets is the digest-bound spend contract the Treasury CLI
// now records; a short authorization/settlement id is no longer a valid target.
func cf1MeasuredX402Targets(config cf1TreasuryRunConfig) (string, string) {
	idempotencyKey := config.X402IdempotencyKey + "-measured"
	return (cliutil.TreasurySpendIntent{
		AuthorizationID: config.X402AuthorizationID + "-measured", MandateID: config.MandateID,
		AmountMinor: config.X402AmountMinor, Currency: "USD", Counterparty: config.X402Counterparty,
		AuthorizationIdempotencyKey: idempotencyKey,
		SettlementID:                config.X402SettlementID + "-measured", InstrumentID: config.X402InstrumentID,
		SettlementIdempotencyKey: idempotencyKey + "-settlement",
	}).Targets()
}
