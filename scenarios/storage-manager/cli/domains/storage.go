package domains

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

func storageGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	client, base := cliapp.NewConnectHTTPClient(core)
	read := func() error {
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/census", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("storage census: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("storage census: %s", string(body))
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	return cliapp.SubcommandGroup{Name: "storage", Description: "Inspect declared, attributed, and unattributed storage", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "status", Description: "Show the current closed storage accounting", Run: func([]string) error { return read() }},
		{Name: "census", Description: "Run a read-only storage census", Run: func([]string) error { return read() }},
	}}
}
