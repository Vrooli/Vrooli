package onboard

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// CredentialStoreEscrow holds each machine's node credential-store passphrase
// on the control plane. The operator directed (2026-09-15) that connecting a
// machine must set it up completely with nothing done at the machine, which
// includes its own credential store: the control plane generates the
// passphrase, keeps it with the machine's record, and supplies it. Production
// stores it in the control-plane credential authority; it never reaches the
// onboarding DB, step events, logs, or argv.
type CredentialStoreEscrow interface {
	// Load returns the escrowed passphrase for key, or found=false when none
	// is held. The caller zeroes the returned slice.
	Load(ctx context.Context, key string) (secret []byte, found bool, err error)
	// Save durably records the passphrase for key before it is used.
	Save(ctx context.Context, key string, secret []byte) error
}

// CredentialStorePendingEscrow is the second slot rotation writes a new
// passphrase to before any node changes. Between "node changed" and "escrow
// promoted" the node opens only with the pending value, so a rotation that is
// interrupted anywhere leaves at least one escrowed value that opens the node.
// The pending slot has no grant, so it is never pushed to a node.
type CredentialStorePendingEscrow interface {
	LoadPending(ctx context.Context, key string) (secret []byte, found bool, err error)
	SavePending(ctx context.Context, key string, secret []byte) error
	ClearPending(ctx context.Context, key string) error
}

// CredentialStoreEscrowNamespace and CredentialStoreEscrowField address a
// node's store passphrase in the control-plane credential authority. The node
// agent recognises pushed grants by this namespace prefix, so the escrow, the
// grant, and the agent must all use exactly these values.
const (
	CredentialStoreEscrowNamespace    = "vrooli-bridge/node-credential-store/"
	CredentialStoreEscrowField        = "passphrase"
	CredentialStoreEscrowPendingField = "passphrase-next"
)

// CredentialStoreLogicalID is the credential-authority address for key.
func CredentialStoreLogicalID(key string) string {
	return CredentialStoreEscrowNamespace + key
}

// NodeStoreGrantEnsurer makes sure a node holds an ephemeral infrastructure
// grant for its own store passphrase, so Bridge re-pushes it sealed on every
// reconnect and the node agent can open the store after a reboot with nobody
// at the machine.
type NodeStoreGrantEnsurer interface {
	EnsureNodeStoreGrant(ctx context.Context, nodeID, logicalID, field string) error
}

// WithCredentialStoreEscrow enables the credential-store step. Without it the
// step is not run and nodes keep the setup-time behaviour.
func WithCredentialStoreEscrow(escrow CredentialStoreEscrow) Option {
	return func(s *service) { s.storeEscrow = escrow }
}

// WithNodeStoreGrant supplies the grant ensurer run once a node's store
// passphrase is escrowed.
func WithNodeStoreGrant(ensurer NodeStoreGrantEnsurer) Option {
	return func(s *service) { s.storeGrant = ensurer }
}

// ensureStoreGrant records the grant outcome as a clause for the step detail.
func (s *service) ensureStoreGrant(ctx context.Context, nodeID, key string, secret []byte) string {
	if s.storeGrant == nil {
		return ""
	}
	if err := s.storeGrant.EnsureNodeStoreGrant(ctx, nodeID, CredentialStoreLogicalID(key), CredentialStoreEscrowField); err != nil {
		return "; the node's unlock grant could not be ensured (" + redact(err.Error(), secret) + "), so it cannot reopen the store after a reboot until the next connect"
	}
	return "; the node holds an unlock grant, so Bridge reopens the store after a reboot"
}

const storePassphraseBytes = 32

// generateStorePassphrase returns a fresh random passphrase as base64url text.
func generateStorePassphrase() ([]byte, error) {
	raw := make([]byte, storePassphraseBytes)
	defer zeroBytes(raw)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generate credential store passphrase: %w", err)
	}
	out := make([]byte, base64.RawURLEncoding.EncodedLen(len(raw)))
	base64.RawURLEncoding.Encode(out, raw)
	return out, nil
}

// credentialStoreEscrowKey names the machine the passphrase belongs to. The
// durable machine identity is preferred; a direct node onboarding without one
// falls back to its node id.
func credentialStoreEscrowKey(machineID, nodeID string) string {
	if id := strings.ToLower(strings.TrimSpace(machineID)); id != "" {
		return id
	}
	return "node-" + strings.ToLower(strings.TrimSpace(nodeID))
}

type nodeStoreWrap struct {
	Provider string `json:"provider"`
}

type nodeStoreStatus struct {
	Initialized bool                 `json:"initialized"`
	Unlocked    bool                 `json:"unlocked"`
	ActiveWrap  string               `json:"active_wrap"`
	Wraps       []nodeStoreWrap      `json:"wraps"`
	Entries     int                  `json:"entries"`
	Unattended  nodeUnattendedStatus `json:"unattended"`
}

// hasWrap reports whether the store has a wrap from provider.
func (status nodeStoreStatus) hasWrap(provider string) bool {
	for _, wrap := range status.Wraps {
		if strings.TrimSpace(wrap.Provider) == provider {
			return true
		}
	}
	return false
}

// nodePassphraseWrapProvider is the node store's provider name for the
// passphrase wrap (securestore providerPassphrase).
const nodePassphraseWrapProvider = "passphrase"

type nodeUnattendedStatus struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	Blocked  string `json:"blocked"`
}

var (
	storeStatusArgs = []string{"credentials", "store", "status", "--format", "json"}
	storeInitArgs   = []string{"credentials", "store", "init", "--format", "json"}
	storeRewrapArgs = []string{"credentials", "store", "rewrap", "--format", "json"}
	// add-passphrase and verify-passphrase exist on nodes whose control plane
	// includes them (2026-09-15); an older node answers with a usage error,
	// which the callers report as "update the node" rather than a failure.
	storeAddPassphraseArgs    = []string{"credentials", "store", "add-passphrase", "--format", "json"}
	storeVerifyPassphraseArgs = []string{"credentials", "store", "verify-passphrase", "--format", "json"}
	storeChangePassphraseArgs = []string{"credentials", "store", "change-passphrase"}
)

// windowsCredentialStoreSkip is the operator-visible reason Windows nodes keep
// their setup-time store. `vrooli setup` on Windows protects the store with a
// DPAPI wrap bound to the node user, but Bridge's node-CLI path runs a POSIX
// shell command over SSH, so no recovery passphrase is escrowed for Windows
// nodes yet. No Windows node has validated either path.
const windowsCredentialStoreSkip = "Windows node: its credential store stays as `vrooli setup` created it (DPAPI, bound to the node user). Bridge does not yet escrow a recovery passphrase for Windows nodes because it reaches the node CLI with a POSIX shell command; keep a recovery bundle (`vrooli credentials recovery export`) for this node"

// provisionCredentialStore gives an online node its own encrypted credential
// store with nothing done at the node. Every outcome is recorded; none of them
// fails an otherwise-paired onboarding, because the node is usable and the
// step can converge on the next connect.
func (s *service) provisionCredentialStore(ctx context.Context, opID string, seq *uint64, conn Conn, platform NodePlatform, machineID, nodeID string) {
	if s.storeEscrow == nil {
		return
	}
	if strings.EqualFold(platform.OS, "windows") {
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, windowsCredentialStoreSkip)
		return
	}
	s.emit(ctx, opID, seq, StepCredentialStore, StepStatusStarted, "checking the node's encrypted credential store")

	status, err := s.readNodeStoreStatus(ctx, conn, platform)
	if err != nil {
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, "degraded: could not read the node's credential store status: "+err.Error())
		return
	}
	key := credentialStoreEscrowKey(machineID, nodeID)
	secret, found, err := s.storeEscrow.Load(ctx, key)
	defer func() { zeroBytes(secret) }()
	if err != nil {
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, "degraded: this control plane could not read its escrowed credential-store passphrases: "+redact(err.Error(), secret))
		return
	}

	switch {
	case !status.Initialized:
		if !found {
			// Escrow first: a crash after init must never leave a store whose
			// only passphrase existed in this process.
			generated, genErr := generateStorePassphrase()
			if genErr != nil {
				s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, "degraded: "+genErr.Error())
				return
			}
			secret = generated
			if saveErr := s.storeEscrow.Save(ctx, key, secret); saveErr != nil {
				s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped,
					"degraded: the passphrase could not be escrowed on this control plane, so the node's credential store was left uninitialized: "+redact(saveErr.Error(), secret))
				return
			}
		}
		res, runErr := s.driver.RunNodeCLI(ctx, conn, platform, storeInitArgs, stdinLine(secret))
		if runErr != nil || res.ExitCode != 0 {
			s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped,
				"degraded: creating the node's credential store failed: "+redact(commandFailure(res, runErr), secret))
			return
		}
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusOK,
			"created the node's encrypted credential store; its passphrase is escrowed on this control plane; "+s.convergeNodeUnattended(ctx, conn, platform, secret)+s.ensureStoreGrant(ctx, nodeID, key, secret))
	case found:
		secret, detail, ok := s.convergeEscrowedStore(ctx, conn, platform, key, status, secret)
		defer zeroBytes(secret)
		if !ok {
			s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, "degraded: "+detail)
			return
		}
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusOK,
			detail+s.convergeNodeUnattended(ctx, conn, platform, secret)+s.ensureStoreGrant(ctx, nodeID, key, secret))
	case !status.hasWrap(nodePassphraseWrapProvider):
		// A store `vrooli setup` created behind an unattended wrap alone (TPM,
		// host key, Keychain). Losing that binding would lose every value, so
		// Bridge adds an escrowed recovery passphrase wrap beside it. Escrow
		// first: a crash after the node change must never leave a passphrase
		// only this process knew. A crash before it leaves an unused escrowed
		// value, which the found branch above converges on the next connect.
		generated, genErr := generateStorePassphrase()
		if genErr != nil {
			s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, "degraded: "+genErr.Error())
			return
		}
		secret = generated
		if saveErr := s.storeEscrow.Save(ctx, key, secret); saveErr != nil {
			s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped,
				"degraded: the recovery passphrase could not be escrowed on this control plane, so the node's store was left unchanged: "+redact(saveErr.Error(), secret))
			return
		}
		if addErr := s.addNodePassphraseWrap(ctx, conn, platform, secret); addErr != nil {
			s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, "degraded: "+redact(addErr.Error(), secret))
			return
		}
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusOK, fmt.Sprintf(
			"added a recovery passphrase wrap, escrowed on this control plane, beside the store's existing wraps (%s), which are unchanged; ",
			wrapProviders(status.Wraps))+s.convergeNodeUnattended(ctx, conn, platform, secret)+s.ensureStoreGrant(ctx, nodeID, key, secret))
	default:
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, fmt.Sprintf(
			"degraded: the node already has a credential store (wraps: %s) whose passphrase this control plane does not hold; it was left unchanged. It opens only through its own wraps; recovering it needs its creator's passphrase or a recovery bundle",
			wrapProviders(status.Wraps)))
	}
}

// convergeEscrowedStore brings a node store whose passphrase this control
// plane holds to the escrowed state: an interrupted rotation is finished or
// discarded, a store with no passphrase wrap gains the escrowed one, and a
// store with one must open with it. It returns the passphrase that opens the
// node (the caller zeroes it), the step detail prefix, and false when the node
// cannot be converged without a human decision.
func (s *service) convergeEscrowedStore(ctx context.Context, conn Conn, platform NodePlatform, key string, status nodeStoreStatus, secret []byte) ([]byte, string, bool) {
	if !status.hasWrap(nodePassphraseWrapProvider) {
		if err := s.addNodePassphraseWrap(ctx, conn, platform, secret); err != nil {
			return secret, redact(err.Error(), secret), false
		}
		return secret, "added the escrowed recovery passphrase wrap beside the store's existing wraps (" + wrapProviders(status.Wraps) + "), which are unchanged; ", true
	}
	effective, resumed, err := s.reconcilePendingEscrow(ctx, conn, platform, key, secret)
	if err != nil {
		return effective, redact(err.Error(), effective), false
	}
	note := "node credential store present and its passphrase is escrowed on this control plane; "
	if resumed {
		note = "finished an interrupted passphrase rotation (the node already used the new passphrase; it is now the escrowed one); "
	}
	check, err := s.verifyNodePassphrase(ctx, conn, platform, effective)
	switch {
	case errors.Is(err, errNodeCLIPredates):
		return effective, note + "the node's CLI cannot verify passphrases yet, so the escrowed value was not re-checked; ", true
	case err != nil:
		return effective, "could not verify the escrowed passphrase on the node: " + redact(err.Error(), effective), false
	case !check:
		return effective, "the escrowed passphrase does not open this node's passphrase wrap; the store was left unchanged and no unlock grant was pushed. Rotate it from a passphrase that opens it, or restore the node's store from a recovery bundle", false
	}
	return effective, note, true
}

// reconcilePendingEscrow finishes or discards a rotation that stopped between
// its escrow writes. The pending value is promoted only when it is the one
// that opens the node; otherwise it is dropped. It returns the passphrase that
// now matches the escrow and whether a rotation was finished.
func (s *service) reconcilePendingEscrow(ctx context.Context, conn Conn, platform NodePlatform, key string, current []byte) ([]byte, bool, error) {
	pendingEscrow, ok := s.storeEscrow.(CredentialStorePendingEscrow)
	if !ok {
		return current, false, nil
	}
	pending, found, err := pendingEscrow.LoadPending(ctx, key)
	if err != nil {
		return current, false, fmt.Errorf("read the pending rotation passphrase: %w", err)
	}
	if !found {
		return current, false, nil
	}
	opens, err := s.verifyNodePassphrase(ctx, conn, platform, pending)
	if err != nil {
		zeroBytes(pending)
		return current, false, fmt.Errorf("check an interrupted rotation on the node: %w", err)
	}
	if !opens {
		zeroBytes(pending)
		if clearErr := pendingEscrow.ClearPending(ctx, key); clearErr != nil {
			return current, false, fmt.Errorf("discard an interrupted rotation that never reached the node: %w", clearErr)
		}
		return current, false, nil
	}
	if saveErr := s.storeEscrow.Save(ctx, key, pending); saveErr != nil {
		zeroBytes(pending)
		return current, false, fmt.Errorf("promote the passphrase an interrupted rotation left on the node: %w", saveErr)
	}
	// A failed clear leaves pending equal to the escrow, which the next
	// reconcile promotes again harmlessly.
	_ = pendingEscrow.ClearPending(ctx, key)
	zeroBytes(current)
	return pending, true, nil
}

// errNodeCLIPredates marks a node whose installed vrooli CLI has no
// add-passphrase/verify-passphrase verb yet.
var errNodeCLIPredates = errors.New("the node's vrooli CLI predates this credential-store command; update the node, then retry")

// verifyNodePassphrase asks the node whether secret opens its store's
// passphrase wrap, consulting no other wrap.
func (s *service) verifyNodePassphrase(ctx context.Context, conn Conn, platform NodePlatform, secret []byte) (bool, error) {
	res, err := s.driver.RunNodeCLI(ctx, conn, platform, storeVerifyPassphraseArgs, stdinLine(secret))
	if err != nil {
		return false, err
	}
	// verify-passphrase prints its answer before choosing an exit code, so a
	// wrong passphrase is JSON with valid=false and a non-zero exit.
	var check struct {
		Valid *bool `json:"valid"`
	}
	if json.Unmarshal(firstJSONObject(res.Stdout), &check) == nil && check.Valid != nil {
		return *check.Valid, nil
	}
	if res.ExitCode != 0 && nodeCLIUnknownCommand(res) {
		return false, errNodeCLIPredates
	}
	return false, fmt.Errorf("%s", commandFailure(res, nil))
}

// addNodePassphraseWrap adds secret as the store's passphrase wrap, beside
// its unattended wraps.
func (s *service) addNodePassphraseWrap(ctx context.Context, conn Conn, platform NodePlatform, secret []byte) error {
	res, err := s.driver.RunNodeCLI(ctx, conn, platform, storeAddPassphraseArgs, stdinLine(secret))
	if err == nil && res.ExitCode == 0 {
		return nil
	}
	if err == nil && nodeCLIUnknownCommand(res) {
		return fmt.Errorf("adding the escrowed recovery passphrase wrap: %w", errNodeCLIPredates)
	}
	return fmt.Errorf("adding the escrowed recovery passphrase wrap failed: %s", commandFailure(res, err))
}

// nodeCLIUnknownCommand recognises the usage error an older node CLI prints
// for a verb it does not have.
func nodeCLIUnknownCommand(res NodeCommandResult) bool {
	text := strings.ToLower(res.Stderr + "\n" + res.Stdout)
	return strings.Contains(text, "unknown command") || strings.Contains(text, "unknown subcommand") || strings.Contains(text, "no such command")
}

func (s *service) readNodeStoreStatus(ctx context.Context, conn Conn, platform NodePlatform) (nodeStoreStatus, error) {
	res, err := s.driver.RunNodeCLI(ctx, conn, platform, storeStatusArgs, nil)
	if err != nil || res.ExitCode != 0 {
		return nodeStoreStatus{}, fmt.Errorf("%s", commandFailure(res, err))
	}
	var status nodeStoreStatus
	if decodeErr := json.Unmarshal(firstJSONObject(res.Stdout), &status); decodeErr != nil {
		return nodeStoreStatus{}, fmt.Errorf("status output was not JSON: %v", decodeErr)
	}
	return status, nil
}

// convergeNodeUnattended asks the node to add or repair a wrap that opens the
// store at boot with no one present, and says whether it could.
func (s *service) convergeNodeUnattended(ctx context.Context, conn Conn, platform NodePlatform, secret []byte) string {
	res, err := s.driver.RunNodeCLI(ctx, conn, platform, storeRewrapArgs, stdinLine(secret))
	var unattended nodeUnattendedStatus
	// rewrap prints its status before choosing an exit code, so a blocked host
	// still yields the reason; that is degraded, not a failure.
	if err == nil && json.Unmarshal(firstJSONObject(res.Stdout), &unattended) == nil {
		if unattended.Enabled {
			return "opens unattended at boot via " + nonEmpty(unattended.Provider, "an unattended wrap")
		}
		return "unattended boot is blocked on this host (" + redact(nonEmpty(unattended.Blocked, "no unattended wrap is available"), secret) + "); a reboot needs the escrowed passphrase"
	}
	return "unattended access could not be evaluated: " + redact(commandFailure(res, err), secret)
}

// stdinLine returns secret followed by a newline in a fresh buffer; the node
// CLI reads one line. The driver zeroes its own copy.
func stdinLine(secret []byte) []byte {
	out := make([]byte, 0, len(secret)+1)
	out = append(out, secret...)
	return append(out, '\n')
}

func commandFailure(res NodeCommandResult, err error) string {
	if err != nil {
		return err.Error()
	}
	detail := strings.TrimSpace(res.Stderr)
	if detail == "" {
		detail = strings.TrimSpace(res.Stdout)
	}
	if len(detail) > 300 {
		detail = "…" + detail[len(detail)-300:]
	}
	if detail == "" {
		return fmt.Sprintf("exit %d", res.ExitCode)
	}
	return fmt.Sprintf("exit %d: %s", res.ExitCode, detail)
}

// redact removes any occurrence of secret from text destined for a step event.
// The node never prints the passphrase; this is the belt to that brace.
func redact(text string, secret []byte) string {
	if len(secret) == 0 {
		return text
	}
	return strings.ReplaceAll(text, string(secret), "[redacted]")
}

// firstJSONObject returns the first top-level JSON object in output, skipping
// any leading log lines the CLI may have written to stdout.
func firstJSONObject(output string) []byte {
	data := []byte(output)
	if start := bytes.IndexByte(data, '{'); start >= 0 {
		return data[start:]
	}
	return data
}

func wrapProviders(wraps []nodeStoreWrap) string {
	names := make([]string, 0, len(wraps))
	for _, wrap := range wraps {
		if name := strings.TrimSpace(wrap.Provider); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "none reported"
	}
	return strings.Join(names, ", ")
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
