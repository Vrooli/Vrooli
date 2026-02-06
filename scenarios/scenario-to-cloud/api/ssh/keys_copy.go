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
	"scenario-to-cloud/internal/shellutil"
)

// KeyCopier copies an SSH public key to a remote server.
type KeyCopier interface {
	CopyKey(ctx context.Context, req CopyKeyRequest) CopyKeyResponse
}

// ExecKeyCopier implements KeyCopier using golang.org/x/crypto/ssh directly.
type ExecKeyCopier struct{}

var _ KeyCopier = ExecKeyCopier{}

// CopyKey copies the public key to the server using password authentication.
func (ExecKeyCopier) CopyKey(ctx context.Context, req CopyKeyRequest) CopyKeyResponse {
	timestamp := nowTimestamp()

	cfg := NewConfig(req.Host, req.Port, req.User, ExpandPath(req.KeyPath))

	// Validate key path
	if err := ValidateSSHPath(cfg.KeyPath); err != nil {
		return CopyKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusError,
				Message:   "Invalid SSH key path",
				Hint:      err.Error(),
				Timestamp: timestamp,
			},
		}
	}

	// Read public key
	pkPath := pubKeyPath(cfg.KeyPath)

	pubKeyContent, err := os.ReadFile(pkPath)
	if err != nil {
		return CopyKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusError,
				Message:   "Cannot read public key",
				Hint:      err.Error(),
				Timestamp: timestamp,
			},
		}
	}
	pubKeyLine := strings.TrimSpace(string(pubKeyContent))

	// Validate public key format
	pubKeyParts := strings.Fields(pubKeyLine)
	if len(pubKeyParts) < 2 {
		return CopyKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusInvalidInput,
				Message:   "Invalid public key format",
				Hint:      "The public key file does not contain a valid SSH public key",
				Timestamp: timestamp,
			},
		}
	}

	// Connect using password authentication
	config := &gossh.ClientConfig{
		User: cfg.User,
		Auth: []gossh.AuthMethod{
			gossh.Password(req.Password),
		},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), // TOFU - Trust On First Use
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port))
	client, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		classified := ClassifyError(err.Error(), cfg.Host, err.Error())
		return CopyKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusFromError(classified),
				Message:   classified.Message,
				Hint:      classified.Hint,
				Timestamp: timestamp,
			},
		}
	}
	defer client.Close()

	// Check if key already exists in authorized_keys
	keyData := pubKeyParts[1] // The base64-encoded key data
	checkSession, err := client.NewSession()
	if err != nil {
		return CopyKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusError,
				Message:   "Failed to create SSH session",
				Hint:      err.Error(),
				Timestamp: timestamp,
			},
		}
	}

	// Use grep to check if key exists
	checkCmd := fmt.Sprintf("grep -qF %s ~/.ssh/authorized_keys 2>/dev/null", shellutil.QuoteSingle(keyData))
	checkErr := checkSession.Run(checkCmd)
	checkSession.Close()

	if checkErr == nil {
		// Key already exists
		slog.Info("ssh.key_copied", "host", cfg.Host, "status", StatusAlreadyExists)
		return CopyKeyResponse{
			Outcome: Outcome{
				OK:        true,
				Status:    StatusAlreadyExists,
				Message:   "SSH key is already authorized on the server",
				Timestamp: timestamp,
			},
			KeyCopied:     false,
			AlreadyExists: true,
		}
	}

	// Key doesn't exist, add it
	addSession, err := client.NewSession()
	if err != nil {
		return CopyKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusError,
				Message:   "Failed to create SSH session",
				Hint:      err.Error(),
				Timestamp: timestamp,
			},
		}
	}
	defer addSession.Close()

	// Create .ssh directory, add key, set permissions
	// The complex condition ensures we add a newline before the key if the file
	// exists but doesn't end with a newline (which would corrupt both keys)
	addCmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && { [ ! -f ~/.ssh/authorized_keys ] || [ -z "$(tail -c1 ~/.ssh/authorized_keys)" ] || echo ''; printf '%%s\n' %s; } >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`,
		shellutil.QuoteSingle(pubKeyLine),
	)

	if err := addSession.Run(addCmd); err != nil {
		return CopyKeyResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusError,
				Message:   "Failed to add key to authorized_keys",
				Hint:      err.Error(),
				Timestamp: timestamp,
			},
		}
	}

	slog.Info("ssh.key_copied", "host", cfg.Host, "status", StatusSuccess)

	return CopyKeyResponse{
		Outcome: Outcome{
			OK:        true,
			Status:    StatusSuccess,
			Message:   "SSH key successfully copied to server",
			Timestamp: timestamp,
		},
		KeyCopied:     true,
		AlreadyExists: false,
	}
}
