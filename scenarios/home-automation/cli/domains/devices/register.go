package devices

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"home-automation/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "devices",
		Description: "List devices, inspect live state, and send control commands",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List devices [--filter text] [--json]", Run: func(args []string) error { return runList(core, args) }},
			{Name: "status", Description: "Show one device state", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "control", Description: "Control one device", Run: func(args []string) error { return runControl(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("devices list")
	filter := fs.String("filter", "", "Filter by id, name, or type")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/devices", nil)
	if err != nil {
		return err
	}

	var response support.DeviceListResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	filtered := make([]support.DeviceStatus, 0, len(response.Devices))
	query := strings.ToLower(strings.TrimSpace(*filter))
	for _, device := range response.Devices {
		if query == "" || strings.Contains(strings.ToLower(device.DeviceID), query) || strings.Contains(strings.ToLower(device.Name), query) || strings.Contains(strings.ToLower(device.Type), query) {
			filtered = append(filtered, device)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})

	results := make([]string, 0, len(filtered))
	for _, device := range filtered {
		state := support.FormatMapInline(device.State)
		results = append(results, fmt.Sprintf("%s | %s | type=%s | available=%t | state=%s", device.DeviceID, firstNonEmpty(device.Name, device.DeviceID), device.Type, device.Available, state))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Devices returned: %d", len(filtered)),
			fmt.Sprintf("Reported count: %d", response.Count),
			fmt.Sprintf("Data source: %s", firstNonEmpty(response.DataSource, "unknown")),
			support.BoolStatus(response.MockData, "Mock data is active", "Live or cached Home Assistant data is active"),
		},
		ResultsHeading: "Devices",
		Results:        results,
		RetrievalHints: []string{"home-automation devices status <device-id>", "home-automation devices control <device-id> turn_on --profile <profile-id>"},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("devices status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: home-automation devices status <device-id> [--json]")
	}

	deviceID := strings.TrimSpace(fs.Arg(0))
	body, err := core.Get("/devices/"+deviceID+"/status", nil)
	if err != nil {
		return err
	}

	var device support.DeviceStatus
	if err := support.Decode(body, &device); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Device: %s", firstNonEmpty(device.Name, device.DeviceID)),
			fmt.Sprintf("ID: %s", device.DeviceID),
			fmt.Sprintf("Type: %s", firstNonEmpty(device.Type, "unknown")),
			fmt.Sprintf("Available: %t", device.Available),
			fmt.Sprintf("Last updated: %s", support.HumanTime(device.LastUpdated)),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "State", Items: []string{support.FormatMapInline(device.State)}},
			{Heading: "Attributes", Items: []string{support.FormatMapInline(device.Attributes)}},
		},
		NextSteps: []string{
			fmt.Sprintf("home-automation devices control %s toggle --profile <profile-id>", device.DeviceID),
			"home-automation devices list --filter " + device.DeviceID,
		},
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runControl(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("devices control")
	profileID := fs.String("profile", "", "Profile ID used for authorization")
	userID := fs.String("user", "", "User ID override used for authorization")
	jsonOutput := cliutil.JSONFlag(fs)
	var params cliutil.StringList
	fs.Var(&params, "param", "Additional key=value parameter (repeatable)")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: home-automation devices control <device-id> <action> [--profile id] [--user id] [--param key=value] [--json]")
	}

	deviceID := strings.TrimSpace(fs.Arg(0))
	action := normalizeAction(fs.Arg(1))
	parameters, err := parseParams(params.Values())
	if err != nil {
		return err
	}
	if strings.TrimSpace(*profileID) == "" && strings.TrimSpace(*userID) == "" {
		*profileID = "mock-user-id"
	}

	body, err := core.Request("POST", "/devices/"+deviceID+"/control", nil, map[string]interface{}{
		"device_id":  deviceID,
		"action":     action,
		"profile_id": strings.TrimSpace(*profileID),
		"user_id":    strings.TrimSpace(*userID),
		"parameters": parameters,
	})
	if err != nil {
		return err
	}

	var response support.DeviceControlResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Control request success: %t", response.Success),
			fmt.Sprintf("Device: %s", response.DeviceID),
			fmt.Sprintf("Action: %s", response.Action),
			fmt.Sprintf("Message: %s", firstNonEmpty(response.Message, "No message returned")),
		},
		Changes: []string{
			"State: " + support.FormatMapInline(response.DeviceState),
			fmt.Sprintf("Execution time: %dms", response.ExecutionTimeMS),
			fmt.Sprintf("Request ID: %s", firstNonEmpty(response.RequestID, "unknown")),
		},
		NextCommand: []string{
			fmt.Sprintf("home-automation devices status %s", response.DeviceID),
			fmt.Sprintf("home-automation devices list --filter %s", response.DeviceID),
		},
	}
	if strings.TrimSpace(response.Error) != "" {
		report.Changes = append(report.Changes, "Error: "+response.Error)
	}
	if *jsonOutput {
		return support.PrintJSONReport(true, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func parseParams(values []string) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(values))
	for _, value := range values {
		parts := strings.SplitN(strings.TrimSpace(value), "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid --param value %q; expected key=value", value)
		}
		result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return result, nil
}

func normalizeAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on":
		return "turn_on"
	case "off":
		return "turn_off"
	default:
		return strings.TrimSpace(value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
