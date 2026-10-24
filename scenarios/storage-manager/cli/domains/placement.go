package domains

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type verificationRow struct {
	Kind              string `json:"kind"`
	Owner             string `json:"owner"`
	Entry             string `json:"entry"`
	Platform          string `json:"platform"`
	Applicable        bool   `json:"applicable"`
	DeclaredAbsent    bool   `json:"declared_absent"`
	SyntheticIdentity bool   `json:"synthetic_identity"`
	Path              string `json:"path,omitempty"`
	Error             string `json:"error,omitempty"`
}

func placementGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	client, base := cliapp.NewConnectHTTPClient(core)
	return cliapp.SubcommandGroup{Name: "placement", Description: "Verify storage placement without moving bytes", NeedsAPI: true, Subcommands: []cliapp.Command{{
		Name: "verify", Description: "Verify declarations for a target platform", Run: func(args []string) error {
			fs := flag.NewFlagSet("placement verify", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			platform := fs.String("platform", "", "target platform: linux, macos, or windows")
			allPlatforms := fs.Bool("all-platforms", false, "verify all supported platforms")
			jsonOutput := cliutil.JSONFlag(fs)
			if err := cliutil.ParseInterspersed(fs, args); err != nil {
				return err
			}
			platforms := []string{strings.TrimSpace(*platform)}
			if *allPlatforms {
				platforms = []string{"linux", "macos", "windows"}
			}
			matrices := make([]verificationRow, 0)
			for _, target := range platforms {
				rows, body, err := fetchVerification(client, base, target)
				if err != nil {
					return err
				}
				if *jsonOutput {
					matrices = append(matrices, rows...)
					if len(platforms) == 1 {
						_, err = os.Stdout.Write(body)
						return err
					}
					continue
				}
				renderVerificationTable(rows)
			}
			if *jsonOutput && len(platforms) > 1 {
				return json.NewEncoder(os.Stdout).Encode(matrices)
			}
			return nil
		},
	}}}
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func fetchVerification(client httpDoer, base, platform string) ([]verificationRow, []byte, error) {
	endpoint := base + "/api/v1/placement/verify"
	if strings.TrimSpace(platform) != "" {
		endpoint += "?platform=" + url.QueryEscape(platform)
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("placement verify: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("placement verify: read response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, nil, fmt.Errorf("placement verify: %s", string(body))
	}
	var rows []verificationRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, nil, fmt.Errorf("placement verify: decode response: %w", err)
	}
	return rows, body, nil
}

func renderVerificationTable(rows []verificationRow) {
	fmt.Fprintln(os.Stdout, "OWNER\tENTRY\tPLATFORM\tAPPLICABLE\tPATH")
	for _, row := range rows {
		path := row.Path
		if row.DeclaredAbsent {
			path = "<declared absent>"
		}
		if row.Error != "" {
			path = "ERROR: " + row.Error
		}
		fmt.Fprintf(os.Stdout, "%s/%s\t%s\t%s\t%t\t%s\n", row.Kind, row.Owner, row.Entry, row.Platform, row.Applicable, path)
	}
}
