package sshadapter

import (
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/shellutil"
)

// previewConfig is the connection a preview renders for: the locator only.
// The key reference is resolved at execution time from the credential
// binding and never appears in a displayed command.
func previewConfig(loc identity.TargetLocator) ConnectionConfig {
	cfg := NewConfig(loc.Host, loc.Port, loc.User, "")
	cfg.KnownHostsFile = ""
	return cfg
}

// LocalSSHCommand renders the operator-side ssh invocation for a remote
// string. Display only; nothing it returns is executed.
func LocalSSHCommand(loc identity.TargetLocator, remote string) string {
	cfg := previewConfig(loc)
	args := append([]string{"ssh"}, buildArgs(cfg, RunOptions{ConnectTimeout: 5 * time.Second}, "-p")...)
	args = append(args, fmt.Sprintf("%s@%s", cfg.User, cfg.Host), "--", "bash", "-lc", shellutil.QuoteSingle(remote))
	return strings.Join(args, " ")
}

// LocalSCPCommand renders the operator-side scp invocation for one file.
// Display only.
func LocalSCPCommand(loc identity.TargetLocator, localPath, remotePath string) string {
	cfg := previewConfig(loc)
	args := append([]string{"scp"}, buildArgs(cfg, RunOptions{ConnectTimeout: 5 * time.Second, StrictHostKey: true}, "-P")...)
	args = append(args, localPath, fmt.Sprintf("%s@%s:%s", cfg.User, cfg.Host, remotePath))
	return strings.Join(args, " ")
}
