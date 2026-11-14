package capacitycli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/cliout"
)

type ResidencyArgs struct {
	Action, RequestFile, TransitionID string
}

func ParseResidencyRequest(args []string) (ResidencyArgs, error) {
	p, err := parse(CommandResidency, "vrooli capacity residency", args)
	if err != nil {
		return ResidencyArgs{}, err
	}
	if len(p.Positionals) != 1 {
		return ResidencyArgs{}, fmt.Errorf("residency requires one action: begin|complete|reconcile|recover|get|history|release")
	}
	out := ResidencyArgs{Action: p.Positionals[0], RequestFile: strings.TrimSpace(p.FlagValue("--request-file")), TransitionID: strings.TrimSpace(p.FlagValue("--transition-id"))}
	switch out.Action {
	case "begin", "complete", "reconcile", "recover", "release":
		if out.RequestFile == "" {
			return out, fmt.Errorf("residency %s requires --request-file", out.Action)
		}
		if out.TransitionID != "" {
			return out, fmt.Errorf("residency %s does not accept --transition-id", out.Action)
		}
	case "get", "history":
		if out.TransitionID == "" || out.RequestFile != "" {
			return out, fmt.Errorf("residency %s requires --transition-id only", out.Action)
		}
	default:
		return out, fmt.Errorf("unknown residency action %q", out.Action)
	}
	return out, nil
}

func ReadResidencyRequest(path string) (capacityapp.ResidencyRequest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return capacityapp.ResidencyRequest{}, fmt.Errorf("read residency request: %w", err)
	}
	var req capacityapp.ResidencyRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return capacityapp.ResidencyRequest{}, fmt.Errorf("decode residency request: %w", err)
	}
	return req, nil
}

func ReadResidencyReleaseRequest(path string) (capacityapp.ResidencyReleaseRequest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return capacityapp.ResidencyReleaseRequest{}, fmt.Errorf("read residency release request: %w", err)
	}
	var req capacityapp.ResidencyReleaseRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return req, fmt.Errorf("decode residency release request: %w", err)
	}
	if req.OwnerID == "" || req.ClaimID == "" || req.ClaimGeneration <= 0 || req.TransitionID == "" || req.Revision <= 0 || !req.Model.Complete() {
		return req, fmt.Errorf("residency release request requires exact owner, claim generation, transition revision, and model identity")
	}
	return req, nil
}

func RenderResidencyHistory(w io.Writer, format cliout.Format, events []engine.ResidencyTransitionEvent) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, events) }, func(w io.Writer) error {
		for _, event := range events {
			if _, err := fmt.Fprintf(w, "residency transition %s event=%s revision=%d recorded=%s known=%t backend=%s\n", event.Record.Transition.ID, event.EventKind, event.Record.Revision, event.RecordedAt.Format("2006-01-02T15:04:05Z07:00"), event.Record.Transition.Observed.Known, event.Record.Transition.Observed.Backend); err != nil {
				return err
			}
		}
		return nil
	})
}

func RenderResidencyRecord(w io.Writer, format cliout.Format, record engine.ResidencyTransitionRecord) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, record) }, func(w io.Writer) error {
		_, err := fmt.Fprintf(w, "residency transition %s claim=%s generation=%d revision=%d observed=%s known=%t released=%d\n", record.Transition.ID, record.ClaimID, record.ClaimGeneration, record.Revision, record.Transition.Observed.Backend, record.Transition.Observed.Known, record.Transition.VerifiedReleasedBytes())
		return err
	})
}

func RenderResidencyClaim(w io.Writer, format cliout.Format, claim engine.CapacityClaim) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, claim) }, func(w io.Writer) error {
		_, err := fmt.Fprintf(w, "released claim %s generation=%d status=%s\n", claim.ClaimID, claim.Generation, claim.Status)
		return err
	})
}
