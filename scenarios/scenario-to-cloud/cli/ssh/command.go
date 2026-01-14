package ssh

import (
	"fmt"
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
  generate <name>         Generate a new SSH key
  delete <name>           Delete an SSH key
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
		fmt.Printf("%-20s %-10s %-47s %s\n", truncate(k.Name, 20), k.Type, k.Fingerprint, k.CreatedAt)
	}

	return nil
}

func runGenerate(client *Client, args []string) error {
	var name, keyType, comment string
	bits := 0
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud ssh generate <name> [flags]

Flags:
  --type <type>     Key type: ed25519 (default), rsa
  --bits <n>        Key size for RSA (2048, 4096)
  --comment <text>  Key comment
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
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && name == "" {
				name = args[i]
			}
		}
	}

	if name == "" {
		return fmt.Errorf("usage: scenario-to-cloud ssh generate <name>")
	}

	req := GenerateRequest{
		Name:    name,
		Type:    keyType,
		Bits:    bits,
		Comment: comment,
	}

	body, resp, err := client.Generate(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("Generated SSH key: %s\n", resp.Key.Name)
		fmt.Printf("Type:        %s\n", resp.Key.Type)
		fmt.Printf("Fingerprint: %s\n", resp.Key.Fingerprint)
		if resp.PrivateKey != "" {
			fmt.Println("\nPrivate Key (save this securely - it won't be shown again):")
			fmt.Println(resp.PrivateKey)
		}
		if resp.Key.PublicKey != "" {
			fmt.Println("\nPublic Key:")
			fmt.Println(resp.Key.PublicKey)
		}
	} else {
		fmt.Printf("Failed to generate key: %s\n", resp.Message)
	}

	return nil
}

func runDelete(client *Client, args []string) error {
	var name string
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud ssh delete <name> [flags]

Flags:
  --json    Output raw JSON`)
			return nil
		case "--json":
			jsonOutput = true
		default:
			if !strings.HasPrefix(args[i], "-") && name == "" {
				name = args[i]
			}
		}
	}

	if name == "" {
		return fmt.Errorf("usage: scenario-to-cloud ssh delete <name>")
	}

	body, resp, err := client.Delete(name)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Success {
		fmt.Printf("Deleted SSH key: %s\n", resp.Name)
	} else {
		fmt.Printf("Failed to delete key '%s': %s\n", resp.Name, resp.Message)
	}

	return nil
}

func runTest(client *Client, args []string) error {
	var host, user, keyName string
	port := 22
	timeout := 10
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Println(`Usage: scenario-to-cloud ssh test <host> [flags]

Flags:
  --port <n>        SSH port (default: 22)
  --user <name>     SSH user (default: root)
  --key <name>      SSH key name to use
  --timeout <n>     Connection timeout in seconds (default: 10)
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
				keyName = args[i]
			}
		case "--timeout":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil {
					timeout = n
				}
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

	req := TestRequest{
		Host:    host,
		Port:    port,
		User:    user,
		KeyName: keyName,
		Timeout: timeout,
	}

	body, resp, err := client.Test(req)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("SSH Connection Test to %s@%s:%d\n", resp.User, resp.Host, resp.Port)
	fmt.Println(strings.Repeat("-", 50))

	if resp.Success {
		fmt.Println("Status: SUCCESS")
		if resp.Latency != "" {
			fmt.Printf("Latency: %s\n", resp.Latency)
		}
		if resp.ServerBanner != "" {
			fmt.Printf("Server: %s\n", resp.ServerBanner)
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
	var host, user, keyName, password string
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
  --key <name>      SSH key name to copy
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
				keyName = args[i]
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

	req := CopyKeyRequest{
		Host:     host,
		Port:     port,
		User:     user,
		KeyName:  keyName,
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

	if resp.Success {
		fmt.Printf("Successfully copied key '%s' to %s@%s\n", resp.KeyName, resp.User, resp.Host)
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
