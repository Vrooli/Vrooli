package vps

import (
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// Run executes VPS subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "setup":
		return runSetup(client, args[1:])
	case "deploy":
		return runDeploy(client, args[1:])
	case "instance":
		return runInstance(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud vps help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud vps <command> [arguments]

Commands:
  setup plan <manifest.json> <bundle.tar.gz>    Generate VPS setup plan
  setup apply <manifest.json> <bundle.tar.gz>   Execute VPS setup
  deploy plan <manifest.json>                   Generate VPS deploy plan
  deploy apply <manifest.json>                  Execute VPS deploy
  instance plan --name <name> --image <path> --workdir <path>
                                                Validate the local-QEMU lane
  instance create --name <name> --image <path> --workdir <path>
                                                Register a disposable VM
  instance <start|stop|wait-for-ssh|snapshot|reset|destroy> <id> [snapshot]
                                                Manage a registered VM

Run 'scenario-to-cloud vps <command> -h' for command-specific options.`)
	return nil
}

func runInstance(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: scenario-to-cloud vps instance <plan|create|start|stop|destroy>")
	}
	if args[0] == "start" || args[0] == "stop" || args[0] == "destroy" || args[0] == "wait-for-ssh" || args[0] == "snapshot" || args[0] == "reset" {
		if len(args) < 2 || len(args) > 3 || args[1] == "" {
			return fmt.Errorf("usage: scenario-to-cloud vps instance %s <id>", args[0])
		}
		body, err := client.InstanceAction(args[1], args[0], optionalSnapshot(args))
		if err != nil {
			return err
		}
		cliutil.PrintJSON(body)
		return nil
	}
	if args[0] != "plan" && args[0] != "create" {
		return fmt.Errorf("unknown instance subcommand %q", args[0])
	}
	fs := flag.NewFlagSet("vps instance "+args[0], flag.ContinueOnError)
	name := fs.String("name", "", "instance name")
	image := fs.String("image", "", "path to a qcow2 image")
	workdir := fs.String("workdir", "", "instance work directory")
	memory := fs.String("memory", "2048", "RAM in megabytes")
	cpus := fs.Int("cpus", 2, "virtual CPUs")
	profile := fs.String("profile", "headless-linux", "VM profile")
	user := fs.String("user", "vrooli", "non-root VM user")
	authorizedKey := fs.String("authorized-key", "", "SSH public key for the VM user")
	sshPort := fs.Int("ssh-port", 2222, "host TCP port forwarded to VM SSH")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	request := map[string]interface{}{"name": *name, "image": *image, "workdir": *workdir, "memory": *memory, "cpus": *cpus, "profile": *profile, "user": *user, "authorized_key": *authorizedKey, "ssh_port": *sshPort}
	var body []byte
	var err error
	if args[0] == "plan" {
		body, err = client.InstancePlan(request)
	} else {
		body, err = client.InstanceCreate(request)
	}
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func optionalSnapshot(args []string) string {
	if len(args) == 3 {
		return args[2]
	}
	return ""
}

func runSetup(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: scenario-to-cloud vps setup <plan|apply> <manifest.json> <bundle.tar.gz>")
	}

	switch args[0] {
	case "plan":
		return runSetupPlan(client, args[1:])
	case "apply":
		return runSetupApply(client, args[1:])
	default:
		return fmt.Errorf("unknown setup subcommand: %s\n\nUsage: scenario-to-cloud vps setup <plan|apply>", args[0])
	}
}

func runSetupPlan(client *Client, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: scenario-to-cloud vps setup plan <manifest.json> <bundle.tar.gz>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.SetupPlan(manifest, args[1])
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runSetupApply(client *Client, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: scenario-to-cloud vps setup apply <manifest.json> <bundle.tar.gz>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.SetupApply(manifest, args[1])
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runDeploy(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: scenario-to-cloud vps deploy <plan|apply> <manifest.json>")
	}

	switch args[0] {
	case "plan":
		return runDeployPlan(client, args[1:])
	case "apply":
		return runDeployApply(client, args[1:])
	default:
		return fmt.Errorf("unknown deploy subcommand: %s\n\nUsage: scenario-to-cloud vps deploy <plan|apply>", args[0])
	}
}

func runDeployPlan(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud vps deploy plan <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.DeployPlan(manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runDeployApply(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud vps deploy apply <manifest.json>")
	}
	manifest, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.DeployApply(manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
