package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/activityedge"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

var buildFingerprint, buildTimestamp, buildSourceRoot = "unknown", "unknown", ""

func main() {
	app, err := newApp()
	if err == nil {
		err = app.CLI.Run(os.Args[1:])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func newApp() (*cliapp.ResourceApp, error) {
	env := cliapp.StandardResourceEnv("songgen", cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{Name: "songgen", Version: "0.1.0",
		Description:       "Isolated local song generation and owned model retention",
		SourceRootEnvVars: env.SourceRootEnvVars, ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint: buildFingerprint, BuildTimestamp: buildTimestamp, BuildSourceRoot: buildSourceRoot})
	if err != nil {
		return nil, err
	}
	edge, err := activityedge.ForResource("songgen")
	if err != nil {
		return nil, err
	}
	app.SetCommandsWithSubgroups(append(app.StandardLifecycleCommands(), cliapp.CommandGroup{
		Title: "Capacity", Commands: []cliapp.Command{activityedge.Command(edge)},
	}), []cliapp.SubcommandGroup{
		companion.LifecycleCapacityCommands(companion.LifecycleVerbsConfig{Resource: "songgen", Steps: []companion.LifecycleStep{{Label: "isolated", Action: "restart", Description: "Apply the bounded isolated-worker profile"}}}),
		{Name: "model-cache", Description: "Preview, pin, or apply last-used model retention", Subcommands: []cliapp.Command{
			{Name: "preview", Description: "Read eligible whole-model bundles; never delete", Run: func(args []string) error { return cacheCommand("preview", args) }},
			{Name: "apply", Description: "Apply an exact saved preview; active or changed bundles are refused", Run: func(args []string) error { return cacheCommand("apply", args) }},
			{Name: "pin", Description: "Retain a declared model independently of its last-used age", Run: func(args []string) error { return cacheCommand("pin", args) }},
		}},
	})
	return app, nil
}

func cacheCommand(mode string, args []string) error {
	fs := flag.NewFlagSet("model-cache "+mode, flag.ContinueOnError)
	file := fs.String("preview", "", "Exact preview JSON file (required for apply)")
	profile := fs.String("profile", "", "Profile to pin")
	pinned := fs.Bool("pinned", true, "Pin value; use --pinned=false to unpin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	request := map[string]any{"dry_run": true}
	switch mode {
	case "apply":
		if *file == "" {
			return fmt.Errorf("--preview FILE is required")
		}
		stream, err := os.Open(*file)
		if err != nil {
			return err
		}
		defer stream.Close()
		data, err := io.ReadAll(io.LimitReader(stream, 65537))
		if err != nil {
			return err
		}
		var preview any
		if len(data) > 65536 {
			return fmt.Errorf("preview exceeds 64KiB")
		}
		if err := json.Unmarshal(data, &preview); err != nil {
			return err
		}
		request["dry_run"], request["preview"] = false, preview
	case "pin":
		if *profile == "" {
			return fmt.Errorf("--profile is required")
		}
		request["pin"] = map[string]bool{*profile: *pinned}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	endpoint := os.Getenv("SONGGEN_URL")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8897"
	}
	if endpoint != "http://127.0.0.1:8897" && endpoint != "http://localhost:8897" {
		return fmt.Errorf("SONGGEN_URL must be the declared local resource origin")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/v1/models/retention", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("resource redirects refused") }}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil {
		return err
	}
	if len(data) > 65536 {
		return fmt.Errorf("resource response exceeds 64KiB")
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("model retention %s: %s", response.Status, data)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}
