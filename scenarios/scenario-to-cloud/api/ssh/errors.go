// DOC: docs/reference/configuration.md#error-categories — error taxonomy and retryability
package ssh

import (
	"errors"
	"fmt"
	"scenario-to-cloud/domain"
	"strings"
)

// Sentinel error categories for errors.Is checks.
var (
	ErrAuth        = errors.New("authentication failed")
	ErrHostKey     = errors.New("host key verification failed")
	ErrTimeout     = errors.New("connection timed out")
	ErrUnreachable = errors.New("host unreachable")
	ErrIPv6        = errors.New("ipv6 connectivity unavailable")
	ErrCommand     = errors.New("command execution failed")
	ErrDiskSpace   = errors.New("no space left on device")
	ErrDNS         = errors.New("DNS resolution failed")
	ErrKeyFormat   = errors.New("SSH key format error")
)

// SSHError carries category, human guidance, and retryability.
type SSHError struct {
	Category  error  // Sentinel above -- enables errors.Is(err, ssh.ErrTimeout)
	Message   string // Human-readable summary
	Hint      string // Actionable recovery suggestion
	Retryable bool   // Can caller retry this?
	ExitCode  int    // Remote exit code (0 if N/A)
	Host      string // Target host (for log correlation)
}

// Error returns the human-readable message.
func (e *SSHError) Error() string { return e.Message }

// Unwrap returns the sentinel category for errors.Is/As.
func (e *SSHError) Unwrap() error { return e.Category }

// ClassifyError analyzes an SSH error string and returns a structured SSHError.
// It replaces the former private classifySSHError function with a public API.
func ClassifyError(errStr, host, defaultHint string) *SSHError {
	errLower := strings.ToLower(errStr)

	switch {
	case strings.Contains(errStr, "Host key verification failed"):
		return &SSHError{
			Category:  ErrHostKey,
			Message:   "Server host key has changed",
			Hint:      "The server's identity has changed since you last connected. This could indicate a server rebuild or a security issue. Remove the old key with: ssh-keygen -R " + host,
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errStr, "Permission denied"):
		return &SSHError{
			Category:  ErrAuth,
			Message:   "SSH authentication failed",
			Hint:      "The server rejected the SSH key. Use 'Copy Key to Server' to add your key, or verify the correct key is selected.",
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errLower, "unable to authenticate"),
		strings.Contains(errLower, "no supported methods remain"):
		return &SSHError{
			Category:  ErrAuth,
			Message:   "Password authentication failed",
			Hint:      "The password may be incorrect, or password authentication may be disabled on the server.",
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errLower, "connection refused"):
		return &SSHError{
			Category:  ErrUnreachable,
			Message:   "SSH connection refused",
			Hint:      "Verify the host and port are correct and that SSH is running on the server.",
			Retryable: true,
			Host:      host,
		}

	case strings.Contains(errLower, "i/o timeout"),
		strings.Contains(errLower, "connection timed out"),
		strings.Contains(errLower, "timed out"):
		if IsIPv6(host) {
			return &SSHError{
				Category:  ErrIPv6,
				Message:   "SSH connection timed out (IPv6)",
				Hint:      IPv6ConnectivityHint,
				Retryable: true,
				Host:      host,
			}
		}
		return &SSHError{
			Category:  ErrTimeout,
			Message:   "SSH connection timed out",
			Hint:      "Check network connectivity and firewall rules.",
			Retryable: true,
			Host:      host,
		}

	case strings.Contains(errLower, "no route to host"),
		strings.Contains(errLower, "network is unreachable"):
		if IsIPv6(host) {
			return &SSHError{
				Category:  ErrIPv6,
				Message:   "IPv6 not available",
				Hint:      IPv6ConnectivityHint,
				Retryable: true,
				Host:      host,
			}
		}
		return &SSHError{
			Category:  ErrUnreachable,
			Message:   "Host unreachable",
			Hint:      "The network path to the host could not be found. Check the IP address and network connectivity.",
			Retryable: true,
			Host:      host,
		}

	case strings.Contains(errLower, "no space left on device"):
		return &SSHError{
			Category:  ErrDiskSpace,
			Message:   "No space left on device",
			Hint:      "The server has run out of disk space. SSH in and run `df -h` to investigate.",
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errLower, "could not resolve hostname"),
		strings.Contains(errLower, "name or service not known"):
		return &SSHError{
			Category:  ErrDNS,
			Message:   "DNS resolution failed",
			Hint:      "DNS lookup failed. Check the hostname or IP address.",
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errLower, "load key") && strings.Contains(errLower, "invalid format"),
		strings.Contains(errLower, "bad permissions"):
		return &SSHError{
			Category:  ErrKeyFormat,
			Message:   "SSH key format error",
			Hint:      "The SSH key file is corrupted or has wrong permissions. Regenerate it or fix permissions with `chmod 600`.",
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errLower, "connection reset"):
		return &SSHError{
			Category:  ErrUnreachable,
			Message:   "SSH connection reset",
			Hint:      "The server closed the connection. This may be temporary (server load, fail2ban), or sshd MaxStartups may be exceeded.",
			Retryable: true,
			Host:      host,
		}

	case strings.Contains(errLower, "broken pipe"):
		return &SSHError{
			Category:  ErrUnreachable,
			Message:   "SSH connection lost",
			Hint:      "The connection was dropped during command execution. Check network stability.",
			Retryable: true,
			Host:      host,
		}

	case strings.Contains(errLower, "too many authentication failures"):
		return &SSHError{
			Category:  ErrAuth,
			Message:   "Too many authentication failures",
			Hint:      "The SSH agent offered too many keys. Try adding IdentitiesOnly=yes to your SSH config, or specify the exact key with -i.",
			Retryable: false,
			Host:      host,
		}

	default:
		return &SSHError{
			Category:  ErrCommand,
			Message:   "SSH connection failed",
			Hint:      defaultHint,
			Retryable: false,
			Host:      host,
		}
	}
}

// StatusFromError maps an SSHError to the corresponding status constant for DTOs.
func StatusFromError(err *SSHError) string {
	if err == nil {
		return StatusSuccess
	}
	switch {
	case errors.Is(err, ErrAuth):
		return StatusAuthFailed
	case errors.Is(err, ErrHostKey):
		return StatusHostKeyChanged
	case errors.Is(err, ErrTimeout):
		return StatusTimeout
	case errors.Is(err, ErrUnreachable):
		return StatusHostUnreachable
	case errors.Is(err, ErrIPv6):
		return StatusIPv6Unavailable
	case errors.Is(err, ErrDiskSpace):
		return StatusDiskFull
	case errors.Is(err, ErrDNS):
		return StatusDNSFailed
	case errors.Is(err, ErrKeyFormat):
		return StatusKeyError
	default:
		return StatusError
	}
}

// newCommandError wraps a command execution failure with output context.
// It attempts to classify the error using stderr + error text. If a specific
// category is matched (not ErrCommand), that category is used; otherwise
// it falls through to the generic ErrCommand behavior.
func newCommandError(err error, res Result, host string) *SSHError {
	// Build combined text for classification
	combined := res.Stderr + " " + err.Error()
	classified := ClassifyError(combined, host, "")

	// Build the hint from output context
	var hint string
	if res.Stderr != "" {
		hint = res.Stderr
	} else if res.Stdout != "" {
		lines := strings.Split(res.Stdout, "\n")
		if len(lines) > 50 {
			lines = lines[len(lines)-50:]
			hint = fmt.Sprintf("(last 50 lines of stdout):\n%s", strings.Join(lines, "\n"))
		} else {
			hint = "stdout: " + res.Stdout
		}
	}
	if hint == "" {
		hint = err.Error()
	}

	// If classification found a specific category, use it but preserve output context
	if !errors.Is(classified, ErrCommand) {
		return &SSHError{
			Category:  classified.Category,
			Message:   classified.Message,
			Hint:      classified.Hint,
			Retryable: classified.Retryable,
			ExitCode:  res.ExitCode,
			Host:      host,
		}
	}

	return &SSHError{
		Category:  ErrCommand,
		Message:   err.Error(),
		Hint:      hint,
		Retryable: false,
		ExitCode:  res.ExitCode,
		Host:      host,
	}
}

// ErrorInfoFromSSHError converts an *SSHError to a *domain.ErrorInfo.
// Returns nil for nil input.
func ErrorInfoFromSSHError(err *SSHError) *domain.ErrorInfo {
	if err == nil {
		return nil
	}
	return &domain.ErrorInfo{
		Message:   err.Message,
		Category:  StatusFromError(err),
		Hint:      err.Hint,
		Retryable: err.Retryable,
		ExitCode:  err.ExitCode,
	}
}
