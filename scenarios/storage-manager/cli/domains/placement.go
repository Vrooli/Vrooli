package domains

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

func placementGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	client, base := cliapp.NewConnectHTTPClient(core)
	return cliapp.SubcommandGroup{Name: "placement", Description: "Verify storage placement without moving bytes", NeedsAPI: true, Subcommands: []cliapp.Command{{
		Name: "verify", Description: "Verify declarations for a target platform", Run: func(args []string) error {
			platform := ""
			for i := 0; i < len(args); i++ {
				if args[i] == "--platform" && i+1 < len(args) {
					platform = args[i+1]
					i++
				}
			}
			url := base + "/api/v1/placement/verify"
			if strings.TrimSpace(platform) != "" {
				url += "?platform=" + platform
			}
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("placement verify: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode >= http.StatusBadRequest {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("placement verify: %s", string(body))
			}
			_, err = io.Copy(os.Stdout, resp.Body)
			return err
		},
	}}}
}
