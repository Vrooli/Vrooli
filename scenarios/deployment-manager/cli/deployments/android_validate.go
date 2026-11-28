package deployments

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"deployment-manager/cli/cmdutil"
	"github.com/vrooli/cli-core/cliutil"
)

// AndroidValidate sends one immutable Android artifact through the
// scenario-to-android physical-device journey and waits through short status
// requests for the Deployment Manager evidence receipt.
func (c *Commands) AndroidValidate(args []string) error {
	fs := flag.NewFlagSet("android-validate", flag.ContinueOnError)
	profile := fs.String("profile", "", "deployment profile ID (required)")
	commit := fs.String("commit", "", "full source git commit hash (required)")
	artifactRef := fs.String("artifact-ref", "", "scenario-to-android immutable artifact ref (required)")
	targetID := fs.String("target", "", "exact physical Android target; omitted when exactly one is available")
	format := fs.String("format", "", "output format (json)")
	timeout := fs.Duration("timeout", 20*time.Minute, "maximum time to wait for device validation")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: deployment-manager android-validate --profile ID --commit SHA --artifact-ref REF [--target DEVICE_ID]\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*profile) == "" || strings.TrimSpace(*commit) == "" || strings.TrimSpace(*artifactRef) == "" {
		fs.Usage()
		return errors.New("--profile, --commit, and --artifact-ref are required")
	}
	if c.api == nil {
		return errors.New("deployment-manager API client is unavailable")
	}
	payload := map[string]string{"profile_id": strings.TrimSpace(*profile), "git_commit_hash": strings.TrimSpace(*commit), "artifact_ref": strings.TrimSpace(*artifactRef)}
	if strings.TrimSpace(*targetID) != "" {
		payload["target_id"] = strings.TrimSpace(*targetID)
	}
	body, err := c.api.Request("POST", "/api/v1/android/validate", url.Values{}, payload)
	if err != nil {
		return fmt.Errorf("Android deployment validation failed: %w", err)
	}
	var accepted struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(body, &accepted); err != nil || strings.TrimSpace(accepted.RunID) == "" {
		return fmt.Errorf("deployment-manager returned no Android validation run id")
	}
	deadline := time.Now().Add(*timeout)
	for {
		status, err := c.api.Request("GET", "/api/v1/android/validate/"+url.PathEscape(accepted.RunID), nil, nil)
		if err != nil {
			return fmt.Errorf("Android validation run %s did not produce a confirmed evidence receipt: %w", accepted.RunID, err)
		}
		var result struct {
			State         string         `json:"state"`
			GatePassed    bool           `json:"gate_passed"`
			ReleaseReport map[string]any `json:"release_report"`
		}
		if err := json.Unmarshal(status, &result); err != nil {
			return fmt.Errorf("decode Android validation status: %w", err)
		}
		receiptAt := strings.TrimSpace(fmt.Sprint(result.ReleaseReport["reported_at"]))
		if (result.State == "completed" || result.State == "failed" || result.State == "cancelled") && receiptAt != "" && receiptAt != "<nil>" {
			cmdutil.PrintByFormat(*format, status)
			if !result.GatePassed {
				return fmt.Errorf("Android validation run %s did not pass its release gate", accepted.RunID)
			}
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Android validation run %s is still running or waiting for its evidence receipt; inspect it with scenario-to-android android matrix-get %s", accepted.RunID, accepted.RunID)
		}
		time.Sleep(3 * time.Second)
	}
}
