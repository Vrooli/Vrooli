package profiles

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

type Profile struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Scenario string                 `json:"scenario"`
	Tiers    []int                  `json:"tiers"`
	Swaps    map[string]string      `json:"swaps"`
	Secrets  map[string]interface{} `json:"secrets"`
	Settings map[string]interface{} `json:"settings"`
	Version  int                    `json:"version"`
}

type ProfileHistory struct {
	ProfileID string    `json:"profile_id"`
	Versions  []Profile `json:"versions"`
}

type Commands struct {
	api *cliutil.APIClient
}

func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

// Profiles

func (c *Commands) List(args []string) error {
	fs := flag.NewFlagSet("profiles", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json|table)")
	_, _ = cmdutil.ParseArgs(fs, args)
	body, err := c.api.Get("/api/v1/profiles", nil)
	if err != nil {
		return err
	}
	var profiles []Profile
	if err := json.Unmarshal(body, &profiles); err != nil {
		return fmt.Errorf("parse profiles: %w", err)
	}
	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "table" {
		rows := make([][]string, 0, len(profiles))
		for _, p := range profiles {
			rows = append(rows, []string{
				p.ID,
				p.Scenario,
				intSliceToString(p.Tiers),
				fmt.Sprintf("%d swaps", len(p.Swaps)),
				fmt.Sprintf("v%d", versionNumber(p)),
			})
		}
		cmdutil.PrintTable([]string{"ID", "Scenario", "Tiers", "Swaps", "Version"}, rows)
		return nil
	}
	cmdutil.PrintByFormat(formatVal, body)
	return nil
}

// Profile subcommands

func (c *Commands) Create(args []string) error {
	fs := flag.NewFlagSet("profile create", flag.ContinueOnError)
	tier := fs.String("tier", "2", "deployment tier")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) < 2 {
		return errors.New("usage: profile create <name> <scenario> [--tier <tier>]")
	}
	payload := Profile{
		Name:     remaining[0],
		Scenario: remaining[1],
		Tiers:    []int{cmdutil.TierToNumber(*tier)},
		Swaps:    map[string]string{},
		Secrets:  map[string]interface{}{},
		Settings: map[string]interface{}{},
	}
	body, err := c.api.Request("POST", "/api/v1/profiles", nil, payload)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (c *Commands) Show(args []string) error {
	fs := flag.NewFlagSet("profile show", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	body, err := c.api.Get("/api/v1/profiles/"+id, nil)
	if err != nil {
		return err
	}
	if _, err := decodeProfile(body); err != nil {
		return fmt.Errorf("parse profile: %w", err)
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Delete(args []string) error {
	if len(args) == 0 {
		return errors.New("profile id is required")
	}
	id := args[0]
	body, err := c.api.Request("DELETE", "/api/v1/profiles/"+id, nil, nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (c *Commands) Export(args []string) error {
	fs := flag.NewFlagSet("profile export", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	outputPath := fs.String("output", "", "output file")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	body, err := c.api.Get("/api/v1/profiles/"+id, nil)
	if err != nil {
		return err
	}
	profile, err := decodeProfile(body)
	if err != nil {
		return fmt.Errorf("parse profile: %w", err)
	}
	if *outputPath != "" {
		data, marshalErr := json.MarshalIndent(profile, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("encode profile: %w", marshalErr)
		}
		if err := os.WriteFile(*outputPath, data, 0o644); err != nil {
			return fmt.Errorf("write export: %w", err)
		}
		fmt.Printf("Profile exported to %s\n", *outputPath)
		return nil
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Import(args []string) error {
	fs := flag.NewFlagSet("profile import", flag.ContinueOnError)
	name := fs.String("name", "", "override profile name")
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("file path is required")
	}
	path := remaining[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	var payload Profile
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}
	if *name != "" {
		payload.Name = *name
	}
	normalizeProfile(&payload)
	body, err := c.api.Request("POST", "/api/v1/profiles", nil, payload)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Update(args []string) error {
	fs := flag.NewFlagSet("profile update", flag.ContinueOnError)
	tier := fs.String("tier", "", "deployment tier")
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]

	body, err := c.api.Get("/api/v1/profiles/"+id, nil)
	if err != nil {
		return err
	}
	profile, err := decodeProfile(body)
	if err != nil {
		return fmt.Errorf("parse profile: %w", err)
	}
	if *tier != "" {
		profile.Tiers = []int{cmdutil.TierToNumber(*tier)}
	}
	payload := extractUpdatableProfileFields(profile)
	updated, err := c.api.Request("PUT", "/api/v1/profiles/"+id, nil, payload)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, updated)
	return nil
}

func (c *Commands) Set(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: profile set <id> <key> [value]")
	}
	id := args[0]
	key := args[1]
	rest := args[2:]

	body, err := c.api.Get("/api/v1/profiles/"+id, nil)
	if err != nil {
		return err
	}
	profile, err := decodeProfile(body)
	if err != nil {
		return fmt.Errorf("parse profile: %w", err)
	}

	switch key {
	case "tier":
		if len(rest) == 0 {
			return errors.New("tier value is required")
		}
		profile.Tiers = []int{cmdutil.TierToNumber(rest[0])}
	case "env":
		if len(rest) < 2 {
			return errors.New("usage: profile set <id> env <key> <value>")
		}
		envKey := rest[0]
		envVal := rest[1]
		if profile.Settings == nil {
			profile.Settings = map[string]interface{}{}
		}
		envMap := ensureNestedMap(profile.Settings, "env")
		envMap[envKey] = envVal
	default:
		if len(rest) == 0 {
			return errors.New("value is required")
		}
		if profile.Settings == nil {
			profile.Settings = map[string]interface{}{}
		}
		profile.Settings[key] = rest[0]
	}

	payload := extractUpdatableProfileFields(profile)
	updated, err := c.api.Request("PUT", "/api/v1/profiles/"+id, nil, payload)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(updated)
	return nil
}

func (c *Commands) Swap(args []string) error {
	if len(args) < 3 {
		return errors.New("usage: profile swap <id> <add|remove> <from> [to]")
	}
	id := args[0]
	action := args[1]
	from := args[2]
	to := ""
	if len(args) > 3 {
		to = args[3]
	}

	body, err := c.api.Get("/api/v1/profiles/"+id, nil)
	if err != nil {
		return err
	}
	profile, err := decodeProfile(body)
	if err != nil {
		return fmt.Errorf("parse profile: %w", err)
	}

	switch action {
	case "add", "set":
		if to == "" {
			return errors.New("target dependency is required")
		}
		profile.Swaps[from] = to
	case "remove":
		delete(profile.Swaps, from)
	default:
		return fmt.Errorf("unknown swap action: %s", action)
	}

	payload := extractUpdatableProfileFields(profile)
	updated, err := c.api.Request("PUT", "/api/v1/profiles/"+id, nil, payload)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(updated)
	return nil
}

func (c *Commands) Versions(args []string) error {
	fs := flag.NewFlagSet("profile versions", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	body, err := c.api.Get("/api/v1/profiles/"+id+"/versions", nil)
	if err != nil {
		return err
	}
	var history ProfileHistory
	if err := json.Unmarshal(body, &history); err != nil {
		return fmt.Errorf("parse version history: %w", err)
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Analyze(args []string) error {
	fs := flag.NewFlagSet("profile analyze", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	body, err := c.api.Get("/api/v1/profiles/"+id, nil)
	if err != nil {
		return err
	}
	if _, err := decodeProfile(body); err != nil {
		return fmt.Errorf("parse profile: %w", err)
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Save(args []string) error {
	if len(args) == 0 {
		return errors.New("profile id is required")
	}
	id := args[0]
	body, err := c.api.Get("/api/v1/profiles/"+id, nil)
	if err != nil {
		return err
	}

	current, err := decodeProfile(body)
	if err != nil {
		return fmt.Errorf("parse profile: %w", err)
	}

	payload := extractUpdatableProfileFields(current)
	updated, err := c.api.Request("PUT", "/api/v1/profiles/"+id, nil, payload)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(updated)
	return nil
}

func (c *Commands) Diff(args []string) error {
	fs := flag.NewFlagSet("profile diff", flag.ContinueOnError)
	format := fs.String("format", "json", "output format")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]

	history, err := fetchVersionHistory(c.api, id)
	if err != nil {
		return err
	}
	if len(history.Versions) < 2 {
		return fmt.Errorf("profile '%s' has only %d version(s); nothing to diff", id, len(history.Versions))
	}

	current := history.Versions[0]
	previous := history.Versions[1]
	diff := computeProfileDiff(previous, current)

	if len(diff) == 0 {
		fmt.Printf("No differences between versions %d and %d\n", versionNumber(previous), versionNumber(current))
		return nil
	}

	payload := map[string]interface{}{
		"profile_id":  history.ProfileID,
		"from":        versionNumber(previous),
		"to":          versionNumber(current),
		"change_set":  diff,
		"description": "Computed locally from version history",
	}

	if strings.ToLower(*format) == "json" {
		cmdutil.PrintJSONMap(payload)
		return nil
	}
	cmdutil.PrintJSONMap(payload)
	return nil
}

func (c *Commands) Rollback(args []string) error {
	fs := flag.NewFlagSet("profile rollback", flag.ContinueOnError)
	target := fs.String("version", "", "version number")
	toVersion := fs.String("to-version", "", "version number")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	selected := strings.TrimSpace(*target)
	if selected == "" {
		selected = strings.TrimSpace(*toVersion)
	}
	if selected == "" {
		return errors.New("version is required (use --version or --to-version)")
	}

	history, err := fetchVersionHistory(c.api, id)
	if err != nil {
		return err
	}

	versionMap, ok := findVersion(history.Versions, selected)
	if !ok {
		return fmt.Errorf("version %s not found for profile '%s'", selected, id)
	}

	payload := extractUpdatableProfileFields(versionMap)
	body, err := c.api.Request("PUT", "/api/v1/profiles/"+id, nil, payload)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	result := map[string]interface{}{
		"profile_id": id,
		"rolled_to":  selected,
		"applied":    payload,
	}
	if len(body) > 0 {
		result["server_response"] = json.RawMessage(body)
	}

	cmdutil.PrintJSONMap(result)
	return nil
}

// Secrets (profile-scoped)

func (c *Commands) Secrets(args []string) error {
	if len(args) == 0 {
		return errors.New("secrets subcommand is required")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "identify":
		return c.secretsIdentify(rest)
	case "template":
		return c.secretsTemplate(rest)
	case "validate":
		return c.secretsValidate(rest)
	default:
		return fmt.Errorf("unknown secrets subcommand: %s", sub)
	}
}

func (c *Commands) secretsIdentify(args []string) error {
	fs := flag.NewFlagSet("secrets identify", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	body, err := c.api.Get("/api/v1/profiles/"+id+"/secrets", nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) secretsTemplate(args []string) error {
	fs := flag.NewFlagSet("secrets template", flag.ContinueOnError)
	format := fs.String("format", "env", "template format (env|json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	path := fmt.Sprintf("/api/v1/profiles/%s/secrets/template?format=%s", id, *format)
	body, err := c.api.Get(path, nil)
	if err != nil {
		return err
	}
	if *format == "env" {
		var parsed map[string]interface{}
		if err := json.Unmarshal(body, &parsed); err == nil {
			if tmpl, ok := parsed["template"].(string); ok {
				fmt.Println(tmpl)
				return nil
			}
		}
	}
	cliutil.PrintJSON(body)
	return nil
}

func (c *Commands) secretsValidate(args []string) error {
	fs := flag.NewFlagSet("secrets validate", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	body, err := c.api.Request("POST", "/api/v1/profiles/"+id+"/secrets/validate", nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

// Helpers

func decodeProfile(data []byte) (Profile, error) {
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, err
	}
	normalizeProfile(&p)
	return p, nil
}

func normalizeProfile(p *Profile) {
	if p.Swaps == nil {
		p.Swaps = map[string]string{}
	}
	if p.Secrets == nil {
		p.Secrets = map[string]interface{}{}
	}
	if p.Settings == nil {
		p.Settings = map[string]interface{}{}
	}
}

func ensureNestedMap(obj map[string]interface{}, key string) map[string]interface{} {
	if val, ok := obj[key]; ok {
		if cast, ok := val.(map[string]interface{}); ok && cast != nil {
			return cast
		}
	}
	newMap := map[string]interface{}{}
	obj[key] = newMap
	return newMap
}

func fetchVersionHistory(api *cliutil.APIClient, profileID string) (*ProfileHistory, error) {
	body, err := api.Get("/api/v1/profiles/"+profileID+"/versions", nil)
	if err != nil {
		return nil, err
	}
	var history ProfileHistory
	if err := json.Unmarshal(body, &history); err != nil {
		return nil, fmt.Errorf("parse version history: %w", err)
	}
	for i := range history.Versions {
		normalizeProfile(&history.Versions[i])
	}
	return &history, nil
}

func extractUpdatableProfileFields(source Profile) map[string]interface{} {
	fields := map[string]interface{}{
		"tiers":    source.Tiers,
		"swaps":    source.Swaps,
		"secrets":  source.Secrets,
		"settings": source.Settings,
	}
	return fields
}

func versionNumber(v Profile) int {
	if v.Version != 0 {
		return v.Version
	}
	return 0
}

func findVersion(versions []Profile, version string) (Profile, bool) {
	for _, v := range versions {
		if fmt.Sprint(versionNumber(v)) == version {
			return v, true
		}
	}
	return Profile{}, false
}

func computeProfileDiff(previous, current Profile) map[string]map[string]interface{} {
	diff := map[string]map[string]interface{}{}

	if !intSlicesEqual(previous.Tiers, current.Tiers) {
		diff["tiers"] = map[string]interface{}{"from": previous.Tiers, "to": current.Tiers}
	}
	if !valuesEqual(previous.Swaps, current.Swaps) {
		diff["swaps"] = map[string]interface{}{"from": previous.Swaps, "to": current.Swaps}
	}
	if !valuesEqual(previous.Secrets, current.Secrets) {
		diff["secrets"] = map[string]interface{}{"from": previous.Secrets, "to": current.Secrets}
	}
	if !valuesEqual(previous.Settings, current.Settings) {
		diff["settings"] = map[string]interface{}{"from": previous.Settings, "to": current.Settings}
	}

	return diff
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func valuesEqual(a, b interface{}) bool {
	switch aVal := a.(type) {
	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for i := range aVal {
			if !valuesEqual(aVal[i], bVal[i]) {
				return false
			}
		}
		return true
	case []int:
		bVal, ok := b.([]int)
		if !ok {
			return false
		}
		return intSlicesEqual(aVal, bVal)
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for k, v := range aVal {
			if !valuesEqual(v, bVal[k]) {
				return false
			}
		}
		return true
	case map[string]string:
		bVal, ok := b.(map[string]string)
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for k, v := range aVal {
			if bVal[k] != v {
				return false
			}
		}
		return true
	default:
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
}

func intSliceToString(values []int) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}
