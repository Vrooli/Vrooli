package onboard

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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

// CredentialStoreEscrowNamespace and CredentialStoreEscrowField address a
// node's store passphrase in the control-plane credential authority. The node
// agent recognises pushed grants by this namespace prefix, so the escrow, the
// grant, and the agent must all use exactly these values.
const (
	CredentialStoreEscrowNamespace = "vrooli-bridge/node-credential-store/"
	CredentialStoreEscrowField     = "passphrase"
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
	Initialized bool            `json:"initialized"`
	Wraps       []nodeStoreWrap `json:"wraps"`
	Entries     int             `json:"entries"`
}

type nodeUnattendedStatus struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	Blocked  string `json:"blocked"`
}

var (
	storeStatusArgs = []string{"credentials", "store", "status", "--format", "json"}
	storeInitArgs   = []string{"credentials", "store", "init", "--format", "json"}
	storeRewrapArgs = []string{"credentials", "store", "rewrap", "--format", "json"}
)

// provisionCredentialStore gives an online node its own encrypted credential
// store with nothing done at the node. Every outcome is recorded; none of them
// fails an otherwise-paired onboarding, because the node is usable and the
// step can converge on the next connect.
func (s *service) provisionCredentialStore(ctx context.Context, opID string, seq *uint64, conn Conn, platform NodePlatform, machineID, nodeID string) {
	if s.storeEscrow == nil {
		return
	}
	if strings.EqualFold(platform.OS, "windows") {
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, "credential store provisioning over SSH is not yet supported on Windows nodes; the node keeps its setup-time store")
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
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusOK,
			"node credential store present and its passphrase is escrowed on this control plane; "+s.convergeNodeUnattended(ctx, conn, platform, secret)+s.ensureStoreGrant(ctx, nodeID, key, secret))
	default:
		s.emit(ctx, opID, seq, StepCredentialStore, StepStatusSkipped, fmt.Sprintf(
			"degraded: the node already has a credential store (wraps: %s) whose passphrase this control plane does not hold; it was left unchanged. It opens only through its own wraps; recovering it needs its creator's passphrase or a recovery bundle",
			wrapProviders(status.Wraps)))
	}
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
