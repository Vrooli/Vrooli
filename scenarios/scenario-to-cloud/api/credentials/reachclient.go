package credentials

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// ArgvRunner runs one argv on a target with an optional standard input. It is
// the only shape a value may travel through: stdin, never argv. It matches
// credentialclient.SSHRunner so one adapter serves both.
type ArgvRunner interface {
	Run(ctx context.Context, target string, args []string, stdin io.Reader) ([]byte, error)
}

// ReachArgvRunner adapts the bound reach to ArgvRunner. The argv the
// credential client builds (`vrooli credentials <sub…> --flag …`) becomes a
// typed reach command: the verb is the command path up to the first flag,
// the flags are the arguments, and stdin travels on the command's Stdin.
type ReachArgvRunner struct {
	Reach  reach.Reach
	Target identity.TargetRef
}

// effectfulCredentialVerbs are the credential subcommands that change the
// target's authority state and therefore need the write scope.
var effectfulCredentialVerbs = map[string]bool{
	"credentials provision":        true,
	"credentials recovery export":  true,
	"credentials recovery restore": true,
}

// CommandFor turns a credential-client argv into a reach command.
func CommandFor(args []string, stdin []byte) (reach.Command, error) {
	if len(args) == 0 {
		return reach.Command{}, fmt.Errorf("credential argv is empty")
	}
	if args[0] == "vrooli" {
		args = args[1:]
	}
	verb := make([]string, 0, 3)
	rest := args
	for len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		verb = append(verb, rest[0])
		rest = rest[1:]
	}
	if len(verb) == 0 {
		return reach.Command{}, fmt.Errorf("credential argv names no verb")
	}
	cmd := reach.Command{Verb: strings.Join(verb, " "), Args: rest, Timeout: 60 * time.Second, RequiredScope: "vrooli:read"}
	if effectfulCredentialVerbs[cmd.Verb] || len(stdin) > 0 {
		cmd.Effectful = true
		cmd.RequiredScope = "vrooli:write"
	}
	if len(stdin) > 0 {
		cmd.Stdin = stdin
	}
	return cmd, nil
}

// Run executes args on the bound target.
func (r ReachArgvRunner) Run(ctx context.Context, _ string, args []string, stdin io.Reader) ([]byte, error) {
	var payload []byte
	if stdin != nil {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}
		payload = data
	}
	cmd, err := CommandFor(args, payload)
	if err != nil {
		return nil, err
	}
	if r.Reach == nil {
		return nil, &reach.Error{Kind: reach.KindUnavailable, Target: r.Target.Key(), Detail: "no reach bound for credential verbs"}
	}
	result, err := r.Reach.Exec(ctx, r.Target, cmd)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return []byte(result.Stdout), &RemoteExitError{ExitCode: result.ExitCode, Stdout: result.Stdout, Stderr: result.Stderr}
	}
	return []byte(result.Stdout), nil
}

// RemoteExitError is a command that ran on the target and failed. It is
// distinct from a transport failure, which means the target was not reached.
type RemoteExitError struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func (e *RemoteExitError) Error() string {
	return fmt.Sprintf("remote command exited %d: %s", e.ExitCode, strings.TrimSpace(e.Stderr))
}

// TargetLabel renders the user@host label the credential client tags its
// calls with.
func TargetLabel(target identity.TargetRef) string {
	user := target.Locator.User
	if user == "" {
		user = "root"
	}
	return fmt.Sprintf("%s@%s", user, target.Locator.Host)
}

// NewReachCredentialClient builds the remote credential-authority client over
// the bound reach. Metadata reads and stdin-only provisioning are the whole
// surface; Resolve is never called by the cloud side.
func NewReachCredentialClient(rr reach.Reach, target identity.TargetRef) (credentialclient.Client, error) {
	return credentialclient.NewClient(credentialclient.ClientOptions{RemoteTarget: TargetLabel(target), RemoteRunner: ReachArgvRunner{Reach: rr, Target: target}})
}
