package deployment

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/protoout"
	"scenario-to-cloud/cli/internal/selector"
)

// RunPublication executes the publication command group (request, apply,
// status) over the Evidence service.
func RunPublication(client *Client, args []string) error {
	if len(args) == 0 {
		return printPublicationUsage()
	}
	switch args[0] {
	case "request":
		return runPublicationRequest(client, args[1:])
	case "apply":
		return runPublicationApply(client, args[1:])
	case "status":
		return runPublicationStatus(client, args[1:])
	case "help", "-h", "--help":
		return printPublicationUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud publication help' for usage", args[0])
	}
}

func printPublicationUsage() error {
	fmt.Println(`Usage: scenario-to-cloud publication <command> <selector> [flags]

Governed publication: a release is published only against an approved
review bound to the exact deployment, release and plan.

Commands:
  request <selector> --request-key <key> --profile <id> --candidate-commit <sha> --artifact-digest <digest> [--channel <name>]
                                        Open a publication and prepare its review
  apply <selector> --request-key <key> --review-ref <ref> --plan-digest <digest>
                                        Activate the approved publication (re-checks approval server-side)
  status <selector> --request-key <key> Publication standing

Selector: ` + selector.Usage)
	return nil
}

func runPublicationRequest(client *Client, args []string) error {
	fs := flag.NewFlagSet("publication request", flag.ContinueOnError)
	sel := selector.Register(fs)
	requestKey := fs.String("request-key", "", "Publication request key (required)")
	profile := fs.String("profile", "", "Deployment profile id")
	candidateCommit := fs.String("candidate-commit", "", "Candidate commit")
	artifactDigest := fs.String("artifact-digest", "", "Candidate artifact digest")
	channel := fs.String("channel", "", "Release channel")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*requestKey) == "" {
		return apierr.Refused("--request-key is required")
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	pub, err := client.RequestPublication(context.Background(), &evidencev1.PublicationRequest{
		DeploymentId: ref.GetId(), RequestKey: *requestKey,
		Identity: &evidencev1.ReviewIdentity{
			ScenarioId: ref.GetScenarioId(), Environment: ref.GetEnvironment(), ProfileId: *profile,
			CandidateCommit: *candidateCommit, ArtifactDigest: *artifactDigest, Channel: *channel,
		},
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(pub)
	}
	fmt.Println(selector.Identity(ref))
	return writePublication(pub)
}

func runPublicationApply(client *Client, args []string) error {
	fs := flag.NewFlagSet("publication apply", flag.ContinueOnError)
	sel := selector.Register(fs)
	requestKey := fs.String("request-key", "", "Publication request key (required)")
	reviewRef := fs.String("review-ref", "", "Approved review reference (required)")
	planDigest := fs.String("plan-digest", "", "Reviewed plan digest (required)")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	for name, value := range map[string]string{"--request-key": *requestKey, "--review-ref": *reviewRef, "--plan-digest": *planDigest} {
		if strings.TrimSpace(value) == "" {
			return apierr.Refused("%s is required", name)
		}
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	pub, err := client.ApplyPublication(context.Background(), &evidencev1.ApplyPublicationRequest{DeploymentId: ref.GetId(), RequestKey: *requestKey, ReviewRef: *reviewRef, PlanDigest: *planDigest})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(pub)
	}
	fmt.Println(selector.Identity(ref))
	return writePublication(pub)
}

func runPublicationStatus(client *Client, args []string) error {
	fs := flag.NewFlagSet("publication status", flag.ContinueOnError)
	sel := selector.Register(fs)
	requestKey := fs.String("request-key", "", "Publication request key (required)")
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*requestKey) == "" {
		return apierr.Refused("--request-key is required")
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	pub, err := client.GetPublication(context.Background(), ref.GetId(), *requestKey)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(pub)
	}
	fmt.Println(selector.Identity(ref))
	return writePublication(pub)
}

// writePublication prints the standing and maps its state to the exit
// contract: published/approved states exit 0, refused/failed exit 1,
// anything still in flight exits 3.
func writePublication(pub *evidencev1.Publication) error {
	fmt.Printf("publication: %s  state: %s\n", pub.GetId(), pub.GetState())
	fmt.Printf("  request key:    %s\n", pub.GetRequestKey())
	if pub.GetReviewRef() != "" {
		fmt.Printf("  review ref:     %s\n", pub.GetReviewRef())
	}
	if pub.GetReleaseDigest() != "" {
		fmt.Printf("  release digest: %s\n", pub.GetReleaseDigest())
	}
	if pub.GetPlanDigest() != "" {
		fmt.Printf("  plan digest:    %s\n", pub.GetPlanDigest())
	}
	if pub.GetOperationId() != "" {
		fmt.Printf("  operation:      %s\n", pub.GetOperationId())
	}
	if r := pub.GetRefusal(); r != nil && r.GetCode() != "" {
		fmt.Printf("  refusal:        %s (%s)\n", r.GetReason(), r.GetCode())
	}
	if t := pub.GetTargetReceipt(); t != nil && t.GetActiveRelease() != "" {
		fmt.Printf("  target receipt: active %s previous %s fence %d\n", t.GetActiveRelease(), t.GetPreviousRelease(), t.GetFence())
	}
	switch pub.GetState() {
	case "published", "approved", "review_prepared":
		return nil
	case "refused", "failed":
		return apierr.Failed("publication %s %s", pub.GetId(), pub.GetState())
	default:
		return apierr.Pending("publication %s is %s", pub.GetId(), pub.GetState())
	}
}
