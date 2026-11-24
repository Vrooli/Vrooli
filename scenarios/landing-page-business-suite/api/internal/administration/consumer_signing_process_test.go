package administration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/vrooli/api-core/consumeridentity"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	signerWorkerModeEnv      = "LPBS_SIGNER_PROCESS_MODE"
	signerWorkerAuthorityEnv = "LPBS_SIGNER_PROCESS_AUTHORITY"
	signerWorkerPassFileEnv  = "LPBS_SIGNER_PROCESS_PASS_FILE"
	signerWorkerRequestEnv   = "LPBS_SIGNER_PROCESS_REQUEST"
	signerWorkerGateEnv      = "LPBS_SIGNER_PROCESS_GATE"
)

type signerWorkerRequest struct {
	Candidate       credentialauthority.CandidateRef `json:"candidate"`
	ExpectedVersion string                           `json:"expected_version"`
}

type signerWorkerReceipt struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error_type,omitempty"`
}

type signerProcessMetadata struct {
	Version         string `json:"version"`
	KeyID           string `json:"key_id"`
	OverlapKeyCount int    `json:"overlap_key_count"`
}

// TestConsumerSigningProcessWorker is re-executed as a real child process by
// the integration tests below. Its environment contains only synthetic store
// paths and a mode; private key bytes remain inside the encrypted test store.
func TestConsumerSigningProcessWorker(t *testing.T) {
	mode := os.Getenv(signerWorkerModeEnv)
	if mode == "" {
		return
	}
	authority, err := openSignerWorkerAuthority()
	if err != nil {
		t.Fatalf("worker authority initialization failed: %v", err)
	}
	switch mode {
	case "activate-candidate":
		requestBytes, err := os.ReadFile(os.Getenv(signerWorkerRequestEnv))
		if err != nil {
			t.Fatal("worker candidate request unavailable")
		}
		var request signerWorkerRequest
		if err := json.Unmarshal(requestBytes, &request); err != nil {
			t.Fatal("worker candidate request invalid")
		}
		fmt.Fprintln(os.Stdout, "READY")
		if err := waitForSignerWorkerGate(os.Getenv(signerWorkerGateEnv)); err != nil {
			t.Fatal(err)
		}
		receipt := signerWorkerReceipt{Status: "activated"}
		if err := authority.ActivateCandidateIfVersion(request.Candidate, request.ExpectedVersion); err != nil {
			if errors.Is(err, credentialauthority.ErrVersionConflict) {
				receipt.Status = "version_conflict"
			} else {
				receipt.Status = "error"
				receipt.Error = fmt.Sprintf("%T: %v", err, err)
			}
		} else {
			receipt.Version = request.Candidate.Version
		}
		status := authority.Status(credentialAuthorityIdentity, consumerSigningAuthorityField)
		if status.Version != "" {
			receipt.Version = status.Version
		}
		if receipt.Version == request.Candidate.Version {
			// Candidate cleanup can fail after the pointer commit. The active
			// version receipt is authoritative for the commit result.
			receipt.Status = "activated"
		}
		if err := json.NewEncoder(os.Stdout).Encode(receipt); err != nil {
			t.Fatal("worker metadata receipt write failed")
		}
	case "serve-signer":
		if err := runSignerHTTPWorker(authority); err != nil {
			t.Fatalf("signer worker failed: %v", err)
		}
	default:
		t.Fatal("unknown signer worker mode")
	}
}

func TestConsumerSigningCASAllowsSingleWinnerAcrossProcesses(t *testing.T) {
	authorityPath, passphraseFile, authority := newProcessSignerAuthority(t)
	t.Setenv("HOME", filepath.Dir(passphraseFile))
	active := authority.Status(credentialAuthorityIdentity, consumerSigningAuthorityField)
	if active.Version == "" {
		t.Fatal("initial authority version unavailable")
	}
	candidates := make([]credentialauthority.CandidateRef, 0, 2)
	for _, keyID := range []string{"process-candidate-a", "process-candidate-b"} {
		signer, err := consumeridentity.GenerateSigner(keyID, "lpbs.process.test", time.Minute)
		if err != nil {
			t.Fatal("synthetic candidate key generation failed")
		}
		privatePEM, err := consumeridentity.MarshalPrivateKeyPEM(signer.Private)
		if err != nil {
			t.Fatal("synthetic candidate key encoding failed")
		}
		bundle, err := json.Marshal(ConsumerSigningCredential{Format: consumerSigningBundleFormat, KeyID: keyID, PrivateKeyPEM: string(privatePEM)})
		if err != nil {
			t.Fatal("synthetic candidate bundle encoding failed")
		}
		candidate, err := authority.PutCandidate(credentialAuthorityIdentity, consumerSigningAuthorityField, string(bundle))
		if err != nil {
			t.Fatal("synthetic authority candidate staging failed")
		}
		candidates = append(candidates, candidate)
	}
	gatePath := filepath.Join(t.TempDir(), "activate.gate")
	workers := make([]*signerWorkerProcess, 0, len(candidates))
	for _, candidate := range candidates {
		requestPath := filepath.Join(t.TempDir(), "candidate.json")
		request, _ := json.Marshal(signerWorkerRequest{Candidate: candidate, ExpectedVersion: active.Version})
		if err := os.WriteFile(requestPath, request, 0o600); err != nil {
			t.Fatal(err)
		}
		worker := startSignerWorker(t, "activate-candidate", authorityPath, passphraseFile, requestPath, gatePath)
		if worker.ready != "READY" {
			t.Fatalf("candidate worker readiness = %q", worker.ready)
		}
		workers = append(workers, worker)
	}
	if err := os.WriteFile(gatePath, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	activated, conflicted := 0, 0
	for _, worker := range workers {
		var receipt signerWorkerReceipt
		output := worker.finish(t)
		var receiptLine string
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "{") {
				receiptLine = line
				break
			}
		}
		if err := json.Unmarshal([]byte(receiptLine), &receipt); err != nil {
			t.Fatalf("candidate worker returned no metadata receipt: %v", err)
		}
		if receipt.Status == "activated" {
			activated++
		} else if receipt.Status == "version_conflict" {
			conflicted++
		} else {
			t.Fatalf("candidate activation status = %q version=%q error=%q", receipt.Status, receipt.Version, receipt.Error)
		}
	}
	if activated != 1 || conflicted != 1 {
		t.Fatalf("subprocess CAS outcomes activated=%d conflicted=%d; want one winner and one stale writer", activated, conflicted)
	}
	finalStatus := authority.Status(credentialAuthorityIdentity, consumerSigningAuthorityField)
	if finalStatus.Version == "" || finalStatus.Version == active.Version {
		t.Fatal("subprocess activation did not advance the active authority version")
	}
}

func TestConsumerSigningSubprocessReloadAndExternalStandardsJWKSVerifier(t *testing.T) {
	authorityPath, passphraseFile, authority := newProcessSignerAuthority(t)
	t.Setenv("HOME", filepath.Dir(passphraseFile))
	initialStatus := authority.Status(credentialAuthorityIdentity, consumerSigningAuthorityField)
	initialBinding, err := ResolveBoundCredentialFromAuthority(authority, credentialAuthorityIdentity, consumerSigningAuthorityField, ConsumerSigningAuthorityScope)
	if err != nil {
		t.Fatal("initial synthetic signer binding unavailable")
	}
	initialBundle, err := ParseConsumerSigningCredential(initialBinding.Value, "initial", "")
	if err != nil {
		t.Fatal("initial synthetic signer bundle invalid")
	}
	worker := startSignerWorker(t, "serve-signer", authorityPath, passphraseFile, "", "")
	firstProcess := worker
	if !strings.HasPrefix(worker.ready, "READY ") {
		t.Fatalf("signer process readiness = %q", worker.ready)
	}
	baseURL := "http://" + strings.TrimPrefix(worker.ready, "READY ")
	workerRunning := true
	defer func() {
		if workerRunning {
			worker.stop(t, baseURL)
		}
	}()

	oldToken, err := getSignerWorkerToken(baseURL)
	if err != nil {
		t.Fatal("initial signer process did not issue a token")
	}
	initialJWKS, err := getSignerWorkerJWKS(baseURL)
	if err != nil {
		t.Fatal("initial signer process did not publish JWKS")
	}
	if err := verifyStandardsJWKS(initialJWKS, oldToken, "lpbs.process.test", time.Now().UTC()); err != nil {
		t.Fatalf("independent JOSE verifier rejected initial token: %v", err)
	}
	initialMetadata, err := getSignerWorkerMetadata(baseURL)
	if err != nil || initialMetadata.Version != initialStatus.Version || initialMetadata.KeyID != "process-initial" {
		t.Fatalf("initial child metadata = %+v err=%v", initialMetadata, err)
	}
	oldKid, err := signedTokenKeyID(oldToken)
	if err != nil {
		t.Fatal("old token key ID unavailable")
	}

	rotatedAt := time.Now().UTC()
	rotation, err := RotateConsumerSigningCredential(authority, ConsumerSigningRotationOptions{
		Identity: credentialAuthorityIdentity, Field: consumerSigningAuthorityField, Scope: ConsumerSigningAuthorityScope,
		ExpectedVersion: initialStatus.Version, LegacyKeyID: "initial", Issuer: "lpbs.process.test", AccessTTL: 5 * time.Second,
		MaxTokenTTL: 5 * time.Second, ClockSkew: 20 * time.Millisecond, JWKSCacheMaxAge: 20 * time.Millisecond, Now: rotatedAt,
	})
	if err != nil {
		t.Fatal("synthetic authority rotation failed")
	}
	newToken, err := getSignerWorkerToken(baseURL)
	if err != nil {
		t.Fatal("live child process did not reload and issue with the new active key")
	}
	newJWKS, err := getSignerWorkerJWKS(baseURL)
	if err != nil {
		t.Fatal("reloaded child process did not publish JWKS")
	}
	newKid, err := signedTokenKeyID(newToken)
	if err != nil || newKid == oldKid {
		t.Fatalf("child process did not switch signer key ID: old=%q new=%q err=%v", oldKid, newKid, err)
	}
	rotatedMetadata, err := getSignerWorkerMetadata(baseURL)
	if err != nil || rotatedMetadata.Version != rotation.Receipt.Version || rotatedMetadata.KeyID != newKid || rotatedMetadata.OverlapKeyCount != 1 {
		t.Fatalf("rotated child metadata = %+v err=%v; want active version and one overlap key", rotatedMetadata, err)
	}
	if err := verifyStandardsJWKS(newJWKS, oldToken, "lpbs.process.test", time.Now().UTC()); err != nil {
		t.Fatalf("external verifier rejected old token during overlap: %v", err)
	}
	if err := verifyStandardsJWKS(newJWKS, newToken, "lpbs.process.test", time.Now().UTC()); err != nil {
		t.Fatalf("external verifier rejected new token: %v", err)
	}
	if !jwksHasKey(newJWKS, oldKid) || !jwksHasKey(newJWKS, newKid) {
		t.Fatal("published overlap JWKS omitted the old or active public key")
	}
	worker.stop(t, baseURL)
	workerRunning = false
	worker = startSignerWorker(t, "serve-signer", authorityPath, passphraseFile, "", "")
	workerRunning = true
	if !strings.HasPrefix(worker.ready, "READY ") {
		t.Fatalf("restarted signer process readiness = %q", worker.ready)
	}
	baseURL = "http://" + strings.TrimPrefix(worker.ready, "READY ")
	restartedToken, err := getSignerWorkerToken(baseURL)
	if err != nil {
		t.Fatal("restarted process failed to issue from the active authority version")
	}
	restartedJWKS, err := getSignerWorkerJWKS(baseURL)
	if err != nil || !jwksHasKey(restartedJWKS, oldKid) || !jwksHasKey(restartedJWKS, newKid) {
		t.Fatalf("restarted process did not load active and overlap public keys: err=%v old=%s new=%s jwks=%s", err, oldKid, newKid, publicJWKSKeyIDs(restartedJWKS))
	}
	if err := verifyStandardsJWKS(restartedJWKS, oldToken, "lpbs.process.test", time.Now().UTC()); err != nil {
		t.Fatalf("standards verifier rejected pre-restart token during overlap: %v", err)
	}
	if err := verifyStandardsJWKS(restartedJWKS, restartedToken, "lpbs.process.test", time.Now().UTC()); err != nil {
		t.Fatalf("standards verifier rejected restarted process token: %v", err)
	}
	restartedMetadata, err := getSignerWorkerMetadata(baseURL)
	if err != nil || restartedMetadata.Version != rotation.Receipt.Version || restartedMetadata.KeyID != newKid || restartedMetadata.OverlapKeyCount != 1 {
		t.Fatalf("restarted child metadata = %+v err=%v; want monotonically applied active version and overlap", restartedMetadata, err)
	}
	offlinePath := authorityPath + ".temporarily-unavailable"
	if err := os.Rename(authorityPath, offlinePath); err != nil {
		t.Fatal("could not simulate local authority outage")
	}
	if _, err := getSignerWorkerToken(baseURL); err == nil {
		t.Fatal("signer child issued a token while its authority backend was unavailable")
	}
	if _, err := getSignerWorkerJWKS(baseURL); err == nil {
		t.Fatal("signer child published a keyset while its authority backend was unavailable")
	}
	if err := os.Rename(offlinePath, authorityPath); err != nil {
		t.Fatal("could not restore local authority fixture")
	}
	recoveredToken, err := getSignerWorkerToken(baseURL)
	if err != nil {
		t.Fatal("signer child failed to recover after authority restoration")
	}
	recoveredMetadata, err := getSignerWorkerMetadata(baseURL)
	if err != nil || recoveredMetadata.Version != rotation.Receipt.Version || recoveredMetadata.KeyID != newKid {
		t.Fatalf("recovered signer metadata = %+v err=%v", recoveredMetadata, err)
	}
	if err := verifyStandardsJWKS(restartedJWKS, recoveredToken, "lpbs.process.test", time.Now().UTC()); err != nil {
		t.Fatalf("external verifier rejected recovered child token: %v", err)
	}
	privateBinding, err := ResolveBoundCredentialFromAuthority(authority, credentialAuthorityIdentity, consumerSigningAuthorityField, ConsumerSigningAuthorityScope)
	if err != nil {
		t.Fatal("current synthetic private signer could not be read from the fixture authority")
	}
	if bytes.Contains(newJWKS, []byte("PRIVATE KEY")) || bytes.Contains(newJWKS, []byte(privateBinding.Value)) {
		t.Fatal("HTTP JWKS response exposed synthetic private signing material")
	}
	currentBundle, err := ParseConsumerSigningCredential(privateBinding.Value, "initial", "")
	if err != nil {
		t.Fatal("active synthetic signer bundle invalid")
	}
	forged := mutateJWTSignature(newToken)
	if err := verifyStandardsJWKS(newJWKS, forged, "lpbs.process.test", time.Now().UTC()); err == nil {
		t.Fatal("external verifier accepted a forged signature")
	}
	unknownKidToken := mutateJWTKeyID(newToken, "unknown-synthetic-kid")
	if err := verifyStandardsJWKS(newJWKS, unknownKidToken, "lpbs.process.test", time.Now().UTC()); err == nil {
		t.Fatal("external verifier accepted an unknown key ID")
	}

	if wait := time.Until(rotation.Receipt.PreviousKeysExpireAt.Add(50 * time.Millisecond)); wait > 0 {
		time.Sleep(wait)
	}
	if err := verifyStandardsJWKS(restartedJWKS, oldToken, "lpbs.process.test", time.Now().UTC()); !errors.Is(err, josejwt.ErrExpired) {
		t.Fatalf("old token after its bounded overlap = %v, want expired", err)
	}
	postOverlapJWKS, err := getSignerWorkerJWKS(baseURL)
	if err != nil {
		t.Fatal("signer process failed to refresh after overlap expiry")
	}
	if jwksHasKey(postOverlapJWKS, oldKid) || !jwksHasKey(postOverlapJWKS, newKid) {
		t.Fatal("post-overlap JWKS did not remove the expired key and retain the active key")
	}
	currentToken, err := getSignerWorkerToken(baseURL)
	if err != nil || verifyStandardsJWKS(postOverlapJWKS, currentToken, "lpbs.process.test", time.Now().UTC()) != nil {
		t.Fatal("child process did not recover issuance with the active key after overlap cleanup")
	}
	worker.stop(t, baseURL)
	workerRunning = false

	for _, process := range []*signerWorkerProcess{firstProcess, worker} {
		workerMetadata := append([]byte(process.ready+"\n"), process.output...)
		workerMetadata = append(workerMetadata, process.stderr.Bytes()...)
		assertNoSignerPrivateMaterial(t, workerMetadata, initialBundle.PrivateKeyPEM, currentBundle.PrivateKeyPEM)
		workerArguments := []byte(strings.Join(process.cmd.Args, "\n") + "\n" + strings.Join(process.cmd.Env, "\n"))
		assertNoSignerPrivateMaterial(t, workerArguments, initialBundle.PrivateKeyPEM, currentBundle.PrivateKeyPEM)
	}
	if bytes.Contains(newJWKS, []byte(currentBundle.PrivateKeyPEM)) || bytes.Contains(restartedJWKS, []byte(currentBundle.PrivateKeyPEM)) {
		t.Fatal("public HTTP JWKS exposed active synthetic private key bytes")
	}
}

func openSignerWorkerAuthority() (*credentialauthority.Authority, error) {
	path := os.Getenv(signerWorkerAuthorityEnv)
	passFile := os.Getenv(signerWorkerPassFileEnv)
	passphrase, err := os.ReadFile(passFile)
	if err != nil {
		return nil, errors.New("test authority passphrase file unavailable")
	}
	return credentialauthority.OpenEncryptedFile(path, string(passphrase))
}

func runSignerHTTPWorker(authority *credentialauthority.Authority) error {
	source, err := NewAuthorityConsumerSigningSource(authority, credentialAuthorityIdentity, consumerSigningAuthorityField, ConsumerSigningAuthorityScope, "initial", "", "lpbs.process.test", 5*time.Second)
	if err != nil {
		return err
	}
	binding, err := ResolveBoundCredentialFromAuthority(authority, credentialAuthorityIdentity, consumerSigningAuthorityField, ConsumerSigningAuthorityScope)
	if err != nil {
		return errors.New("test signer binding unavailable")
	}
	bundle, err := ParseConsumerSigningCredential(binding.Value, "initial", "")
	if err != nil {
		return errors.New("test signer bundle invalid")
	}
	previousKeys := make([]consumeridentity.PublicKey, 0)
	if strings.TrimSpace(bundle.PreviousJWKS) != "" {
		previousSet, err := consumeridentity.ParseJWKS([]byte(bundle.PreviousJWKS))
		if err != nil {
			return errors.New("test signer overlap set invalid")
		}
		for id, key := range previousSet.Keys {
			previousKeys = append(previousKeys, consumeridentity.PublicKey{ID: id, Key: key})
		}
	}
	service := NewUserAuthService(UserAuthServiceOptions{
		JWTIssuer: "lpbs.process.test", ConsumerSigningKeyPEM: bundle.PrivateKeyPEM, ConsumerSigningKeyID: bundle.KeyID,
		ConsumerPreviousKeys: previousKeys, ConsumerPreviousKeyExpirations: bundle.PreviousKeyExpirations,
		ConsumerSigningVersion: binding.Version, ConsumerSigningSource: source, AccessTTL: 5 * time.Second,
	})
	if service == nil {
		return errors.New("test signer service could not initialize")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		token, _, err := service.GenerateAccessToken("process-user", "synthetic@example.test", "process-session")
		if err != nil {
			http.Error(w, "signing unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, token)
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		jwks, err := service.PublicKeySet()
		if err != nil {
			http.Error(w, "keyset unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/jwk-set+json")
		_, _ = w.Write(jwks)
	})
	mux.HandleFunc("/meta", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := service.RefreshConsumerSigningKeyset(); err != nil {
			http.Error(w, "keyset unavailable", http.StatusServiceUnavailable)
			return
		}
		service.consumerKeyMu.RLock()
		metadata := signerProcessMetadata{Version: service.consumerSigningVersion, KeyID: service.consumerSigner.KeyID, OverlapKeyCount: len(service.consumerPreviousKeys)}
		service.consumerKeyMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metadata)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return errors.New("test signer listener unavailable")
	}
	server := &http.Server{Handler: mux}
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		go func() { _ = server.Shutdown(context.Background()) }()
	})
	fmt.Fprintf(os.Stdout, "READY %s\n", listener.Addr().String())
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.New("test signer server stopped unexpectedly")
	}
	return nil
}

type signerWorkerProcess struct {
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   bytes.Buffer
	ready    string
	output   []byte
	finished bool
}

func startSignerWorker(t *testing.T, mode, authorityPath, passphraseFile, requestPath, gatePath string) *signerWorkerProcess {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal("test executable path unavailable")
	}
	cmd := exec.Command(executable, "-test.run=^TestConsumerSigningProcessWorker$")
	cmd.Env = []string{
		"HOME=" + filepath.Dir(passphraseFile),
		signerWorkerModeEnv + "=" + mode,
		signerWorkerAuthorityEnv + "=" + authorityPath,
		signerWorkerPassFileEnv + "=" + passphraseFile,
		signerWorkerRequestEnv + "=" + requestPath,
		signerWorkerGateEnv + "=" + gatePath,
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal("signer worker stdout pipe unavailable")
	}
	worker := &signerWorkerProcess{cmd: cmd, stdout: stdout}
	cmd.Stderr = &worker.stderr
	if err := cmd.Start(); err != nil {
		t.Fatal("signer worker failed to start")
	}
	t.Cleanup(func() {
		if worker.finished || cmd.Process == nil || cmd.ProcessState != nil {
			return
		}
		_ = cmd.Process.Kill()
		_, _ = io.Copy(io.Discard, stdout)
		_ = cmd.Wait()
		worker.finished = true
	})
	readiness := make(chan string, 1)
	readErr := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(stdout).ReadString('\n')
		if err != nil {
			readErr <- err
			return
		}
		readiness <- strings.TrimSpace(line)
	}()
	select {
	case worker.ready = <-readiness:
	case err := <-readErr:
		t.Fatalf("signer worker did not become ready: %v", err)
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("signer worker readiness timed out")
	}
	return worker
}

func (worker *signerWorkerProcess) finish(t *testing.T) []byte {
	t.Helper()
	output, err := io.ReadAll(worker.stdout)
	if err != nil {
		t.Fatal("signer worker output could not be read")
	}
	if err := worker.cmd.Wait(); err != nil {
		t.Fatalf("signer worker exited unsuccessfully: %v", err)
	}
	worker.finished = true
	if bytes.Contains(output, []byte("PRIVATE KEY")) || bytes.Contains(worker.stderr.Bytes(), []byte("PRIVATE KEY")) {
		t.Fatal("signer subprocess output contained a private key marker")
	}
	worker.output = append([]byte(nil), output...)
	return output
}

func (worker *signerWorkerProcess) stop(t *testing.T, baseURL string) {
	t.Helper()
	response, err := http.Get(baseURL + "/shutdown")
	if err == nil {
		_ = response.Body.Close()
	}
	_ = worker.finish(t)
}

func newProcessSignerAuthority(t *testing.T) (string, string, *credentialauthority.Authority) {
	t.Helper()
	dir := t.TempDir()
	authorityPath := filepath.Join(dir, "authority.enc")
	passphraseFile := filepath.Join(dir, "authority.passphrase")
	passphrase := "synthetic-local-test-store-passphrase-only"
	if err := os.WriteFile(passphraseFile, []byte(passphrase), 0o600); err != nil {
		t.Fatal("synthetic authority passphrase file unavailable")
	}
	authority, err := credentialauthority.OpenEncryptedFile(authorityPath, passphrase)
	if err != nil {
		t.Fatalf("isolated encrypted authority fixture unavailable: %v", err)
	}
	signer, err := consumeridentity.GenerateSigner("process-initial", "lpbs.process.test", time.Second)
	if err != nil {
		t.Fatal("initial synthetic signer generation failed")
	}
	privatePEM, err := consumeridentity.MarshalPrivateKeyPEM(signer.Private)
	if err != nil {
		t.Fatal("initial synthetic signer encoding failed")
	}
	bundle, err := json.Marshal(ConsumerSigningCredential{Format: consumerSigningBundleFormat, KeyID: "process-initial", PrivateKeyPEM: string(privatePEM)})
	if err != nil {
		t.Fatal("initial synthetic bundle encoding failed")
	}
	candidate, err := authority.PutCandidate(credentialAuthorityIdentity, consumerSigningAuthorityField, string(bundle))
	if err != nil {
		t.Fatal("initial synthetic candidate staging failed")
	}
	if err := authority.ActivateCandidate(candidate); err != nil {
		t.Fatal("initial synthetic signer activation failed")
	}
	return authorityPath, passphraseFile, authority
}

func waitForSignerWorkerGate(path string) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return errors.New("candidate activation gate timed out")
}

func getSignerWorkerToken(baseURL string) (string, error) {
	response, err := http.Get(baseURL + "/token")
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("signer process refused token issuance")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 16<<10))
	return string(data), err
}

func getSignerWorkerJWKS(baseURL string) ([]byte, error) {
	response, err := http.Get(baseURL + "/jwks")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("signer process refused JWKS publication")
	}
	return io.ReadAll(io.LimitReader(response.Body, 64<<10))
}

func getSignerWorkerMetadata(baseURL string) (signerProcessMetadata, error) {
	var metadata signerProcessMetadata
	response, err := http.Get(baseURL + "/meta")
	if err != nil {
		return metadata, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return metadata, errors.New("signer process refused metadata request")
	}
	err = json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&metadata)
	return metadata, err
}

func verifyStandardsJWKS(jwks []byte, token string, issuer string, now time.Time) error {
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(jwks, &set); err != nil {
		return err
	}
	parsed, err := josejwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return err
	}
	if len(parsed.Headers) != 1 {
		return errors.New("expected one signed JWT header")
	}
	keys := set.Key(parsed.Headers[0].KeyID)
	if len(keys) != 1 {
		return errors.New("JWKS key ID is missing or ambiguous")
	}
	claims := josejwt.Claims{}
	if err := parsed.Claims(keys[0].Key, &claims); err != nil {
		return err
	}
	return claims.ValidateWithLeeway(josejwt.Expected{Issuer: issuer, Time: now}, 0)
}

func jwksHasKey(jwks []byte, keyID string) bool {
	var set jose.JSONWebKeySet
	if json.Unmarshal(jwks, &set) != nil {
		return false
	}
	return len(set.Key(keyID)) == 1
}

func assertNoSignerPrivateMaterial(t *testing.T, observed []byte, privatePEMs ...string) {
	t.Helper()
	if bytes.Contains(observed, []byte("PRIVATE KEY")) {
		t.Fatal("signer process metadata exposed a private key marker")
	}
	for _, privatePEM := range privatePEMs {
		if strings.TrimSpace(privatePEM) != "" && bytes.Contains(observed, []byte(privatePEM)) {
			t.Fatal("signer process metadata exposed private signing bytes")
		}
	}
}

func publicJWKSKeyIDs(jwks []byte) []string {
	var set jose.JSONWebKeySet
	if json.Unmarshal(jwks, &set) != nil {
		return nil
	}
	ids := make([]string, 0, len(set.Keys))
	for _, key := range set.Keys {
		ids = append(ids, key.KeyID)
	}
	return ids
}

func signedTokenKeyID(token string) (string, error) {
	parsed, err := josejwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil || len(parsed.Headers) != 1 {
		return "", errors.New("signed token header unavailable")
	}
	return parsed.Headers[0].KeyID, nil
}

func mutateJWTSignature(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(parts[2]) == 0 {
		return "forged-token"
	}
	index := len(parts[2]) / 2
	changed := byte('A')
	if parts[2][index] == changed {
		changed = 'B'
	}
	parts[2] = parts[2][:index] + string(changed) + parts[2][index+1:]
	return strings.Join(parts, ".")
}

func mutateJWTKeyID(token, keyID string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "unknown-key-token"
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "unknown-key-token"
	}
	var header map[string]any
	if json.Unmarshal(headerBytes, &header) != nil {
		return "unknown-key-token"
	}
	header["kid"] = keyID
	headerBytes, err = json.Marshal(header)
	if err != nil {
		return "unknown-key-token"
	}
	parts[0] = base64.RawURLEncoding.EncodeToString(headerBytes)
	return strings.Join(parts, ".")
}
