package hosthandlers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostwireless"
)

type wirelessServices struct {
	operation         func(context.Context, string, hostwireless.WirelessOperationRequest, bool) (hostwireless.WirelessOperationResult, error)
	sign              func(context.Context, string, string, string, string) (hostwireless.SignedBundleReceipt, error)
	publish           func(context.Context, string, string, string, string) (hostwireless.SignedBundleReceipt, error)
	verify            func(context.Context, string, string, string, string) (hostwireless.SignedCopyEvidence, error)
	plan              func(context.Context, string, string, string) (hostwireless.ActivationPlan, error)
	trust             func(context.Context, string) (hostwireless.SigningTrust, error)
	elevationRequired func() bool
}

func defaultWirelessServices() wirelessServices {
	return wirelessServices{
		operation: hostwireless.RunWirelessOperation, sign: hostwireless.SignPreparedBundle, publish: hostwireless.PublishSignedBundle,
		verify: hostwireless.VerifySignedCopy, plan: hostwireless.PreviewActivation, trust: hostwireless.InspectSigningTrust, elevationRequired: func() bool { return runtime.GOOS == "linux" && os.Geteuid() != 0 },
	}
}

func isWirelessTransactionAction(action string) bool {
	return slices.Contains([]string{"stage", "activate", "recover", "rollback", "status", "record-recovery"}, action)
}

// Reject flags that would otherwise look effective while being silently ignored.
func validateWirelessActionFlags(run cliapp.RunContext, action string) error {
	allowed := map[string][]string{
		"prepare": {"profile", "apply"}, "qualify-firmware": {"profile", "apply"},
		"activation-plan":      {"profile", "bundle", "recovery-kernel"},
		"activation-preflight": {"profile", "bundle", "recovery-kernel", "signed-dir", "cert"},
		"verify-signed-copy":   {"profile", "bundle", "signed-dir", "cert"},
		"publish-signed":       {"profile", "bundle", "signed-dir", "cert", "apply"},
		"sign":                 {"profile", "bundle", "cert", "receipt-sha256", "apply"},
		"stage":                {"profile", "bundle", "signed-bundle", "receipt-sha256", "recovery-kernel", "recovery-entry", "maintenance-window", "apply"},
		"activate":             {"profile", "transaction", "maintenance-window", "apply"},
		"recover":              {"profile", "transaction", "maintenance-window", "apply"},
		"rollback":             {"profile", "transaction", "maintenance-window", "apply"},
		"status":               {"profile", "transaction"},
		"record-recovery":      {"profile", "recovery-kernel", "recovery-entry", "apply"},
	}
	flags, known := allowed[action]
	if !known {
		return nil
	}
	for _, flag := range []string{"profile", "bundle", "signed-dir", "cert", "receipt-sha256", "signed-bundle", "recovery-kernel", "recovery-entry", "transaction", "apply", "maintenance-window"} {
		if !run.FlagDeclared(flag) {
			continue
		}
		provided := run.FlagProvided(flag)
		if provided && !slices.Contains(flags, flag) {
			return rootcli.UsageErrorf("host wireless "+action, "--%s is not supported for this action", flag)
		}
	}
	return nil
}

func runWirelessOperationAction(run cliapp.RunContext, parent *rootcli.CommandContext, action string, s wirelessServices) error {
	get := func(flag string) string { return strings.TrimSpace(run.Flag(flag)) }
	profile, bundle, cert := get("profile"), get("bundle"), get("cert")
	apply := run.BoolFlag("apply")
	ctx := parent.OperationContext()
	if action == "publish-signed" {
		if bundle == "" || get("signed-dir") == "" || cert == "" {
			return rootcli.UsageErrorf("host wireless publish-signed", "--bundle, --signed-dir and --cert are required; add --apply to retain authenticated public artifacts")
		}
		if !apply {
			result, err := s.verify(ctx, profile, bundle, get("signed-dir"), cert)
			return writeWirelessResult(run, parent, result, [][]string{{"Profile", result.Profile}, {"Bundle", result.Bundle}, {"Signatures verified", fmt.Sprint(result.SignaturesVerified)}, {"Publication", "preview; add --apply to publish"}}, err)
		}
		result, err := s.publish(ctx, profile, bundle, get("signed-dir"), cert)
		return writeWirelessSignedBundle(run, parent, result, err)
	}
	if action == "sign" {
		pin := get("receipt-sha256")
		if bundle == "" || cert == "" || (apply && pin == "") {
			return rootcli.UsageErrorf("host wireless sign", "--bundle and --cert are required; --apply also requires the independently reviewed --receipt-sha256")
		}
		if !apply {
			plan, err := s.plan(ctx, profile, bundle, "")
			if err == nil && pin != "" && pin != plan.BundleReceiptSHA256 {
				err = errors.New("reviewed receipt SHA-256 differs from the canonical preparation")
			}
			trust, certErr := s.trust(ctx, cert)
			preview := struct {
				Operation   string                      `json:"operation"`
				State       string                      `json:"state"`
				Plan        hostwireless.ActivationPlan `json:"plan"`
				Certificate hostwireless.SigningTrust   `json:"certificate"`
			}{"sign", "preview", plan, trust}
			return writeWirelessResult(run, parent, preview, [][]string{{"Operation", "sign"}, {"State", "preview"}, {"Profile", profile}, {"Bundle", bundle}, {"Preparation receipt SHA-256", plan.BundleReceiptSHA256}, {"Certificate SHA-256", trust.CertificateSHA256}, {"Signing", "requires explicit --apply, reviewed receipt pin and an elevated caller"}}, errors.Join(err, certErr))
		}
		if !validWirelessReceiptPin(pin) {
			return rootcli.UsageErrorf("host wireless sign", "--receipt-sha256 must be the independently reviewed lowercase SHA-256 digest")
		}
		if s.elevationRequired() {
			return fmt.Errorf("%w: signing requires an elevated caller; rerun this exact host wireless sign command with sudo", hostreqkit.ErrElevationRequired)
		}
		result, err := s.sign(ctx, profile, bundle, cert, pin)
		return writeWirelessSignedBundle(run, parent, result, err)
	}
	req := hostwireless.WirelessOperationRequest{
		Profile: profile, Bundle: bundle, SignedBundle: get("signed-bundle"), ReceiptSHA256: get("receipt-sha256"),
		RecoveryKernel: get("recovery-kernel"), RecoveryEntry: get("recovery-entry"), Transaction: get("transaction"), MaintenanceWindow: run.BoolFlag("maintenance-window"),
	}
	if action == "stage" && (bundle == "" || req.RecoveryKernel == "" || req.RecoveryEntry == "" || (apply && (req.SignedBundle == "" || req.ReceiptSHA256 == ""))) {
		return rootcli.UsageErrorf("host wireless stage", "--bundle, --recovery-kernel and --recovery-entry are required; --apply also requires --signed-bundle and the independently reviewed --receipt-sha256")
	}
	if slices.Contains([]string{"activate", "recover", "rollback", "status"}, action) && req.Transaction == "" {
		return rootcli.UsageErrorf("host wireless "+action, "--transaction is required")
	}
	if action == "record-recovery" && (req.RecoveryKernel == "" || req.RecoveryEntry == "") {
		return rootcli.UsageErrorf("host wireless record-recovery", "--recovery-kernel and --recovery-entry are required")
	}
	result, err := s.operation(ctx, action, req, apply)
	rows := [][]string{{"Operation", result.Operation}, {"Profile", result.Profile}, {"State", result.State}, {"Ready", fmt.Sprint(result.Ready)}}
	if result.Transaction != "" {
		rows = append(rows, []string{"Transaction", result.Transaction})
	}
	for _, check := range result.Checks {
		rows = append(rows, []string{check.Name, fmt.Sprintf("%t: %s", check.Satisfied, check.Detail)})
	}
	if errors.Is(err, hostreqkit.ErrElevationRequired) {
		err = fmt.Errorf("%w; rerun this exact host wireless %s command with sudo", err, action)
	}
	return writeWirelessResult(run, parent, result, rows, err)
}

func writeWirelessSignedBundle(run cliapp.RunContext, parent *rootcli.CommandContext, r hostwireless.SignedBundleReceipt, err error) error {
	return writeWirelessResult(run, parent, r, [][]string{{"Profile", r.Profile}, {"Bundle", r.Bundle}, {"Signed bundle", r.ReceiptSHA256}, {"Directory", r.Directory}, {"Modules", r.ModulesDirectory}, {"Public certificate", r.PublicCertificate}, {"Signatures verified", fmt.Sprint(r.SignaturesVerified)}, {"Kernel trust verified", fmt.Sprint(r.KernelTrustVerified)}}, err)
}

func writeWirelessResult(run cliapp.RunContext, parent *rootcli.CommandContext, result any, rows [][]string, err error) error {
	if parent.Globals.JSON || run.JSON() {
		return errors.Join(cliout.WriteJSONValue(run.Stdout(), result), err)
	}
	return errors.Join(cliout.WriteSection(run.Stdout(), cliout.Section{Rows: rows}), err)
}

func validWirelessReceiptPin(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, ch := range value {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f') {
			return false
		}
	}
	return true
}
