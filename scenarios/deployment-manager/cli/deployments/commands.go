package deployments

import (
	"encoding/json"
	"errors"
	"flag"
	"net/url"
	"strings"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

type Commands struct {
	api *cliutil.APIClient
}

func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

type DeploymentLog struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// Deployments

func (c *Commands) Deploy(args []string) error {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "dry run")
	async := fs.Bool("async", false, "async deploy")
	validateOnly := fs.Bool("validate-only", false, "validate only")
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	if *validateOnly {
		return c.Validate([]string{id})
	}
	payload := map[string]interface{}{
		"dry_run": *dryRun,
		"async":   *async,
	}
	body, err := c.api.Request("POST", "/api/v1/deploy/"+id, nil, payload)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Deployment(args []string) error {
	if len(args) == 0 {
		return errors.New("deployment subcommand is required")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "status":
		return c.deploymentStatus(rest)
	default:
		return errors.New("unknown deployment subcommand: " + sub)
	}
}

func (c *Commands) deploymentStatus(args []string) error {
	fs := flag.NewFlagSet("deployment status", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("deployment id is required")
	}
	id := remaining[0]
	body, err := c.api.Get("/api/v1/deployments/"+id, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

// Packagers

func (c *Commands) Packagers(args []string) error {
	if len(args) == 0 {
		return c.packagersList(args)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return c.packagersList(rest)
	case "discover":
		return c.packagersDiscover(rest)
	default:
		return errors.New("unknown packagers subcommand: " + sub)
	}
}

func (c *Commands) packagersList(args []string) error {
	fs := flag.NewFlagSet("packagers list", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	_, _ = cmdutil.ParseArgs(fs, args)
	body, err := c.api.Get("/api/v1/packagers", nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) packagersDiscover(args []string) error {
	fs := flag.NewFlagSet("packagers discover", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	_, _ = cmdutil.ParseArgs(fs, args)
	body, err := c.api.Request("POST", "/api/v1/packagers/discover", nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) PackageProfile(args []string) error {
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	packager := fs.String("packager", "", "packager name")
	dryRun := fs.Bool("dry-run", false, "dry run")
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	if *packager == "" {
		return errors.New("--packager is required")
	}
	id := remaining[0]
	payload := map[string]interface{}{"packager": *packager, "dry_run": *dryRun}
	body, err := c.api.Request("POST", "/api/v1/package/"+id, nil, payload)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

// Validation/Estimates

func (c *Commands) Validate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	verbose := fs.Bool("verbose", false, "verbose output")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	path := "/api/v1/profiles/" + id + "/validate"
	if *verbose {
		path += "?verbose=true"
	}
	body, err := c.api.Get(path, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) EstimateCost(args []string) error {
	fs := flag.NewFlagSet("estimate-cost", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	verbose := fs.Bool("verbose", false, "verbose output")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	path := "/api/v1/profiles/" + id + "/cost-estimate"
	if *verbose {
		path += "?verbose=true"
	}
	body, err := c.api.Get(path, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

// Logs (deployment scoped)

func (c *Commands) Logs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	level := fs.String("level", "", "log level filter")
	search := fs.String("search", "", "search term")
	format := fs.String("format", "", "output format (json|table)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	q := url.Values{}
	if *level != "" {
		q.Set("level", *level)
	}
	if *search != "" {
		q.Set("search", *search)
	}
	body, err := c.api.Get("/api/v1/logs/"+id, q)
	if err != nil {
		return err
	}
	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "table" {
		if err := printLogsTable(body); err == nil {
			return nil
		}
	}
	cmdutil.PrintByFormat(formatVal, body)
	return nil
}

func printLogsTable(body []byte) error {
	var logs []DeploymentLog
	if err := json.Unmarshal(body, &logs); err != nil {
		return err
	}
	rows := make([][]string, 0, len(logs))
	for _, entry := range logs {
		rows = append(rows, []string{
			entry.Timestamp,
			entry.Level,
			entry.Message,
		})
	}
	cmdutil.PrintTable([]string{"Timestamp", "Level", "Message"}, rows)
	return nil
}
