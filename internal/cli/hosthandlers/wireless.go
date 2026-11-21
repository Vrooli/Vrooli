package hosthandlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostwireless"
)

func runHostWireless(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	return runHostWirelessWithServices(runCtx, parent, defaultWirelessServices())
}

func runHostWirelessWithServices(runCtx cliapp.RunContext, parent *rootcli.CommandContext, services wirelessServices) error {
	action := runCtx.Positional("action")
	if err := validateWirelessActionFlags(runCtx, action); err != nil {
		return err
	}
	profile := strings.TrimSpace(runCtx.Flag("profile"))
	if profile == "" {
		return rootcli.UsageErrorf("host wireless", "--profile is required")
	}
	generation := strings.TrimSpace(runCtx.Flag("bundle"))
	recoveryRelease := strings.TrimSpace(runCtx.Flag("recovery-kernel"))
	signedDir := strings.TrimSpace(runCtx.Flag("signed-dir"))
	cert := strings.TrimSpace(runCtx.Flag("cert"))
	if isWirelessTransactionAction(action) || action == "sign" || action == "publish-signed" {
		return runWirelessOperationAction(runCtx, parent, action, services)
	}

	switch action {
	case "activation-preflight":
		if runCtx.BoolFlag("apply") || generation == "" || (signedDir != "" && cert == "") {
			return rootcli.UsageErrorf("host wireless activation-preflight", "use --bundle, optional --recovery-kernel and --cert; --signed-dir also requires --cert. Preflight is read-only")
		}
		preflight, err := hostwireless.PreflightActivation(parent.OperationContext(), profile, generation, recoveryRelease, signedDir, cert)
		if parent.Globals.JSON || runCtx.JSON() {
			return errors.Join(cliout.WriteJSONValue(runCtx.Stdout(), preflight), err)
		}
		rows := [][]string{{"Profile", preflight.Plan.Profile}, {"Bundle", preflight.Plan.Bundle}}
		for _, step := range preflight.Steps {
			rows = append(rows, []string{step.Name, step.State})
		}
		return errors.Join(cliout.WriteSection(runCtx.Stdout(), cliout.Section{Rows: rows}), err)
	case "qualify-firmware":
		if profile != hostwireless.NobleBluetoothProfile || generation != "" || recoveryRelease != "" {
			return rootcli.UsageErrorf("host wireless qualify-firmware", "use --profile noble-mt6639 and optional --apply for the local firmware artifact")
		}
		return runWirelessFirmwareQualification(runCtx, parent)
	case "verify-signed-copy":
		if runCtx.BoolFlag("apply") || recoveryRelease != "" || generation == "" || signedDir == "" || cert == "" {
			return rootcli.UsageErrorf("host wireless verify-signed-copy", "use --bundle, --signed-dir and --cert for read-only verification")
		}
		return runWirelessSignedCopy(runCtx, parent, profile, generation, signedDir, cert)
	case "activation-plan":
		if runCtx.BoolFlag("apply") {
			return rootcli.UsageErrorf("host wireless activation-plan", "activation-plan is read-only")
		}
		return runWirelessActivationPlan(runCtx, parent, profile, generation, recoveryRelease)
	case "prepare":
		if generation != "" || recoveryRelease != "" {
			return rootcli.UsageErrorf("host wireless prepare", "--bundle is for activation-plan or verify-signed-copy. --recovery-kernel is for activation-plan")
		}
	default:
		return rootcli.UsageErrorf("host wireless", "use prepare, activation-plan, activation-preflight, verify-signed-copy, qualify-firmware, sign, publish-signed, stage, activate, recover, rollback, status or record-recovery")
	}
	receipt, err := hostwireless.Prepare(parent.OperationContext(), profile, runCtx.BoolFlag("apply"))
	if parent.Globals.JSON || runCtx.JSON() {
		return errors.Join(cliout.WriteJSONValue(runCtx.Stdout(), receipt), err)
	}
	rows := [][]string{{"Profile", receipt.Profile}, {"Preparation", receipt.State}}
	if receipt.Reason != "" {
		rows = append(rows, []string{"Reason", receipt.Reason})
	}
	if receipt.ArtifactDirectory != "" {
		rows = append(rows, []string{"Artifacts", receipt.ArtifactDirectory})
	}
	rows = append(rows, []string{"Activation eligible", fmt.Sprint(receipt.ActivationEligible)})
	return errors.Join(cliout.WriteSection(runCtx.Stdout(), cliout.Section{Rows: rows}), err)
}

func runWirelessFirmwareQualification(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	receipt, err := hostwireless.QualifyBluetoothFirmware(parent.OperationContext(), runCtx.BoolFlag("apply"))
	if parent.Globals.JSON || runCtx.JSON() {
		return errors.Join(cliout.WriteJSONValue(runCtx.Stdout(), receipt), err)
	}
	rows := [][]string{
		{"Profile", receipt.Profile},
		{"Qualification", receipt.State},
		{"Firmware SHA-256", receipt.FirmwareSHA256},
		{"Activation eligible", fmt.Sprint(receipt.ActivationEligible)},
		{"Local-use basis", receipt.LocalUseBasis},
		{"Linux-use terms", receipt.LinuxUseTerms},
	}
	if receipt.Reason != "" {
		rows = append(rows, []string{"Reason", receipt.Reason})
	}
	if receipt.ArtifactDirectory != "" {
		rows = append(rows, []string{"Artifact", receipt.ArtifactDirectory})
	}
	return errors.Join(cliout.WriteSection(runCtx.Stdout(), cliout.Section{Rows: rows}), err)
}

func runWirelessSignedCopy(runCtx cliapp.RunContext, parent *rootcli.CommandContext, profile, generation, signedDir, cert string) error {
	evidence, err := hostwireless.VerifySignedCopy(parent.OperationContext(), profile, generation, signedDir, cert)
	if parent.Globals.JSON || runCtx.JSON() {
		return errors.Join(cliout.WriteJSONValue(runCtx.Stdout(), evidence), err)
	}
	rows := [][]string{
		{"Profile", evidence.Profile},
		{"Bundle", evidence.Bundle},
		{"Certificate SHA-256", evidence.CertificateSHA256},
		{"Signatures verified", fmt.Sprint(evidence.SignaturesVerified)},
		{"Kernel trust verified", fmt.Sprint(evidence.KernelTrustVerified)},
	}
	for _, module := range evidence.Modules {
		rows = append(rows, []string{module.Name, module.SignedSHA256})
	}
	return errors.Join(cliout.WriteSection(runCtx.Stdout(), cliout.Section{Rows: rows}), err)
}

func runWirelessActivationPlan(runCtx cliapp.RunContext, parent *rootcli.CommandContext, profile, generation, recoveryRelease string) error {
	plan, err := hostwireless.PreviewActivation(parent.OperationContext(), profile, generation, recoveryRelease)
	if parent.Globals.JSON || runCtx.JSON() {
		return errors.Join(cliout.WriteJSONValue(runCtx.Stdout(), plan), err)
	}
	rows := [][]string{
		{"Profile", plan.Profile},
		{"Preparation", plan.PreparationState},
		{"Activation eligible", fmt.Sprint(plan.ActivationEligible)},
	}
	for _, gate := range plan.Prerequisites {
		rows = append(rows, []string{"Prerequisite", gate})
	}
	return errors.Join(cliout.WriteSection(runCtx.Stdout(), cliout.Section{Rows: rows}), err)
}
