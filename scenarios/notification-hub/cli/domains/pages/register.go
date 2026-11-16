// Package pages is the Hub CLI's page tools: an agent or a page writer checks
// a decision page with the Hub's own packet check (`pages check`, the check
// behind `ask --dry-run`) and photographs it as the Hub frames it, at phone
// width in light and dark (`pages shoot`). A Luna writer validates its page
// with these two commands. The shared decision-page writer itself
// (packages/api-core/decisionpage) runs outside the Hub.
package pages

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/vrooli/api-core/decisionpage"
	"github.com/vrooli/cli-core/cliapp"

	"notification-hub/cli/domains/conversations"
)

// GroupName is the manifest group these commands belong to.
const GroupName = "pages"

// Register builds the pages group from cli/manifest.json.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	return cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"check": conversations.PageCheck(core),
		"shoot": shoot,
	})
}

func shoot(ctx cliapp.RunContext) error {
	page, err := filepath.Abs(ctx.Positional("page"))
	if err != nil {
		return err
	}
	out := ctx.Flag("out")
	if out == "" {
		out = filepath.Join(filepath.Dir(page), decisionpage.ShotsDir)
	}
	// Chrome opens the framed page by a file URL, which must be absolute.
	if out, err = filepath.Abs(out); err != nil {
		return err
	}
	shots := decisionpage.DefaultShots
	if ctx.BoolFlag("narrow") {
		shots = append(shots, decisionpage.Shot{Width: 320, Theme: "light"}, decisionpage.Shot{Width: 320, Theme: "dark"})
	}
	paths, err := decisionpage.Shoot(context.Background(), page, out, filepath.Join(decisionpage.RepoRoot(), decisionpage.KitRelDir), shots)
	for _, p := range paths {
		fmt.Fprintln(ctx.Stdout(), p)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout(), "Open every screenshot and look at it, in light and dark, before you finish.")
	return nil
}
