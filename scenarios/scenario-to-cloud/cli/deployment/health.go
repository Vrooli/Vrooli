package deployment

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/protoout"
	"scenario-to-cloud/cli/internal/selector"
)

// runHealth prints the typed health observation. The verdict is the
// producer's: UNKNOWN and stale evidence are printed as such and exit 1,
// never mapped to healthy.
func runHealth(client *Client, args []string) error {
	fs := flag.NewFlagSet("deployment health", flag.ContinueOnError)
	sel := selector.Register(fs)
	jsonOutput := fs.Bool("json", false, "Output proto JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	ref, err := resolveArgs(client, fs, sel)
	if err != nil {
		return err
	}
	resp, err := client.HealthObservation(context.Background(), ref.GetId())
	if err != nil {
		return err
	}
	if *jsonOutput {
		return protoout.Print(resp)
	}
	fmt.Println(selector.Identity(ref))
	obs := resp.GetObservation()
	status := enumWord(obs.GetStatus().String(), "HEALTH_STATUS_")
	freshness := enumWord(obs.GetFreshness().String(), "FRESHNESS_")
	fmt.Printf("health: %s  freshness: %s  observed: %s  producer: %s\n", status, freshness, obs.GetObservedAt().AsTime().UTC().Format("2006-01-02T15:04:05Z"), obs.GetProducerRef())
	if obs.GetObservedReleaseDigest() != "" {
		fmt.Printf("  observed release:       %s\n", obs.GetObservedReleaseDigest())
	}
	if obs.GetObservedConfigurationDigest() != "" {
		fmt.Printf("  observed configuration: %s\n", obs.GetObservedConfigurationDigest())
	}
	if obs.GetPartial() {
		fmt.Printf("  partial: missing %s\n", strings.Join(obs.GetMissingDependencies(), ", "))
	}
	for _, check := range obs.GetChecks() {
		line := fmt.Sprintf("  %-22s %s", check.GetId(), enumWord(check.GetStatus().String(), "CHECK_STATUS_"))
		if check.GetReasonCode() != "" {
			line += " (" + check.GetReasonCode() + ")"
		}
		if check.GetDetail() != "" {
			line += " " + check.GetDetail()
		}
		fmt.Println(line)
	}
	for _, na := range obs.GetNextActions() {
		fmt.Printf("  next: %s", na.GetLabel())
		if na.GetReference() != "" {
			fmt.Printf(" (%s)", na.GetReference())
		}
		fmt.Println()
	}
	if obs.GetStatus() != healthv1.HealthStatus_HEALTH_STATUS_HEALTHY || obs.GetFreshness() != healthv1.Freshness_FRESHNESS_CURRENT {
		return apierr.Failed("deployment %s is %s (%s)", ref.GetId(), status, freshness)
	}
	return nil
}

func enumWord(name, prefix string) string {
	return strings.ToLower(strings.TrimPrefix(name, prefix))
}
