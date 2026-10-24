// Package credential is the CLI surface of deployment credential bindings:
// list, rotate, revoke, recover and rotation get/resume over the generated
// CredentialsService client. Values never travel in argv: a replacement
// value or a recovery passphrase is read from standard input.
package credential

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	credentialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/credentials"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/credentials/credentialsv1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/protoout"
	"scenario-to-cloud/cli/internal/selector"
	"scenario-to-cloud/cli/internal/transport"
)

// Commands is the credential command group.
type Commands struct {
	Service     credentialsv1connect.CredentialsServiceClient
	Deployments deploymentsv1connect.DeploymentsServiceClient
	Stdin       io.Reader
}

// New builds the group over the shared transport.
func New(tr transport.Transport) Commands {
	return Commands{
		Service:     credentialsv1connect.NewCredentialsServiceClient(tr.HTTP, tr.BaseURL),
		Deployments: deploymentsv1connect.NewDeploymentsServiceClient(tr.HTTP, tr.BaseURL),
		Stdin:       os.Stdin,
	}
}

// Run executes credential subcommands.
func (c Commands) Run(args []string) error {
	if len(args) == 0 {
		return printUsage()
	}
	switch args[0] {
	case "list":
		return c.runList(args[1:])
	case "rotate":
		return c.runRotate(args[1:])
	case "revoke":
		return c.runRevoke(args[1:])
	case "recover":
		return c.runRecover(args[1:])
	case "rotation-get":
		return c.runRotation("get", args[1:])
	case "rotation-resume":
		return c.runRotation("resume", args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud credential help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud credential <command> <selector> [flags]

Credential bindings of a deployment. Values never appear in arguments:
a replacement value (rotate --value-stdin) or a recovery passphrase
(recover --passphrase-stdin) is read from standard input.

Commands:
  list <selector>                                   Bindings, versions and consumers
  rotate <selector> --binding <id> [--value-stdin]  Rotate one binding (versioned; consumers verified first)
  revoke <selector> --binding <id>                  Revoke one binding everywhere it was distributed
  recover <selector> --bundle-ref <path> --passphrase-stdin
                                                    Recover the credential store onto a replacement host
  rotation-get <selector> --rotation <id>           Standing of one rotation/revocation/recovery
  rotation-resume <selector> --rotation <id> [--operator-confirmed]
                                                    Resume an operation parked on operator input

Selector: ` + selector.Usage + `

Exit codes: 0 completed, 1 failed, 2 refused, 3 pending (operator input or
unreached consumers keep the operation explicitly incomplete).`)
	return nil
}

func (c Commands) resolve(fs *flag.FlagSet, sel *selector.Flags) (string, string, error) {
	chosen, err := sel.Selector(fs.Args())
	if err != nil {
		return "", "", err
	}
	ref, err := selector.Resolve(context.Background(), c.Deployments, chosen)
	if err != nil {
		return "", "", err
	}
	return ref.GetId(), selector.Identity(ref), nil
}

func (c Commands) runList(args []string) error {
	fs := flag.NewFlagSet("credential list", flag.ContinueOnError)
	sel := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	id, identity, err := c.resolve(fs, sel)
	if err != nil {
		return err
	}
	resp, err := c.Service.ListBindings(context.Background(), connect.NewRequest(&credentialsv1.ListBindingsRequest{DeploymentId: id}))
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(resp.Msg)
	}
	fmt.Println(identity)
	if len(resp.Msg.GetBindings()) == 0 {
		fmt.Println("No credential bindings.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BINDING\tLOGICAL ID\tFIELD\tCLASS\tSTATE\tVERSION\tCONSUMERS\tACKS")
	for _, view := range resp.Msg.GetBindings() {
		b := view.GetBinding()
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\n", b.GetId(), b.GetDescriptor_().GetLogicalId(), b.GetDescriptor_().GetField(), b.GetClass(), b.GetState(), b.GetVersion().GetNumber(), len(b.GetConsumerRefs()), len(view.GetAcks()))
	}
	return w.Flush()
}

func (c Commands) runRotate(args []string) error {
	fs := flag.NewFlagSet("credential rotate", flag.ContinueOnError)
	sel := selector.Register(fs)
	binding := fs.String("binding", "", "Binding id (required)")
	valueStdin := fs.Bool("value-stdin", false, "Read the replacement value for an external credential from standard input")
	requestKey := fs.String("request-key", "", "Idempotency key")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*binding) == "" {
		return apierr.Refused("--binding is required")
	}
	value := ""
	if *valueStdin {
		v, err := readSecret(c.Stdin)
		if err != nil {
			return err
		}
		value = v
	}
	id, identity, err := c.resolve(fs, sel)
	if err != nil {
		return err
	}
	resp, err := c.Service.RotateCredential(context.Background(), connect.NewRequest(&credentialsv1.RotateCredentialRequest{DeploymentId: id, BindingId: *binding, Value: value, RequestKey: *requestKey}))
	if err != nil {
		return err
	}
	return c.report(identity, resp.Msg, *jsonOutput)
}

func (c Commands) runRevoke(args []string) error {
	fs := flag.NewFlagSet("credential revoke", flag.ContinueOnError)
	sel := selector.Register(fs)
	binding := fs.String("binding", "", "Binding id (required)")
	requestKey := fs.String("request-key", "", "Idempotency key")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*binding) == "" {
		return apierr.Refused("--binding is required")
	}
	id, identity, err := c.resolve(fs, sel)
	if err != nil {
		return err
	}
	resp, err := c.Service.RevokeCredential(context.Background(), connect.NewRequest(&credentialsv1.RevokeCredentialRequest{DeploymentId: id, BindingId: *binding, RequestKey: *requestKey}))
	if err != nil {
		return err
	}
	return c.report(identity, resp.Msg, *jsonOutput)
}

func (c Commands) runRecover(args []string) error {
	fs := flag.NewFlagSet("credential recover", flag.ContinueOnError)
	sel := selector.Register(fs)
	bundleRef := fs.String("bundle-ref", "", "Path of the encrypted recovery bundle on the replacement host (required)")
	passphraseStdin := fs.Bool("passphrase-stdin", false, "Read the recovery passphrase from standard input (required)")
	requestKey := fs.String("request-key", "", "Idempotency key")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*bundleRef) == "" {
		return apierr.Refused("--bundle-ref is required")
	}
	if !*passphraseStdin {
		return apierr.Refused("--passphrase-stdin is required: the passphrase is read from standard input, never from an argument")
	}
	passphrase, err := readSecret(c.Stdin)
	if err != nil {
		return err
	}
	id, identity, err := c.resolve(fs, sel)
	if err != nil {
		return err
	}
	resp, err := c.Service.RecoverCredentials(context.Background(), connect.NewRequest(&credentialsv1.RecoverCredentialsRequest{DeploymentId: id, BundleRef: *bundleRef, Passphrase: passphrase, RequestKey: *requestKey}))
	if err != nil {
		return err
	}
	return c.report(identity, resp.Msg, *jsonOutput)
}

func (c Commands) runRotation(verb string, args []string) error {
	fs := flag.NewFlagSet("credential rotation-"+verb, flag.ContinueOnError)
	sel := selector.Register(fs)
	rotation := fs.String("rotation", "", "Rotation id (required)")
	confirmed := fs.Bool("operator-confirmed", false, "resume only: the handed-off provider step has been completed")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*rotation) == "" {
		return apierr.Refused("--rotation is required")
	}
	id, identity, err := c.resolve(fs, sel)
	if err != nil {
		return err
	}
	var msg *credentialsv1.CredentialOperationResponse
	if verb == "get" {
		resp, err := c.Service.GetRotation(context.Background(), connect.NewRequest(&credentialsv1.GetRotationRequest{DeploymentId: id, RotationId: *rotation}))
		if err != nil {
			return err
		}
		msg = resp.Msg
	} else {
		resp, err := c.Service.ResumeRotation(context.Background(), connect.NewRequest(&credentialsv1.ResumeRotationRequest{DeploymentId: id, RotationId: *rotation, OperatorConfirmed: *confirmed}))
		if err != nil {
			return err
		}
		msg = resp.Msg
	}
	return c.report(identity, msg, *jsonOutput)
}

// report prints the operation and maps its state to the exit contract.
func (c Commands) report(identity string, resp *credentialsv1.CredentialOperationResponse, jsonOutput bool) error {
	if jsonOutput {
		return protoout.Print(resp)
	}
	fmt.Println(identity)
	op := resp.GetOperation()
	fmt.Printf("credential operation: %s  kind: %s  state: %s\n", op.GetId(), op.GetKind(), op.GetState())
	if op.GetBindingId() != "" {
		fmt.Printf("  binding: %s  version: %d -> %d\n", op.GetBindingId(), op.GetFromVersion(), op.GetToVersion())
	}
	for _, p := range op.GetConsumers() {
		line := fmt.Sprintf("  consumer %s: %s", p.GetConsumer(), p.GetState())
		if p.GetReason() != "" {
			line += " (" + p.GetReason() + ")"
		}
		fmt.Println(line)
	}
	if len(op.GetUnreached()) > 0 {
		fmt.Printf("  unreached: %s\n", strings.Join(op.GetUnreached(), ", "))
	}
	if h := op.GetPendingOperatorInput(); h != nil && h.GetReference() != "" {
		fmt.Printf("  operator input: %s (%s) %s\n", h.GetReference(), h.GetProvider(), h.GetInstruction())
		if h.GetResumeWith() != "" {
			fmt.Printf("  resume with: %s\n", h.GetResumeWith())
		}
	}
	if e := op.GetError(); e != nil && e.GetCode() != "" {
		fmt.Printf("  error: %s (%s)\n", e.GetMessage(), e.GetCode())
	}
	return ExitFor(op)
}

// ExitFor maps a credential operation state to the exit contract.
func ExitFor(op *credentialsv1.CredentialOperation) error {
	switch op.GetState() {
	case "completed", "succeeded", "active", "revoked", "recovered":
		return nil
	case "failed":
		return apierr.Failed("credential operation %s failed", op.GetId())
	case "pending_operator_input", "revocation_incomplete", "consumers_updated", "recovering", "distributing", "verifying", "admitted", "running":
		return apierr.Pending("credential operation %s is %s; resume: scenario-to-cloud credential rotation-resume --deployment %s --rotation %s", op.GetId(), op.GetState(), op.GetDeploymentId(), op.GetId())
	default:
		return apierr.Pending("credential operation %s is %s", op.GetId(), op.GetState())
	}
}

func readSecret(r io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(r, 64*1024))
	if err != nil {
		return "", fmt.Errorf("read standard input: %w", err)
	}
	value := strings.TrimRight(string(raw), "\r\n")
	if value == "" {
		return "", apierr.Refused("standard input carried no value")
	}
	return value, nil
}
