package permissions

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Codex runs a hook only after its owner trusts it (`/hooks` in the TUI); the
// grant is recorded in config.toml under hooks.state, keyed by position and
// hashed by content, so an edit revokes it. Vrooli never writes that grant: it
// is a human security decision. It does ask Codex itself, through the
// app-server `hooks/list` request (no model call, no session), so the answer
// is Codex's own and not a guess at its hash.

// Hook trust states, as Codex names them, plus unknown when Codex cannot be asked.
const (
	TrustTrusted   = "trusted"
	TrustUntrusted = "untrusted"
	TrustModified  = "modified"
	TrustManaged   = "managed"
	TrustUnknown   = "unknown"
)

// HookTrust is what Codex says about one installed command.
type HookTrust struct {
	Command string
	Status  string
}

// TrustProbe lists the trust Codex assigns to the hooks in codexHome/hooks.json.
type TrustProbe func(ctx context.Context, codexHome string) ([]HookTrust, error)

// AppServerTrustProbe asks `codex app-server` over stdio. The binary is the
// first `codex` on PATH, which is the resource-managed one.
func AppServerTrustProbe(ctx context.Context, codexHome string) ([]HookTrust, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "codex", "app-server")
	cmd.Env = append(os.Environ(), "CODEX_HOME="+codexHome)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start codex app-server: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		cancel()
		_ = cmd.Wait()
	}()
	return listHookTrust(stdin, stdout, codexHome)
}

// listHookTrust speaks the app-server's newline-delimited JSON-RPC: initialize,
// initialized, then hooks/list for the home directory. Hooks from other files
// (project or plugin hooks) are ignored.
func listHookTrust(w io.Writer, r io.Reader, codexHome string) ([]HookTrust, error) {
	send := func(message map[string]any) error {
		message["jsonrpc"] = "2.0"
		data, err := json.Marshal(message)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "%s\n", data)
		return err
	}
	if err := send(map[string]any{"id": 1, "method": "initialize", "params": map[string]any{"clientInfo": map[string]any{"name": "resource-codex", "version": "1"}}}); err != nil {
		return nil, err
	}
	if err := send(map[string]any{"method": "initialized"}); err != nil {
		return nil, err
	}
	if err := send(map[string]any{"id": 2, "method": "hooks/list", "params": map[string]any{"cwds": []string{codexHome}}}); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for scanner.Scan() {
		var reply struct {
			ID     *int `json:"id"`
			Result struct {
				Data []struct {
					Hooks []struct {
						Command     string `json:"command"`
						SourcePath  string `json:"sourcePath"`
						TrustStatus string `json:"trustStatus"`
					} `json:"hooks"`
				} `json:"data"`
			} `json:"result"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(scanner.Bytes(), &reply) != nil || reply.ID == nil || *reply.ID != 2 {
			continue
		}
		if reply.Error != nil {
			return nil, fmt.Errorf("codex hooks/list: %s", reply.Error.Message)
		}
		var trust []HookTrust
		seen := map[string]bool{}
		for _, entry := range reply.Result.Data {
			for _, hook := range entry.Hooks {
				key := hook.SourcePath + "\x00" + hook.Command
				if strings.HasSuffix(hook.SourcePath, "hooks.json") && !seen[key] {
					seen[key] = true
					trust = append(trust, HookTrust{Command: hook.Command, Status: hook.TrustStatus})
				}
			}
		}
		return trust, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("codex app-server closed without answering hooks/list")
}

// TrustOf returns the trust Codex gives a command, or unknown.
func TrustOf(trust []HookTrust, command string) string {
	for _, hook := range trust {
		if hook.Command == command {
			return hook.Status
		}
	}
	return TrustUnknown
}

// TrustAdvice is the line shown for a hook in a given trust state.
func TrustAdvice(status string) string {
	switch status {
	case TrustTrusted, TrustManaged:
		return "trusted: Codex will run it"
	case TrustUntrusted:
		return "NOT TRUSTED: Codex will not run it until you open Codex, run /hooks, review it and trust it"
	case TrustModified:
		return "CHANGED since you trusted it: Codex will not run it until you run /hooks and trust it again"
	default:
		return "trust unknown (could not ask Codex): run /hooks in Codex to check that it is trusted"
	}
}
