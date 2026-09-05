package domains

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

func declareGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	client, base := cliapp.NewConnectHTTPClient(core)
	call := func(path string, args []string) error {
		owner := ""
		for _, arg := range args {
			if !strings.HasPrefix(arg, "-") && owner == "" {
				owner = arg
			}
		}
		if owner == "" {
			return fmt.Errorf("owner is required")
		}
		req, err := http.NewRequest(http.MethodGet, base+path+"?owner="+owner, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("declare: %s", string(body))
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	return cliapp.SubcommandGroup{Name: "declare", Description: "Inspect and generate storage declarations", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "inspect", Description: "Compare declared and observed storage", Run: func(args []string) error { return call("/api/v1/declare/inspect", args) }},
		{Name: "suggest", Description: "Emit a measured storage.entries block", Run: func(args []string) error {
			owner := ""
			for _, arg := range args {
				if !strings.HasPrefix(arg, "-") && owner == "" {
					owner = arg
				}
			}
			if owner == "" {
				return fmt.Errorf("owner is required")
			}
			req, err := http.NewRequest(http.MethodGet, base+"/api/v1/declare/suggest?owner="+owner, nil)
			if err != nil {
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode >= http.StatusBadRequest {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("declare suggest: %s", string(body))
			}
			var payload struct {
				Block json.RawMessage `json:"block"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				return err
			}
			_, err = os.Stdout.Write(append(payload.Block, '\n'))
			return err
		}},
		{Name: "check", Description: "Report declaration coverage", Run: func(args []string) error {
			kind := ""
			for i := 0; i < len(args); i++ {
				if args[i] == "--kind" && i+1 < len(args) {
					kind = args[i+1]
					i++
				}
			}
			path := "/api/v1/declare/check"
			if kind != "" {
				path += "?kind=" + kind
			}
			req, err := http.NewRequest(http.MethodGet, base+path, nil)
			if err != nil {
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode >= http.StatusBadRequest {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("declare check: %s", string(body))
			}
			_, err = io.Copy(os.Stdout, resp.Body)
			return err
		}},
	}}
}
