package devices

import (
	"encoding/json"
	"fmt"
	"os"

	"device-sync-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `device` subcommand group covering list/get/register/update/delete
// against /api/v1/devices. The API owns all business logic; this package is a
// thin wrapper around the HTTP surface.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "device",
		Description: "Manage registered devices",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List registered devices", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one device", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "register", Description: "Register a new device (body via --body-file)", Run: func(args []string) error { return runRegister(core, args) }},
			{Name: "update", Description: "Update a device (body via --body-file)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a device", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("device list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/devices", nil)
	if err != nil {
		return err
	}
	var devices []support.Device
	if err := support.Decode(body, &devices); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Registered devices: %d", len(devices))},
		ResultsHeading: "Devices",
		Results:        deviceRows(devices),
		RetrievalHints: []string{
			fmt.Sprintf("%s device get <device-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("device get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: device get <device-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/devices/"+id, nil)
	if err != nil {
		return err
	}
	var device support.Device
	if err := support.Decode(body, &device); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", device.ID),
		fmt.Sprintf("Name: %s", device.Name),
		fmt.Sprintf("Type: %s", device.Type),
		fmt.Sprintf("Platform: %s", device.Platform),
		fmt.Sprintf("Online: %t", device.IsOnline),
		fmt.Sprintf("Last seen: %s", support.FormatTimeValue(device.LastSeen)),
	}
	if len(device.Capabilities) > 0 {
		results = append(results, fmt.Sprintf("Capabilities: %v", device.Capabilities))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Device: %s", device.Name)},
		ResultsHeading: "Details",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRegister(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("device register")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the device payload (name, type, platform, capabilities)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/devices", nil, payload)
	if err != nil {
		return err
	}
	var device support.Device
	if err := support.Decode(body, &device); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Device registered: %s", device.ID)},
		Changes: []string{fmt.Sprintf("Name: %s (%s/%s)", device.Name, device.Type, device.Platform)},
		NextCommand: []string{
			fmt.Sprintf("%s device get %s", support.CLIName, device.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("device update")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with update fields (e.g., {\"name\": \"new\"})")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: device update <device-id> --body-file PATH")
	}
	id := fs.Arg(0)

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	// PUT returns 204 No Content (empty body) on success.
	if _, err := core.Request("PUT", "/devices/"+id, nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Device %s updated", id)},
		Changes: []string{"Updated fields from payload"},
		NextCommand: []string{
			fmt.Sprintf("%s device get %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("device delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: device delete <device-id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/devices/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Device %s deleted", id)},
		Changes:     []string{"Removed device and terminated its WebSocket sessions"},
		NextCommand: []string{fmt.Sprintf("%s device list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func deviceRows(devices []support.Device) []string {
	if len(devices) == 0 {
		return []string{"No devices registered"}
	}
	rows := make([]string, 0, len(devices))
	for _, d := range devices {
		online := "offline"
		if d.IsOnline {
			online = "online"
		}
		capsJSON, _ := json.Marshal(d.Capabilities)
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | %s/%s | caps=%s | last_seen=%s",
			d.Name, support.ShortID(d.ID), online, d.Type, d.Platform, string(capsJSON), support.FormatTimeValue(d.LastSeen)))
	}
	return rows
}
