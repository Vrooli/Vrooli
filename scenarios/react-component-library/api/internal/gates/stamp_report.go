package gates

import (
	"encoding/json"
	"os"
	"path/filepath"

	corestorage "github.com/vrooli/api-core/storage"
)

// StampState is the build-time disposition of one catalog asset's DOM marker.
// It exists so a gate can say why an asset went unmeasured. Before this, an
// asset with no rendered evidence and an asset whose rendered evidence carries
// no identity both landed in one "unmeasured" bucket, which made a cheap
// config gap and an expensive capture gap indistinguishable.
type StampState string

const (
	StampStamped         StampState = "stamped"
	StampExemptPermanent StampState = "exempt-permanent"
	StampExemptBacklog   StampState = "exempt-backlog"
	StampUnbundled       StampState = "unbundled"
	StampUnknown         StampState = "unknown"
)

type stampReportDocument struct {
	GeneratedAt string         `json:"generatedAt"`
	Totals      map[string]int `json:"totals"`
	Assets      []struct {
		Asset     string `json:"asset"`
		LibraryID string `json:"libraryId"`
		Identity  string `json:"identity"`
		Version   string `json:"version"`
		State     string `json:"state"`
		Strategy  string `json:"strategy"`
		Reason    string `json:"reason"`
	} `json:"assets"`
}

// StampReport maps a catalog asset id to the state the last build recorded.
// A missing report is not an error: the gates must still run in a checkout
// where the UI has never been built, and every asset then reports unknown.
type StampReport struct {
	States    map[string]StampState
	Reasons   map[string]string
	Present   bool
	Generated string
}

func (r StampReport) State(assetID string) StampState {
	if !r.Present {
		return StampUnknown
	}
	if state, ok := r.States[assetID]; ok {
		return state
	}
	return StampUnknown
}

// Measurable reports whether an asset could ever produce rendered evidence.
// A permanently exempt asset renders no DOM root, so its absence from a
// capture is correct rather than a coverage gap.
func (r StampReport) Measurable(assetID string) bool {
	return r.State(assetID) != StampExemptPermanent
}

func LoadStampReport(root string) (StampReport, error) {
	report := StampReport{States: map[string]StampState{}, Reasons: map[string]string{}}
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		return report, err
	}
	path, err := resolver.ArtifactPath(corestorage.Options{ScenarioID: "react-component-library"}, corestorage.ArtifactRef{
		Owner: "react-component-library", Domain: "gates", Class: corestorage.ClassState, Segments: []string{"asset-stamp-report.json"},
	})
	if err != nil {
		return report, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// The UI build emits the report into its build output without
			// learning the state-class location. The owning API promotes that
			// payload into governed state on first use.
			buildReport := filepath.Join(root, "ui", "dist", "asset-stamp-report.json")
			raw, err = os.ReadFile(buildReport)
			if os.IsNotExist(err) {
				return report, nil
			}
			if err != nil {
				return report, err
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				return report, err
			}
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				return report, err
			}
		} else {
			return report, err
		}
	}
	var document stampReportDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return report, err
	}
	report.Present = true
	report.Generated = document.GeneratedAt
	for _, asset := range document.Assets {
		key := asset.Asset
		if key == "" {
			key = asset.Identity
		}
		if key == "" {
			continue
		}
		report.States[key] = StampState(asset.State)
		if asset.Reason != "" {
			report.Reasons[key] = asset.Reason
		}
	}
	return report, nil
}

// classifyUnmeasured splits a gate's unmeasured set by root cause. Callers
// keep the combined list for backward compatibility and gain the two halves
// that actually drive different work: unstamped assets need a build-config
// fix, uncaptured assets need capture coverage.
func classifyUnmeasured(report StampReport, unmeasured []string) (unstamped, uncaptured []string) {
	for _, assetID := range unmeasured {
		switch report.State(assetID) {
		case StampStamped:
			uncaptured = append(uncaptured, assetID)
		case StampExemptPermanent:
			// Renders no DOM root; not a coverage gap at all.
			continue
		default:
			unstamped = append(unstamped, assetID)
		}
	}
	return unstamped, uncaptured
}
