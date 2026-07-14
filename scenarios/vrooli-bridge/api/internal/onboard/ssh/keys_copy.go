package ssh

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// KeyCopier installs an SSH public key on a remote host using the owner's
// password for the one-time authentication.
type KeyCopier interface {
	CopyKey(ctx context.Context, req CopyKeyRequest) CopyKeyResponse
}

// ExecKeyCopier installs the key over an x/crypto/ssh password session and
// appends it to the remote authorized_keys (idempotent: a key already present
// is reported as already-exists, not re-added).
type ExecKeyCopier struct{}

var _ KeyCopier = ExecKeyCopier{}

// CopyKey copies the public key to the host using password authentication.
func (ExecKeyCopier) CopyKey(ctx context.Context, req CopyKeyRequest) CopyKeyResponse {
	timestamp := nowTimestamp()

	cfg := NewConfig(req.Host, req.Port, req.User, req.KeyPath, req.KnownHostsFile)

	pkPath := pubKeyPath(cfg.KeyPath)
	pubKeyContent, err := os.ReadFile(pkPath)
	if err != nil {
		return copyErr(StatusError, "Cannot read public key", err.Error(), timestamp)
	}
	pubKeyLine := strings.TrimSpace(string(pubKeyContent))
	pubKeyParts := strings.Fields(pubKeyLine)
	if len(pubKeyParts) < 2 {
		return copyErr(StatusInvalidInput, "Invalid public key format",
			"The public key file does not contain a valid SSH public key", timestamp)
	}

	hostKeyCallback, err := newTOFUHostKeyCallback(cfg.Host, cfg.Port, cfg.KnownHostsFile)
	if err != nil {
		return copyErr(StatusError, "Failed to initialize host key verification", err.Error(), timestamp)
	}

	config := &gossh.ClientConfig{
		User:            cfg.User,
		Auth:            []gossh.AuthMethod{gossh.Password(req.Password)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port))
	client, err := dialContext(ctx, addr, config)
	if err != nil {
		classified := ClassifyError(err.Error(), cfg.Host, err.Error())
		return CopyKeyResponse{Outcome: Outcome{
			OK:        false,
			Status:    StatusFromError(classified),
			Message:   classified.Message,
			Hint:      classified.Hint,
			Timestamp: timestamp,
		}}
	}
	defer client.Close()

	// Idempotency: skip the append if the key is already authorized.
	keyData := pubKeyParts[1]
	checkSession, err := client.NewSession()
	if err != nil {
		return copyErr(StatusError, "Failed to create SSH session", err.Error(), timestamp)
	}
	checkCmd := fmt.Sprintf("grep -qF %s ~/.ssh/authorized_keys 2>/dev/null", quoteSingle(keyData))
	checkErr := checkSession.Run(checkCmd)
	checkSession.Close()

	if checkErr == nil {
		slog.Info("ssh.key_copied", "host", cfg.Host, "status", StatusAlreadyExists)
		return CopyKeyResponse{
			Outcome: Outcome{
				OK:        true,
				Status:    StatusAlreadyExists,
				Message:   "SSH key is already authorized on the host",
				Timestamp: timestamp,
			},
			AlreadyExists: true,
		}
	}

	addSession, err := client.NewSession()
	if err != nil {
		return copyErr(StatusError, "Failed to create SSH session", err.Error(), timestamp)
	}
	defer addSession.Close()

	// Create ~/.ssh, append the key on its own line (guarding against a missing
	// trailing newline that would concatenate two keys), and tighten perms.
	addCmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && { [ ! -f ~/.ssh/authorized_keys ] || [ -z "$(tail -c1 ~/.ssh/authorized_keys)" ] || echo ''; printf '%%s\n' %s; } >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`,
		quoteSingle(pubKeyLine),
	)
	if err := addSession.Run(addCmd); err != nil {
		return copyErr(StatusError, "Failed to add key to authorized_keys", err.Error(), timestamp)
	}

	slog.Info("ssh.key_copied", "host", cfg.Host, "status", StatusSuccess)
	return CopyKeyResponse{
		Outcome: Outcome{
			OK:        true,
			Status:    StatusSuccess,
			Message:   "SSH key successfully installed on the host",
			Timestamp: timestamp,
		},
		KeyCopied: true,
	}
}

// dialContext dials the address honoring ctx cancellation/deadline.
func dialContext(ctx context.Context, addr string, config *gossh.ClientConfig) (*gossh.Client, error) {
	d := net.Dialer{Timeout: config.Timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := gossh.NewClientConn(conn, addr, config)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return gossh.NewClient(c, chans, reqs), nil
}

func copyErr(status, message, hint, timestamp string) CopyKeyResponse {
	return CopyKeyResponse{Outcome: Outcome{
		OK:        false,
		Status:    status,
		Message:   message,
		Hint:      hint,
		Timestamp: timestamp,
	}}
}
