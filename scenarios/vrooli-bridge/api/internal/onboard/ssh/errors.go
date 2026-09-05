package ssh

import (
	"errors"
	"fmt"
	"net"
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
	Category  error  // Sentinel above — enables errors.Is(err, ssh.ErrTimeout)
	Message   string // Human-readable summary
	Hint      string // Actionable recovery suggestion
	Retryable bool   // Can the caller retry this?
	ExitCode  int    // Remote exit code (0 if N/A)
	Host      string // Target host (for log correlation)
}

// Error returns the human-readable message.
func (e *SSHError) Error() string { return e.Message }

// Unwrap returns the sentinel category for errors.Is/As.
func (e *SSHError) Unwrap() error { return e.Category }

// IsIPv6 reports whether host is an IPv6 address literal.
func IsIPv6(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.To4() == nil
}

// IPv6ConnectivityHint explains a likely-missing IPv6 path to the operator.
const IPv6ConnectivityHint = "You entered an IPv6 address, but your network may not have IPv6 connectivity. Try the IPv4 address of the host instead."

// ClassifyError analyzes an SSH error string and returns a structured SSHError.
func ClassifyError(errStr, host, defaultHint string) *SSHError {
	errLower := strings.ToLower(errStr)

	switch {
	case strings.Contains(errStr, "Host key verification failed"),
		strings.Contains(errLower, "knownhosts: key mismatch"):
		return &SSHError{
			Category:  ErrHostKey,
			Message:   "Server host key has changed",
			Hint:      "The host's identity has changed since first contact. This could be a rebuild or a security issue. Remove the old key with: ssh-keygen -R " + host,
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errStr, "Permission denied"):
		return &SSHError{
			Category:  ErrAuth,
			Message:   "SSH authentication failed",
			Hint:      "The host rejected the SSH key. First touch installs the key using the owner password; verify the password and that key auth is enabled.",
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errLower, "unable to authenticate"),
		strings.Contains(errLower, "no supported methods remain"):
		return &SSHError{
			Category:  ErrAuth,
			Message:   "Password authentication failed",
			Hint:      "The password may be incorrect, or password authentication may be disabled on the host.",
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errLower, "connection refused"):
		return &SSHError{
			Category:  ErrUnreachable,
			Message:   "SSH connection refused",
			Hint:      "Verify the host and port are correct and that SSH is running on the host.",
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
			Hint:      "The network path to the host could not be found. Check the address and connectivity.",
			Retryable: true,
			Host:      host,
		}

	case strings.Contains(errLower, "no space left on device"):
		return &SSHError{
			Category:  ErrDiskSpace,
			Message:   "No space left on device",
			Hint:      "The host has run out of disk space. SSH in and run `df -h` to investigate.",
			Retryable: false,
			Host:      host,
		}

	case strings.Contains(errLower, "could not resolve hostname"),
		strings.Contains(errLower, "name or service not known"):
		return &SSHError{
			Category:  ErrDNS,
			Message:   "DNS resolution failed",
			Hint:      "DNS lookup failed. Check the hostname or use an IP address.",
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
			Hint:      "The host closed the connection. This may be temporary (load, fail2ban) or sshd MaxStartups being exceeded.",
			Retryable: true,
			Host:      host,
		}

	case strings.Contains(errLower, "broken pipe"):
		return &SSHError{
			Category:  ErrUnreachable,
			Message:   "SSH connection lost",
			Hint:      "The connection dropped during command execution. Check network stability.",
			Retryable: true,
			Host:      host,
		}

	case strings.Contains(errLower, "too many authentication failures"):
		return &SSHError{
			Category:  ErrAuth,
			Message:   "Too many authentication failures",
			Hint:      "The SSH agent offered too many keys. Add IdentitiesOnly=yes or specify the exact key with -i.",
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

// StatusFromError maps an SSHError to a status constant.
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

// newCommandError wraps a command-execution failure with output context.
func newCommandError(err error, res Result, host string) *SSHError {
	combined := res.Stderr + " " + err.Error()
	classified := ClassifyError(combined, host, "")

	var hint string
	switch {
	case res.Stderr != "":
		hint = res.Stderr
	case res.Stdout != "":
		lines := strings.Split(res.Stdout, "\n")
		if len(lines) > 50 {
			lines = lines[len(lines)-50:]
			hint = fmt.Sprintf("(last 50 lines of stdout):\n%s", strings.Join(lines, "\n"))
		} else {
			hint = "stdout: " + res.Stdout
		}
	default:
		hint = err.Error()
	}

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
