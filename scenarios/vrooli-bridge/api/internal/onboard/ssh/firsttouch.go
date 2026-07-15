package ssh

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultKeyName is the basename of the onboarding keypair when the caller does
// not specify one.
const defaultKeyName = "bridge-onboard"

// FirstTouchRequest is the owner-supplied input to a managed first touch.
//
// Password is the transient owner credential. It is held in memory only for the
// single key-copy dial and zeroed by FirstTouch before returning — pass an
// owned, mutable slice you do not reuse (FirstTouch takes ownership of it). It
// is never written to disk, logs, or the database.
type FirstTouchRequest struct {
	Host     string
	Port     int
	User     string
	Password []byte
	KeyName  string

	// ProvisionSudo, when true, installs a scoped passwordless-sudo drop-in for
	// User over the freshly key-authenticated connection — while the password is
	// still held — so every later privileged step works over non-interactive SSH.
	// Declining (or a provisioning failure) never fails the first touch itself.
	ProvisionSudo bool
}

// FirstTouchResult reports the outcome of a managed first touch. It carries no
// credential material — only the durable, shareable facts (key path, public
// key, host fingerprint) later phases record against the node.
type FirstTouchResult struct {
	OK                  bool   `json:"ok"`
	Host                string `json:"host"`
	User                string `json:"user"`
	Port                int    `json:"port"`
	KeyPath             string `json:"key_path"`
	PublicKey           string `json:"public_key,omitempty"`
	Fingerprint         string `json:"fingerprint,omitempty"`
	KeyGenerated        bool   `json:"key_generated"`
	AlreadyPasswordless bool   `json:"already_passwordless"`
	CopyKeyAttempted    bool   `json:"copy_key_attempted"`
	ConnectionVerified  bool   `json:"connection_verified"`
	Status              string `json:"status"`
	Message             string `json:"message,omitempty"`
	// Hint carries the raw, non-sensitive underlying cause on a failure (e.g. the
	// x/crypto dial error) so the operator sees an actionable reason instead of the
	// generic category alone. Never holds credential material.
	Hint string `json:"hint,omitempty"`

	// SudoProvisioned is true only when the passwordless-sudo drop-in was written
	// and verified this run. SudoState carries the full outcome (including the
	// no-op, declined, and password-unavailable cases) for the op step detail.
	SudoProvisioned bool      `json:"sudo_provisioned"`
	SudoState       SudoState `json:"sudo_state,omitempty"`
}

// FirstTouch establishes working passwordless SSH to a single owner-supplied
// host, mirroring scenario-to-cloud's bootstrap flow:
//
//	generate keypair if absent → test passwordless → (if needed) copy key using
//	the password → retest passwordless.
//
// The flow is idempotent: on a host where the key is already authorized the
// initial test succeeds and no password is required, and a re-run with an
// already-present key installs nothing (already_exists).
//
// When ProvisionSudo is set, the password is held one step longer — through the
// optional passwordless-sudo drop-in install over the now-key-authenticated
// connection — and only then zeroed. The password slice is always zeroed before
// returning (deferred), including on every early-return and error path.
func (s *Service) FirstTouch(ctx context.Context, req FirstTouchRequest) (FirstTouchResult, error) {
	// Zero the credential no matter which path we exit through.
	defer zeroBytes(req.Password)

	host := strings.TrimSpace(req.Host)
	if host == "" {
		return FirstTouchResult{}, errors.New("first touch: host is required")
	}
	user := strings.TrimSpace(req.User)
	if user == "" {
		user = DefaultUser
	}
	port := req.Port
	if port == 0 {
		port = DefaultPort
	}
	keyName := strings.TrimSpace(req.KeyName)
	if keyName == "" {
		keyName = defaultKeyName
	}
	if err := ValidateKeyFilename(keyName); err != nil {
		return FirstTouchResult{}, fmt.Errorf("first touch: invalid key name: %w", err)
	}

	if err := ensureDir0700(s.stateDir); err != nil {
		return FirstTouchResult{}, fmt.Errorf("first touch: create state dir: %w", err)
	}

	keyPath := filepath.Join(s.stateDir, keyName)
	if err := s.validateKeyPath(keyPath); err != nil {
		return FirstTouchResult{}, fmt.Errorf("first touch: %w", err)
	}

	result := FirstTouchResult{Host: host, User: user, Port: port, KeyPath: keyPath}

	// 1. Generate the keypair if absent (idempotent across re-runs).
	if _, err := os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		if _, gErr := s.GenerateKey(GenerateKeyRequest{
			Type:     KeyTypeEd25519,
			Filename: keyName,
			Comment:  "vrooli-bridge-onboard",
		}); gErr != nil {
			return result, fmt.Errorf("first touch: generate key: %w", gErr)
		}
		result.KeyGenerated = true
	} else if err != nil {
		return result, fmt.Errorf("first touch: stat key: %w", err)
	}

	if pub, fp, err := s.ReadPublicKey(keyPath); err == nil {
		result.PublicKey = pub
		result.Fingerprint = fp
	}

	testReq := TestConnectionRequest{Host: host, Port: port, User: user, KeyPath: keyPath}

	// 2. Initial passwordless test — already-working hosts short-circuit here.
	initial := s.TestConnection(ctx, testReq)
	if initial.OK {
		result.OK = true
		result.AlreadyPasswordless = true
		result.ConnectionVerified = true
		result.Status = StatusSuccess
		result.Message = "passwordless SSH already working"
		// The key already authorizes, but the passwordless-sudo drop-in may not be
		// installed yet. Provision it if asked — on this path the owner password may
		// be absent (a pure re-run), which the provisioner reports as
		// password-unavailable rather than a failure.
		s.provisionSudo(ctx, req, testReq, &result)
		return result, nil
	}

	// 3. Install the key using the owner password.
	if len(req.Password) == 0 {
		result.Status = StatusAuthFailed
		result.Message = "passwordless SSH is not established and no password was supplied for first touch"
		return result, nil
	}
	result.CopyKeyAttempted = true

	copyResp := s.copier.CopyKey(ctx, CopyKeyRequest{
		Host:           host,
		Port:           port,
		User:           user,
		KeyPath:        keyPath,
		KnownHostsFile: s.knownHostsPath(),
		Password:       string(req.Password),
	})
	// NB: the password is NOT zeroed here — when ProvisionSudo is set it is needed
	// once more for the sudo drop-in below. The deferred zero (and the eager wipe
	// after provisioning) guarantee it never outlives this function.

	if !copyResp.OK {
		result.Status = copyResp.Status
		result.Message = copyResp.Message
		result.Hint = copyResp.Hint
		return result, nil
	}

	// 4. Retest passwordless to prove the installed key works.
	final := s.TestConnection(ctx, testReq)
	result.ConnectionVerified = final.OK
	result.OK = final.OK
	if final.OK {
		result.Status = StatusSuccess
		result.Message = "passwordless SSH established"
		// 5. Optional passwordless-sudo provisioning while the password is still in
		//    hand, over the connection the just-installed key now authorizes.
		s.provisionSudo(ctx, req, testReq, &result)
	} else {
		result.Status = final.Status
		result.Message = fmt.Sprintf("key installed but passwordless SSH still failed: %s", final.Message)
		result.Hint = final.Hint
	}
	return result, nil
}

// provisionSudo runs (or declines) the passwordless-sudo drop-in install for a
// successful first touch, recording the outcome on result. It is the single place
// the password's hold window is extended and then eagerly wiped: the sudo write
// is its last consumer, so we zero it here (the deferred FirstTouch zero remains
// the backstop on every other path).
func (s *Service) provisionSudo(ctx context.Context, req FirstTouchRequest, testReq TestConnectionRequest, result *FirstTouchResult) {
	if !req.ProvisionSudo {
		result.SudoState = SudoStateDeclined
		return
	}
	out := s.provisioner.Provision(ctx, ProvisionSudoRequest{
		Host:           testReq.Host,
		Port:           testReq.Port,
		User:           testReq.User,
		KeyPath:        testReq.KeyPath,
		KnownHostsFile: s.knownHostsPath(),
		Password:       req.Password,
	})
	zeroBytes(req.Password)
	result.SudoState = out.State
	result.SudoProvisioned = out.State == SudoStateProvisioned
}

// zeroBytes overwrites b with zeros. Go strings are immutable and cannot be
// wiped, which is exactly why FirstTouch keeps the credential in a []byte for as
// long as it controls it and only briefly materializes a string for the ssh
// password auth call.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
