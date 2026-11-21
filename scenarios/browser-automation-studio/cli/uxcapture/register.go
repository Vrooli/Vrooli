package uxcapture

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"browser-automation-studio/cli/internal/api"
	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "UX capture", Commands: []cliapp.Command{{
		Name: "ux-capture", NeedsAPI: true,
		Description: "Capture a redacted public page or named local HTML file into an ephemeral EX9 bundle",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "url", Description: "Public HTTP(S) page URL"},
			{Name: "scenario", Description: "Name of a running local scenario; omit --path to capture its main view"},
			{Name: "path", Description: "Local scenario path; defaults to /"},
			{Name: "step", Repeated: true, Description: "Scenario state step such as click:Branches; repeat to apply more than one"},
			{Name: "wait-selector", Description: "Visible scenario CSS selector required before capture; scenario captures only"},
			{Name: "evidence-profile", Description: "Optional scenario evidence profile: componenttests"},
			{Name: "region-selector", Description: "CSS region for the componenttests crop and style sample; scenario captures only"},
			{Name: "file", Description: "Local .html/.htm file; contents are sent to the local BAS API (maximum 2 MiB)"},
			{Name: "width", Description: "Viewport width (320–2560; default 1280)"},
			{Name: "height", Description: "Viewport height (240–1600; default 720)"},
			{Name: "theme", Description: "Color scheme: light, dark, or no-preference (default light)"},
		}},
		RunCtx: func(rc cliapp.RunContext) error {
			urlValue, fileValue, scenario := strings.TrimSpace(rc.Flag("url")), strings.TrimSpace(rc.Flag("file")), strings.TrimSpace(rc.Flag("scenario"))
			provided := 0
			for _, value := range []string{urlValue, fileValue, scenario} {
				if value != "" {
					provided++
				}
			}
			if provided != 1 {
				return fmt.Errorf("provide exactly one of --url, --file, or --scenario")
			}
			width, err := parseDimension(rc.Flag("width"))
			if err != nil {
				return fmt.Errorf("width: %w", err)
			}
			height, err := parseDimension(rc.Flag("height"))
			if err != nil {
				return fmt.Errorf("height: %w", err)
			}
			theme := strings.TrimSpace(rc.Flag("theme"))
			if theme != "" && theme != "light" && theme != "dark" && theme != "no-preference" {
				return fmt.Errorf("theme must be light, dark, or no-preference")
			}
			if theme == "" {
				theme = "light"
			}
			waitSelector, err := validateWaitSelector(scenario, rc.Flag("wait-selector"))
			if err != nil {
				return err
			}
			evidenceProfile := strings.TrimSpace(rc.Flag("evidence-profile"))
			if evidenceProfile != "" && evidenceProfile != "componenttests" {
				return fmt.Errorf("--evidence-profile must be componenttests")
			}
			if evidenceProfile != "" && scenario == "" {
				return fmt.Errorf("--evidence-profile is available for --scenario captures only")
			}
			regionSelector, err := validateRegionSelector(scenario, evidenceProfile, rc.Flag("region-selector"))
			if err != nil {
				return err
			}
			var endpoint string
			var request any
			if fileValue != "" {
				ext := strings.ToLower(filepath.Ext(fileValue))
				if ext != ".html" && ext != ".htm" {
					return fmt.Errorf("--file must name an .html or .htm file")
				}
				f, err := os.Open(fileValue)
				if err != nil {
					return err
				}
				defer f.Close()
				content, err := io.ReadAll(io.LimitReader(f, 2*1024*1024+1))
				if err != nil {
					return err
				}
				if len(content) == 0 || len(content) > 2*1024*1024 {
					return fmt.Errorf("HTML file must be between 1 byte and 2 MiB")
				}
				request = map[string]any{"filename": filepath.Base(fileValue), "html": string(content), "width": width, "height": height, "theme": theme}
				endpoint = "/ux-capture/file"
			} else if scenario != "" {
				request = buildScenarioRequest(scenario, strings.TrimSpace(rc.Flag("path")), rc.FlagValues("step"), width, height, theme, waitSelector, evidenceProfile, regionSelector)
				endpoint = "/ux-capture/scenario"
			} else {
				request = map[string]any{"url": urlValue, "width": width, "height": height, "theme": theme}
				endpoint = "/ux-capture/public"
			}
			// api.Do adds the scenario's /api/v1 prefix; a prefixed path here
			// became /api/v1/api/v1/... and every capture returned 404.
			body, err := json.Marshal(request)
			if err != nil {
				return err
			}
			status, response, err := api.Do(ctx, http.MethodPost, endpoint, nil, body, nil)
			if err != nil {
				return err
			}
			if status < 200 || status >= 300 {
				return fmt.Errorf("ux capture failed (HTTP %d): %s", status, string(response))
			}
			_, err = fmt.Fprintln(rc.Stdout(), string(response))
			return err
		},
	}}}
}

func parseDimension(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func validateWaitSelector(scenario, value string) (string, error) {
	selector := strings.TrimSpace(value)
	if len(selector) > 512 {
		return "", fmt.Errorf("--wait-selector must be at most 512 characters")
	}
	if selector != "" && scenario == "" {
		return "", fmt.Errorf("--wait-selector is available for --scenario captures only")
	}
	return selector, nil
}

func validateRegionSelector(scenario, profile, value string) (string, error) {
	selector := strings.TrimSpace(value)
	if len(selector) > 256 || strings.HasPrefix(selector, "//") {
		return "", fmt.Errorf("--region-selector must be a CSS selector up to 256 characters")
	}
	if selector != "" && (scenario == "" || profile != "componenttests") {
		return "", fmt.Errorf("--region-selector requires --scenario and --evidence-profile componenttests")
	}
	return selector, nil
}

func buildScenarioRequest(scenario, route string, steps []string, width, height int, theme, waitSelector, evidenceProfile, regionSelector string) map[string]any {
	request := map[string]any{
		"scenario": scenario,
		"route":    route,
		"steps":    steps,
		"width":    width,
		"height":   height,
		"theme":    theme,
	}
	if waitSelector != "" {
		request["wait_selector"] = waitSelector
	}
	if evidenceProfile != "" {
		request["evidence_profile"] = evidenceProfile
	}
	if regionSelector != "" {
		request["region_selector"] = regionSelector
	}
	return request
}
