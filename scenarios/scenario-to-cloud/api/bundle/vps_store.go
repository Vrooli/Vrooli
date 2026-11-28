package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// This file is the cloud side of the target release inventory. The target
// owner (`vrooli cloud-target release list|prune`) is the only authority
// over what lives beneath its release store; the cloud reads the typed
// listing, plans retention, and asks the owner to prune digests it names.
// No shell string and no path leaves the cloud.

// TargetRelease is one row of the owner's release listing.
type TargetRelease struct {
	Digest       string `json:"digest"`
	State        string `json:"state"`
	Role         string `json:"role"`
	Path         string `json:"path"`
	SizeBytes    int64  `json:"size_bytes"`
	ModTime      string `json:"mod_time"`
	BundleSHA256 string `json:"bundle_sha256"`
}

// TargetReleaseListing is the owner's durable release view.
type TargetReleaseListing struct {
	DeploymentID string `json:"deployment_id"`
	Active       *struct {
		ActiveRelease   string `json:"active_release"`
		PreviousRelease string `json:"previous_release"`
	} `json:"active,omitempty"`
	InterruptedActivation *struct {
		Candidate string `json:"candidate"`
		Previous  string `json:"previous"`
	} `json:"interrupted_activation,omitempty"`
	Releases []TargetRelease `json:"releases"`
}

// PruneReport is the owner's answer to a prune request.
type PruneReport struct {
	Deleted        []string          `json:"deleted"`
	Refused        map[string]string `json:"refused,omitempty"`
	ReclaimedBytes int64             `json:"reclaimed_bytes"`
}

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

const releaseVerbTimeout = 60 * time.Second

// ownerError is the typed refusal a cloud-target verb prints.
type ownerError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeOwnerReply(res reach.Result, err error, into any) error {
	if err != nil {
		return err
	}
	var reply struct {
		Error *ownerError `json:"error"`
	}
	trimmed := strings.TrimSpace(res.Stdout)
	if trimmed != "" {
		_ = json.Unmarshal([]byte(trimmed), &reply)
	}
	if reply.Error != nil {
		return fmt.Errorf("%s: %s", reply.Error.Code, reply.Error.Message)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("target owner exited %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	if into == nil {
		return nil
	}
	if err := json.Unmarshal([]byte(trimmed), into); err != nil {
		return fmt.Errorf("decode target owner reply: %w", err)
	}
	return nil
}

// ListTargetReleases reads the owner's release listing for one deployment.
func ListTargetReleases(ctx context.Context, r reach.Reach, target identity.TargetRef, deploymentID string) (TargetReleaseListing, error) {
	var listing TargetReleaseListing
	res, err := r.Exec(ctx, target, reach.Command{Verb: "cloud-target release list", Args: []string{"--deployment", deploymentID, "--json"}, RequiredScope: "vrooli:read", Timeout: releaseVerbTimeout})
	if err := decodeOwnerReply(res, err, &listing); err != nil {
		return TargetReleaseListing{}, err
	}
	return listing, nil
}

// PruneTargetReleases asks the owner to remove the named digests. The owner
// refuses active, previous and in-flight releases whatever is asked.
func PruneTargetReleases(ctx context.Context, r reach.Reach, target identity.TargetRef, deploymentID string, digests []string) (PruneReport, error) {
	args := []string{"--deployment", deploymentID}
	for _, d := range digests {
		if !digestPattern.MatchString(d) {
			return PruneReport{}, fmt.Errorf("refusing to prune malformed release digest %q", d)
		}
		args = append(args, "--release", d)
	}
	args = append(args, "--json")
	var reply struct {
		Report PruneReport `json:"report"`
	}
	res, err := r.Exec(ctx, target, reach.Command{Verb: "cloud-target release prune", Args: args, RequiredScope: "vrooli:write", Effectful: true, Timeout: releaseVerbTimeout})
	if err := decodeOwnerReply(res, err, &reply); err != nil {
		return PruneReport{}, err
	}
	return reply.Report, nil
}

// InventoryFromListing renders the owner's listing as the inventory rows the
// management API and CLI display. Filename carries the release digest (the
// owner's identity), Sha256 the bundle digest leases protect.
func InventoryFromListing(listing TargetReleaseListing, scenarioID string) ([]domain.VPSBundleInfo, int64) {
	var rows []domain.VPSBundleInfo
	var total int64
	for _, rel := range listing.Releases {
		rows = append(rows, domain.VPSBundleInfo{
			Filename:   rel.Digest,
			ScenarioID: scenarioID,
			Sha256:     rel.BundleSHA256,
			SizeBytes:  rel.SizeBytes,
			ModTime:    rel.ModTime,
			Role:       rel.Role,
			State:      rel.State,
		})
		total += rel.SizeBytes
	}
	return rows, total
}
