package credentials

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"golang.org/x/term"
)

// StoreRegister exposes the onboarding-owned credential protection verbs. It
// is an API adapter: securestore remains the control-plane authority, and
// secret-bearing operations read only from stdin.
func StoreRegister(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "store", Description: "Configure encrypted credential-store protection", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "status", Description: "Show metadata-safe credential-store status", Run: func(args []string) error { return storeGet(core, args, "/v2/credentials/store/status") }},
		{Name: "select", Description: "Select the credential backend", Run: func(args []string) error { return storeSelect(core, args) }},
		{Name: "init", Description: "Initialize the encrypted store from standard input", Run: func(args []string) error { return storeSecret(core, args, "/v2/credentials/store/init", "init") }},
		{Name: "unlock", Description: "Unlock the encrypted store from standard input", Run: func(args []string) error { return storeSecret(core, args, "/v2/credentials/store/unlock", "unlock") }},
		{Name: "change-passphrase", Description: "Change the encrypted-store passphrase from two stdin lines", Run: func(args []string) error { return storeChangePassphrase(core, args) }},
		{Name: "rewrap", Description: "Add an unattended native wrap using standard input", Run: func(args []string) error { return storeSecret(core, args, "/v2/credentials/store/rewrap", "rewrap") }},
	}}
}

func storeGet(core *cliapp.ScenarioApp, args []string, path string) error {
	fs := support.NewFlagSet("store status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(body, '\n'))
		return err
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("decode credential store response: %w", err)
	}
	pretty, _ := json.MarshalIndent(value, "", "  ")
	_, err = fmt.Fprintln(os.Stdout, string(pretty))
	return err
}

func storeSelect(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("store select")
	backend := fs.String("backend", "", "Backend: native or encrypted-file")
	reason := fs.String("reason", "selected through onboarding", "Metadata-only reason for the selection")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*backend) == "" {
		return fmt.Errorf("--backend is required")
	}
	body, _ := json.Marshal(map[string]string{"backend": strings.TrimSpace(*backend), "reason": strings.TrimSpace(*reason)})
	return storeMutation(core, "/v2/credentials/store/select", body, *jsonOutput, "Credential backend selected")
}

func storeSecret(core *cliapp.ScenarioApp, args []string, path, operation string) error {
	fs := support.NewFlagSet("store " + operation)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	secret, err := readStoreSecret()
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"passphrase": secret})
	return storeMutation(core, path, body, *jsonOutput, "Credential store "+operation+" completed")
}

func storeChangePassphrase(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("store change-passphrase")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	current, next, err := readStorePassphrases()
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"current_passphrase": strings.TrimSpace(current), "new_passphrase": strings.TrimSpace(next)})
	return storeMutation(core, "/v2/credentials/store/change-passphrase", body, *jsonOutput, "Credential store passphrase changed")
}

func readStoreSecret() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		_, _ = fmt.Fprint(os.Stderr, "Credential-store passphrase: ")
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		_, _ = fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("read credential store passphrase: %w", err)
		}
		if len(value) == 0 {
			return "", fmt.Errorf("credential store passphrase must be supplied")
		}
		return string(value), nil
	}
	value, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read credential store passphrase: %w", err)
	}
	value = []byte(strings.TrimSpace(string(value)))
	if len(value) == 0 {
		return "", fmt.Errorf("credential store passphrase must be supplied on standard input")
	}
	return string(value), nil
}

func readStorePassphrases() (string, string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		current, err := readStoreSecretPrompt("Current credential-store passphrase: ")
		if err != nil {
			return "", "", err
		}
		next, err := readStoreSecretPrompt("New credential-store passphrase: ")
		if err != nil {
			return "", "", err
		}
		return current, next, nil
	}
	contents, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", "", fmt.Errorf("read credential store passphrases: %w", err)
	}
	current, next, found := strings.Cut(strings.TrimSpace(string(contents)), "\n")
	if !found || strings.TrimSpace(current) == "" || strings.TrimSpace(next) == "" {
		return "", "", fmt.Errorf("pipe current and new passphrases on separate stdin lines")
	}
	return strings.TrimSpace(current), strings.TrimSpace(next), nil
}

func readStoreSecretPrompt(prompt string) (string, error) {
	_, _ = fmt.Fprint(os.Stderr, prompt)
	value, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read credential store passphrase: %w", err)
	}
	if len(value) == 0 {
		return "", fmt.Errorf("credential store passphrase must be supplied")
	}
	return string(value), nil
}

func storeMutation(core *cliapp.ScenarioApp, path string, body []byte, jsonOutput bool, message string) error {
	response, err := core.Request("POST", path, nil, body)
	if err != nil {
		return err
	}
	if jsonOutput {
		_, err = os.Stdout.Write(append(response, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{message}, NextCommand: []string{support.CLIName + " readiness"}})
}
