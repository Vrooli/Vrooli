package twofa

import (
	"fmt"
	"os"
	"strings"

	"scenario-authenticator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `twofa` subcommand group covering setup/enable/disable/verify.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "twofa",
		Description: "Manage two-factor authentication (TOTP)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "setup", Description: "Begin 2FA setup for the authenticated user", Run: func(args []string) error { return runSetup(core, args) }},
			{Name: "enable", Description: "Verify a TOTP code and enable 2FA", Run: func(args []string) error { return runEnable(core, args) }},
			{Name: "disable", Description: "Disable 2FA (requires current TOTP code)", Run: func(args []string) error { return runDisable(core, args) }},
			{Name: "verify", Description: "Verify a TOTP code during login (requires --body-file)", Run: func(args []string) error { return runVerify(core, args) }},
		},
	}
}

func runSetup(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("twofa setup")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/auth/2fa/setup", nil, nil)
	if err != nil {
		return err
	}

	var resp support.TOTPConfig
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := []string{
		fmt.Sprintf("Secret: %s", resp.Secret),
		fmt.Sprintf("QR code URL: %s", resp.QRCodeURL),
	}
	if len(resp.BackupCodes) > 0 {
		result = append(result, fmt.Sprintf("Backup codes: %s", strings.Join(resp.BackupCodes, ", ")))
	}
	result = append(result, "Run `twofa enable --code <6-digit>` to finish activation.")

	report := cliapp.MutationReport{
		Result:      result,
		Changes:     []string{"2FA secret provisioned (pending activation)"},
		NextCommand: []string{fmt.Sprintf("%s twofa enable --code 123456", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runEnable(core *cliapp.ScenarioApp, args []string) error {
	return runCodeMutation(core, args, "twofa enable", "/auth/2fa/enable", "2FA enabled")
}

func runDisable(core *cliapp.ScenarioApp, args []string) error {
	return runCodeMutation(core, args, "twofa disable", "/auth/2fa/disable", "2FA disabled")
}

func runCodeMutation(core *cliapp.ScenarioApp, args []string, fsName, path, successMsg string) error {
	fs := support.NewFlagSet(fsName)
	code := fs.String("code", "", "6-digit TOTP code from the authenticator app (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*code) == "" {
		return fmt.Errorf("usage: %s --code <6-digit>", fsName)
	}

	body, err := core.Request("POST", path, nil, map[string]string{"code": *code})
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = successMsg
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{successMsg},
		NextCommand: []string{fmt.Sprintf("%s token validate $SCENARIO_AUTHENTICATOR_API_TOKEN", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runVerify(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("twofa verify")
	bodyFile := fs.String("body-file", "", "Path to JSON file with {email,code,token} payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/auth/2fa/verify", nil, raw)
	if err != nil {
		return err
	}

	var resp support.AuthResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	result := []string{"2FA verification complete"}
	if resp.Token != "" {
		result = append(result, fmt.Sprintf("Token: %s...", firstN(resp.Token, 20)))
	}
	if resp.RefreshToken != "" {
		result = append(result, fmt.Sprintf("Refresh token: %s...", firstN(resp.RefreshToken, 20)))
	}

	report := cliapp.MutationReport{
		Result:      result,
		Changes:     []string{"Login session established with 2FA"},
		NextCommand: []string{fmt.Sprintf("export SCENARIO_AUTHENTICATOR_API_TOKEN=%q", resp.Token)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
