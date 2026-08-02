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
	inventory := func() error {
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/storage/inventory", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("storage inventory: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("storage inventory: %s", string(body))
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	history := func() error {
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/census/history", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("storage census history: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("storage census history: %s", string(body))
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	retentionOwners := func() error {
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/retention/owners", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("storage retention owners: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("storage retention owners: %s", string(body))
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	placement := func() error {
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/placement", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("storage placement: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("storage placement: %s", string(body))
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	placementAudit := func() error {
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/placement/audit", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("storage placement audit: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("storage placement audit: %s", string(body))
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	adoption := func() error {
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/adoption", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("storage adoption: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("storage adoption: %s", string(body))
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	infraHealth := func() error {
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/infra-health/storage", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("storage infra health: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("storage infra health: %s", string(body))
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	return cliapp.SubcommandGroup{Name: "storage", Description: "Inspect declared, attributed, and unattributed storage", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "status", Description: "Show the current closed storage accounting", Run: func([]string) error { return read() }},
		{Name: "census", Description: "Run a read-only storage census", Run: func([]string) error { return read() }},
		{Name: "history", Description: "Show persisted census snapshots and growth observations", Run: func([]string) error { return history() }},
		{Name: "retention", Description: "Show retention budgets across every owner kind", Run: func([]string) error { return retentionOwners() }},
		{Name: "placement", Description: "Show resolved cross-platform storage placement", Run: func([]string) error { return placement() }},
		{Name: "placement-audit", Description: "Show placement migration audit events", Run: func([]string) error { return placementAudit() }},
		{Name: "adoption", Description: "Show declaration adoption coverage and suggestions", Run: func([]string) error { return adoption() }},
		{Name: "infra-health", Description: "Show persisted storage infra-health signal", Run: func([]string) error { return infraHealth() }},
		{Name: "inventory", Description: "List every storage owner and declaration", Run: func([]string) error { return inventory() }},
	}}
}
