package ssh

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Run executes SSH subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "keys":
		return runKeys(client, args[1:])
	case "generate":
		return runGenerate(client, args[1:])
	case "delete":
		return runDelete(client, args[1:])
	case "test":
		return runTest(client, args[1:])
	case "bootstrap":
		return runBootstrap(client, args[1:])
	case "copy-key":
		return runCopyKey(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud ssh help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud ssh <command> [arguments]

Commands:
  keys                    List all SSH keys
  generate <filename>     Generate a new SSH key
  delete <key>            Delete an SSH key (path or basename)
  test <host>             Test SSH connection to a host
  bootstrap <host>        Ensure key-based SSH works (test -> generate -> copy-key -> retest)
  copy-key <host>         Copy SSH key to a remote host

Run 'scenario-to-cloud ssh <command> -h' for command-specific options.`)
	return nil
}

func runKeys(client *Client, args []string) error {
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud ssh keys [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		}
	}

	body, resp, err := client.Keys()
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print
	if len(resp.Keys) == 0 {
		fmt.Println("No SSH keys found.")
		return nil
	}

	fmt.Println("SSH Keys:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-20s %-10s %-47s %s\n", "NAME", "TYPE", "FINGERPRINT", "CREATED")

	for _, k := range resp.Keys {
		fmt.Printf("%-20s %-10s %-47s %s\n", truncate(filepath.Base(k.Path), 20), k.Type, k.Fingerprint, k.CreatedAt)
	}

	return nil
}

func runGenerate(client *Client, args []string) error {
	var filename, keyType, comment, password string
	bits := 0
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud ssh generate <filename> [flags]

Flags:
  --type <type>     Key type: ed25519 (default), rsa
  --bits <n>        Key size for RSA (2048, 4096)
  --comment <text>  Key comment
  --password <pwd>  Optional key passphrase
  --json            Output raw JSON`)
			return nil
		case "--type":
			if i+1 < len(args) {
				i++
				keyType = args[i]
			}
		case "--bits":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					bits = n
				}
			}
		case "--comment":
			if i+1 < len(args) {
				i++
				comment = args[i]
			}
		case "--password":
			if i+1 < len(args) {
				i++
				password = args[i]
			}
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && filename == "" {
				filename = args[i]
			}
		}
	}

	if filename == "" {
		return fmt.Errorf("usage: scenario-to-cloud ssh generate <filename>")
	}

	req := GenerateRequest{
		Type:     keyType,
		Bits:     bits,
		Comment:  comment,
		Filename: filename,
		Password: password,
	}

	body, resp, err := client.Generate(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.OK {
		fmt.Printf("Generated SSH key: %s\n", filepath.Base(resp.Key.Path))
		fmt.Printf("Path:        %s\n", resp.Key.Path)
		fmt.Printf("Type:        %s\n", resp.Key.Type)
		fmt.Printf("Fingerprint: %s\n", resp.Key.Fingerprint)
	} else {
		fmt.Printf("Failed to generate key: %s\n", resp.Message)
	}

	return nil
}

func runDelete(client *Client, args []string) error {
	var keyInput string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud ssh delete <key> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && keyInput == "" {
				keyInput = args[i]
			}
		}
	}

	if keyInput == "" {
		return fmt.Errorf("usage: scenario-to-cloud ssh delete <key>")
	}

	keyPath, err := resolveKeyPath(client, keyInput)
	if err != nil {
		return err
	}

	body, resp, err := client.Delete(DeleteRequest{KeyPath: keyPath})
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.OK {
		fmt.Printf("Deleted SSH key: %s\n", keyPath)
	} else {
		fmt.Printf("Failed to delete key '%s': %s\n", keyPath, resp.Message)
	}

	return nil
}

func runTest(client *Client, args []string) error {
	var host, user, keyInput string
	port := 22
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud ssh test <host> [flags]

Flags:
  --port <n>        SSH port (default: 22)
  --user <name>     SSH user (default: root)
  --key <key>       SSH key path or basename
  --json            Output raw JSON`)
			return nil
		case "--port":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					port = n
				}
			}
		case "--user":
			if i+1 < len(args) {
				i++
				user = args[i]
			}
		case "--key":
			if i+1 < len(args) {
				i++
				keyInput = args[i]
			}
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && host == "" {
				host = args[i]
			}
		}
	}

	if host == "" {
		return fmt.Errorf("usage: scenario-to-cloud ssh test <host>")
	}

	keyPath, err := resolveKeyPath(client, keyInput)
	if err != nil {
		return err
	}

	req := TestRequest{
		Host:    host,
		Port:    port,
		User:    user,
		KeyPath: keyPath,
	}

	body, resp, err := client.Test(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	displayUser := user
	if strings.TrimSpace(displayUser) == "" {
		displayUser = "root"
	}
	fmt.Printf("SSH Connection Test to %s@%s:%d\n", displayUser, host, port)
	fmt.Println(strings.Repeat("-", 50))

	if resp.OK {
		fmt.Println("Status: SUCCESS")
		if resp.LatencyMs > 0 {
			fmt.Printf("Latency: %dms\n", resp.LatencyMs)
		}
		if resp.ServerInfo != "" {
			fmt.Printf("Server: %s\n", resp.ServerInfo)
		}
		if resp.Fingerprint != "" {
			fmt.Printf("Fingerprint: %s\n", resp.Fingerprint)
		}
	} else {
		fmt.Println("Status: FAILED")
		if resp.Message != "" {
			fmt.Printf("Error: %s\n", resp.Message)
		}
	}

	return nil
}

func runCopyKey(client *Client, args []string) error {
	var host, user, keyInput, password string
	port := 22
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud ssh copy-key <host> [flags]

Copies an SSH public key to a remote host's authorized_keys file.

Flags:
  --port <n>        SSH port (default: 22)
  --user <name>     SSH user (default: root)
  --key <key>       SSH key path or basename
  --password <pwd>  Password for initial authentication
  --json            Output raw JSON`)
			return nil
		case "--port":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					port = n
				}
			}
		case "--user":
			if i+1 < len(args) {
				i++
				user = args[i]
			}
		case "--key":
			if i+1 < len(args) {
				i++
				keyInput = args[i]
			}
		case "--password":
			if i+1 < len(args) {
				i++
				password = args[i]
			}
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && host == "" {
				host = args[i]
			}
		}
	}

	if host == "" {
		return fmt.Errorf("usage: scenario-to-cloud ssh copy-key <host>")
	}

	keyPath, err := resolveKeyPath(client, keyInput)
	if err != nil {
		return err
	}

	req := CopyKeyRequest{
		Host:     host,
		Port:     port,
		User:     user,
		KeyPath:  keyPath,
		Password: password,
	}

	body, resp, err := client.CopyKey(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.OK {
		displayUser := user
		if strings.TrimSpace(displayUser) == "" {
			displayUser = "root"
		}
		fmt.Printf("Successfully copied key '%s' to %s@%s\n", keyPath, displayUser, req.Host)
	} else {
		fmt.Printf("Failed to copy key: %s\n", resp.Message)
	}

	return nil
}

func runBootstrap(client *Client, args []string) error {
	var host, user, keyInput, password string
	port := 22
	jsonOutput := false
	nonInteractive := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud ssh bootstrap <host> [flags]

Ensures key-based SSH access by running:
  1) key selection/generation
  2) ssh test
  3) key install (if needed)
  4) ssh retest

Flags:
  --port <n>            SSH port (default: 22)
  --user <name>         SSH user (default: root)
  --key <key>           SSH key path or basename (default: first discovered key, or auto-generate "s2c-deploy")
  --password <pwd>      VPS password for one-time key installation
  --non-interactive     Fail with handoff instructions instead of prompting for password
  --json                Output raw JSON`)
			return nil
		case "--port":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					port = n
				}
			}
		case "--user":
			if i+1 < len(args) {
				i++
				user = args[i]
			}
		case "--key":
			if i+1 < len(args) {
				i++
				keyInput = args[i]
			}
		case "--password":
			if i+1 < len(args) {
				i++
				password = args[i]
			}
		case "--non-interactive":
			nonInteractive = true
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && host == "" {
				host = args[i]
			}
		}
	}

	if host == "" {
		return fmt.Errorf("usage: scenario-to-cloud ssh bootstrap <host>")
	}

	displayUser := strings.TrimSpace(user)
	if displayUser == "" {
		displayUser = "root"
	}

	selectedKeyPath, selectedKeyName, generated, err := ensureBootstrapKey(client, keyInput, nonInteractive, host, displayUser, port)
	if err != nil {
		return err
	}

	testReq := TestRequest{
		Host:    host,
		Port:    port,
		User:    displayUser,
		KeyPath: selectedKeyPath,
	}

	_, initialTest, err := client.Test(testReq)
	if err != nil {
		return err
	}

	if initialTest.OK {
		if jsonOutput {
			cliutil.PrintJSON(mustMarshalBootstrap(bootstrapOutput{
				OK:                 true,
				Host:               host,
				User:               displayUser,
				Port:               port,
				KeyPath:            selectedKeyPath,
				KeyGenerated:       generated,
				InitialTest:        initialTest,
				CopyKeyAttempted:   false,
				ConnectionVerified: true,
			}))
			return nil
		}
		fmt.Printf("SSH bootstrap complete: key-based access already working for %s@%s:%d\n", displayUser, host, port)
		fmt.Printf("Using key: %s\n", selectedKeyPath)
		return nil
	}

	if nonInteractive {
		return errors.New(nonInteractiveBootstrapMessage(host, displayUser, port, selectedKeyName))
	}

	if strings.TrimSpace(password) == "" {
		prompt := fmt.Sprintf("Enter VPS password for %s@%s (input visible): ", displayUser, host)
		fmt.Print(prompt)
		reader := bufio.NewReader(os.Stdin)
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			return fmt.Errorf("read password: %w", readErr)
		}
		password = strings.TrimSpace(line)
		if password == "" {
			return fmt.Errorf("password is required to install SSH key. Re-run with --password or enter it at prompt")
		}
	}

	_, copyResp, err := client.CopyKey(CopyKeyRequest{
		Host:     host,
		Port:     port,
		User:     displayUser,
		KeyPath:  selectedKeyPath,
		Password: password,
	})
	if err != nil {
		return err
	}

	_, finalTest, err := client.Test(testReq)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(mustMarshalBootstrap(bootstrapOutput{
			OK:                 finalTest.OK,
			Host:               host,
			User:               displayUser,
			Port:               port,
			KeyPath:            selectedKeyPath,
			KeyGenerated:       generated,
			InitialTest:        initialTest,
			CopyKeyAttempted:   true,
			CopyKey:            &copyResp,
			FinalTest:          &finalTest,
			ConnectionVerified: finalTest.OK,
		}))
		return nil
	}

	if !copyResp.OK {
		return fmt.Errorf("failed to install key on %s@%s: %s", displayUser, host, copyResp.Message)
	}
	if !finalTest.OK {
		return fmt.Errorf("key was installed but SSH test still failed: %s", finalTest.Message)
	}

	fmt.Printf("SSH bootstrap complete for %s@%s:%d\n", displayUser, host, port)
	fmt.Printf("Using key: %s\n", selectedKeyPath)
	return nil
}

type bootstrapOutput struct {
	OK                 bool             `json:"ok"`
	Host               string           `json:"host"`
	User               string           `json:"user"`
	Port               int              `json:"port"`
	KeyPath            string           `json:"key_path"`
	KeyGenerated       bool             `json:"key_generated"`
	InitialTest        TestResponse     `json:"initial_test"`
	CopyKeyAttempted   bool             `json:"copy_key_attempted"`
	CopyKey            *CopyKeyResponse `json:"copy_key,omitempty"`
	FinalTest          *TestResponse    `json:"final_test,omitempty"`
	ConnectionVerified bool             `json:"connection_verified"`
}

func mustMarshalBootstrap(v bootstrapOutput) []byte {
	out, _ := json.MarshalIndent(v, "", "  ")
	return out
}

func ensureBootstrapKey(client *Client, keyInput string, nonInteractive bool, host, user string, port int) (path string, keyName string, generated bool, err error) {
	trimmed := strings.TrimSpace(keyInput)
	if trimmed != "" {
		keyPath, resolveErr := resolveKeyPath(client, trimmed)
		if resolveErr == nil {
			return keyPath, filepath.Base(keyPath), false, nil
		}

		// If user provided a basename and it doesn't exist, bootstrap can generate it.
		if strings.Contains(trimmed, "/") || strings.HasPrefix(trimmed, "~") {
			return "", "", false, resolveErr
		}
		if nonInteractive {
			return "", "", false, errors.New(nonInteractiveBootstrapMessage(host, user, port, trimmed))
		}
		if genErr := generateBootstrapKey(client, trimmed); genErr != nil {
			return "", "", false, genErr
		}
		keyPath, resolveErr = resolveKeyPath(client, trimmed)
		if resolveErr != nil {
			return "", "", false, resolveErr
		}
		return keyPath, trimmed, true, nil
	}

	keyPath, resolveErr := resolveKeyPath(client, "")
	if resolveErr == nil {
		return keyPath, filepath.Base(keyPath), false, nil
	}
	if nonInteractive {
		return "", "", false, errors.New(nonInteractiveBootstrapMessage(host, user, port, "s2c-deploy"))
	}
	if genErr := generateBootstrapKey(client, "s2c-deploy"); genErr != nil {
		return "", "", false, genErr
	}
	keyPath, resolveErr = resolveKeyPath(client, "s2c-deploy")
	if resolveErr != nil {
		return "", "", false, resolveErr
	}
	return keyPath, "s2c-deploy", true, nil
}

func generateBootstrapKey(client *Client, filename string) error {
	_, resp, err := client.Generate(GenerateRequest{Filename: filename})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("generate SSH key %q failed: %s", filename, resp.Message)
	}
	return nil
}

func nonInteractiveBootstrapMessage(host, user string, port int, keyName string) string {
	if strings.TrimSpace(keyName) == "" {
		keyName = "s2c-deploy"
	}
	cmd := fmt.Sprintf("scenario-to-cloud ssh bootstrap %s --user %s --port %d --key %s", host, user, port, keyName)
	return "ssh bootstrap requires interactive password entry to install key authorization on the VPS.\n" +
		"Run this exact command without --non-interactive and enter the VPS password when prompted:\n  " + cmd + "\n" +
		"If you are an AI agent, ask a human to run that command. The password prompt is interactive and cannot be completed autonomously."
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func resolveKeyPath(client *Client, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		_, keysResp, err := client.Keys()
		if err != nil {
			return "", err
		}
		if len(keysResp.Keys) == 0 {
			return "", fmt.Errorf("no SSH keys found; use 'scenario-to-cloud ssh generate <filename>' first")
		}
		return keysResp.Keys[0].Path, nil
	}

	if strings.Contains(input, "/") || strings.HasPrefix(input, "~") {
		return input, nil
	}

	_, keysResp, err := client.Keys()
	if err != nil {
		return "", err
	}
	for _, key := range keysResp.Keys {
		if filepath.Base(key.Path) == input {
			return key.Path, nil
		}
	}

	return "", fmt.Errorf("SSH key %q not found; use a full path or one of: %s", input, joinKeyNames(keysResp.Keys))
}

func joinKeyNames(keys []SSHKey) string {
	if len(keys) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(keys))
	for _, key := range keys {
		names = append(names, filepath.Base(key.Path))
	}
	return strings.Join(names, ", ")
}
