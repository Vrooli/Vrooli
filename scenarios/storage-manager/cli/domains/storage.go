package domains

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

func storageGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	client, base := cliapp.NewConnectHTTPClient(core)
	read := func(args []string, force bool) error {
		for _, arg := range args {
			if strings.EqualFold(arg, "--force") {
				force = true
			}
		}
		path := base + "/api/v1/census"
		if force {
			path += "?force=true"
		}
		req, err := http.NewRequest(http.MethodGet, path, nil)
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
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/census/history?detail=summary", nil)
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
	growth := func(args []string) error {
		window := "24h"
		for i := 0; i < len(args); i++ {
			if args[i] == "--window" && i+1 < len(args) {
				window = args[i+1]
				i++
				continue
			}
			if strings.HasPrefix(args[i], "--window=") {
				window = strings.TrimPrefix(args[i], "--window=")
			}
		}
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/storage/growth?root=/&window="+url.QueryEscape(window), nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("storage growth: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("storage growth: %s", string(body))
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
		// Infra-health is the host-wide signal; select the device-root census
		// explicitly rather than falling back to the repository-root history.
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/infra-health/storage?root=/", nil)
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
	writers := func(args []string) error {
		top := "10"
		for i := 0; i < len(args); i++ {
			if args[i] == "--top" && i+1 < len(args) { top = args[i+1]; i++ }
			if strings.HasPrefix(args[i], "--top=") { top = strings.TrimPrefix(args[i], "--top=") }
		}
		req, err := http.NewRequest(http.MethodGet, base+"/api/v1/storage/writers?top="+url.QueryEscape(top), nil)
		if err != nil { return err }
		resp, err := client.Do(req)
		if err != nil { return fmt.Errorf("storage writers: %w", err) }
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest { body, _ := io.ReadAll(resp.Body); return fmt.Errorf("storage writers: %s", string(body)) }
		_, err = io.Copy(os.Stdout, resp.Body)
		return err
	}
	return cliapp.SubcommandGroup{Name: "storage", Description: "Inspect declared, attributed, and unattributed storage", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "status", Description: "Show the latest persisted storage accounting", Run: func(args []string) error { return read(args, false) }},
		{Name: "census", Description: "Run a read-only storage census", Run: func(args []string) error { return read(args, true) }},
		{Name: "history", Description: "Show persisted census snapshots and growth observations", Run: func([]string) error { return history() }},
		{Name: "growth", Description: "Rank per-owner storage velocity and ceiling projections", Run: growth},
		{Name: "retention", Description: "Show retention budgets across every owner kind", Run: func([]string) error { return retentionOwners() }},
		{Name: "placement", Description: "Show resolved cross-platform storage placement", Run: func([]string) error { return placement() }},
		{Name: "placement-audit", Description: "Show placement migration audit events", Run: func([]string) error { return placementAudit() }},
		{Name: "adoption", Description: "Show declaration adoption coverage and suggestions", Run: func([]string) error { return adoption() }},
		{Name: "infra-health", Description: "Show persisted storage infra-health signal", Run: func([]string) error { return infraHealth() }},
		{Name: "writers", Description: "Rank persisted governed-root writer rates", Run: writers},
		{Name: "inventory", Description: "List every storage owner and declaration", Run: func([]string) error { return inventory() }},
	}}
}
