package ssh

import (
	"fmt"
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
