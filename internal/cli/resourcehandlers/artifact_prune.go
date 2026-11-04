package resourcehandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/resources"
)

// resourceArtifactPruneHandler previews artifact version retention by default
// and deletes only with --apply.
func resourceArtifactPruneHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		parsed, err := commandtree.ParseArgs("resource acquisition prune", resourcecli.AcquisitionCommandHelpText(resourcecli.AcquisitionCommandPrune), resourcecli.AcquisitionPruneArgs(), args)
		if err != nil {
			if commandtree.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return err
		}
		if len(parsed.Positionals) > 1 {
			return rootcli.UsageErrorf("resource acquisition prune", "resource acquisition prune accepts at most one resource name")
		}
		opts := resources.ArtifactRetentionOptions{KeepPrevious: resources.DefaultArtifactKeepPrevious, Apply: parsed.HasFlag("--apply")}
		if parsed.HasFlag("--keep-previous") {
			keep, err := strconv.Atoi(parsed.FlagValue("--keep-previous"))
			if err != nil || keep < 0 {
				return rootcli.UsageErrorf("resource acquisition prune", "--keep-previous requires a non-negative integer")
			}
			opts.KeepPrevious = keep
		}
		names := make([]string, 0, 1)
		if len(parsed.Positionals) == 1 && strings.TrimSpace(parsed.Positionals[0]) != "" {
			names = append(names, strings.TrimSpace(parsed.Positionals[0]))
		} else {
			items, err := controller.Discover()
			if err != nil {
				return err
			}
			for _, item := range items {
				manifest, err := controller.LoadManifest(filepath.Join(controller.Root, "resources", item.Name, "resource.json"))
				if err == nil && manifest.ManagedService != nil && manifest.ManagedService.Acquisition != nil {
					names = append(names, item.Name)
				}
			}
		}
		plans := make([]resources.ArtifactRetentionPlan, 0, len(names))
		for _, name := range names {
			plan, err := controller.PruneManagedServiceArtifacts(name, opts)
			if err != nil {
				return err
			}
			if len(plan.Versions) > 0 {
				plans = append(plans, plan)
			}
		}
		if parsed.HasFlag("--json") {
			return json.NewEncoder(deps.Stdout(ctx)).Encode(plans)
		}
		return writeArtifactRetentionPlans(deps.Stdout(ctx), plans, opts.Apply)
	}
}

func writeArtifactRetentionPlans(w io.Writer, plans []resources.ArtifactRetentionPlan, applied bool) error {
	table := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	_, _ = fmt.Fprintln(table, "RESOURCE\tVERSION\tSIZE\tDECISION\tREASON")
	var total int64
	for _, plan := range plans {
		for _, version := range plan.Versions {
			decision := "keep"
			if !version.Keep {
				decision = "remove"
				if version.Removed {
					decision = "removed"
				}
			}
			_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", plan.Resource, version.Version, artifactBytes(version.Bytes), decision, version.Reason)
		}
		if applied {
			total += plan.RemovedBytes
		} else {
			total += plan.ReclaimableBytes
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}
	if applied {
		_, err := fmt.Fprintf(w, "removed %s of superseded artifact versions\n", artifactBytes(total))
		return err
	}
	_, err := fmt.Fprintf(w, "dry run: %s reclaimable; re-run with --apply to remove\n", artifactBytes(total))
	return err
}

func artifactBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	value, suffix := float64(n), "KMGTP"
	i := -1
	for value >= unit && i < len(suffix)-1 {
		value /= unit
		i++
	}
	return fmt.Sprintf("%.1f%ciB", value, suffix[i])
}
