package validate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	repocontract "github.com/vrooli/repo-contract-go"
)

var schemaOwnerRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// validateOwner checks source files directly so a malformed owner is still
// diagnosable before generated descriptors exist. Buf supplies the parser and
// import validation, while this command supplies the stable owner attribution.
func (h *handlers) validateOwner(ctx cliapp.RunContext) error {
	owner := strings.TrimSpace(ctx.Positional("owner"))
	if !schemaOwnerRE.MatchString(owner) {
		return fmt.Errorf("invalid schema owner %q: use a lowercase scenario-style identifier", owner)
	}
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return fmt.Errorf("find repository root: %w", err)
	}
	protoRoot := filepath.Join(root, "packages", "proto")
	ownerRoot := filepath.Join(protoRoot, "schemas", owner)
	if info, err := os.Stat(ownerRoot); err != nil || !info.IsDir() {
		return fmt.Errorf("schema owner %q was not found at %s", owner, filepath.ToSlash(filepath.Join("packages", "proto", "schemas", owner)))
	}
	if _, err := exec.LookPath("buf"); err != nil {
		return fmt.Errorf("find buf: %w", err)
	}

	checkCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(checkCtx, "buf", "build", "--path", filepath.ToSlash(filepath.Join("schemas", owner)))
	cmd.Dir = protoRoot
	out, runErr := cmd.CombinedOutput()
	diagnostic := strings.TrimSpace(string(out))
	if runErr != nil {
		if diagnostic == "" {
			diagnostic = runErr.Error()
		}
		_ = ctx.RenderList(cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Schema owner %s: invalid", owner)},
			ResultsHeading: "Owner diagnostics",
			Results:        []string{diagnostic},
			RetrievalHints: []string{"Fix the reported source file, then rerun `proto-health validate owner " + owner + "`"},
		})
		return fmt.Errorf("schema owner %s is invalid: %s", owner, diagnostic)
	}

	if err := ctx.RenderList(cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Schema owner %s: valid", owner)},
		ResultsHeading: "Owner diagnostics",
		Results:        []string{"buf build passed for schemas/" + owner},
		RetrievalHints: []string{"Run `proto-health validate scenario <name>` for descriptor-backed policy checks"},
	}); err != nil {
		return err
	}
	return nil
}
