// Package designcapture owns durable composition capture intent and operation
// state. Browser execution and artifact bytes remain owned by BAS.
package designcapture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
)

type State string

const (
	Prepared        State = "prepared"
	Dispatching     State = "dispatching"
	DispatchUnknown State = "dispatch_unknown"
	Running         State = "running"
	CancelRequested State = "cancel_requested"
	Cancelled       State = "cancelled"
	Completed       State = "completed"
	Failed          State = "failed"
)

var ErrConflict = errors.New("capture operation conflict")
var hashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var segmentPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

type Target struct {
	Scenario     string `json:"scenario"`
	DesignID     string `json:"designId"`
	Revision     string `json:"revision"`
	RenderHash   string `json:"renderHash"`
	HTMLSHA256   string `json:"htmlSha256"`
	InputsSHA256 string `json:"inputsSha256"`
	Kind         string `json:"kind"`
	Kit          string `json:"kit"`
	Theme        string `json:"theme"`
	Direction    string `json:"direction"`
}
type Request struct {
	PreviousID string `json:"previousId,omitempty"`
	Target     Target `json:"target"`
	HTML       string `json:"html"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}
type RegionGeometry struct {
	Region string  `json:"region"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
type TargetEvidence struct {
	RenderHash string           `json:"renderHash"`
	Width      float64          `json:"width"`
	Height     float64          `json:"height"`
	Regions    []RegionGeometry `json:"regions"`
}
type Artifact struct {
	Evidence  *TargetEvidence `json:"evidence,omitempty"`
	Kind      string          `json:"kind"`
	Reference string          `json:"reference"`
}
type Operation struct {
	ID          string     `json:"id"`
	RequestHash string     `json:"requestHash"`
	Request     Request    `json:"request"`
	State       State      `json:"state"`
	ProducerID  string     `json:"producerId,omitempty"`
	Artifacts   []Artifact `json:"artifacts,omitempty"`
	Detail      string     `json:"detail,omitempty"`
	Version     int64      `json:"version"`
}
type Repository interface {
	Create(context.Context, string, Request) (Operation, error)
	Get(context.Context, string) (Operation, error)
	Transition(context.Context, string, int64, State, string, []Artifact, string) (Operation, error)
}

func validateRequest(r Request) error {
	if r.PreviousID != "" && (len(r.PreviousID) != 72 || !strings.HasPrefix(r.PreviousID, "capture_") || !hashPattern.MatchString(strings.TrimPrefix(r.PreviousID, "capture_"))) {
		return fmt.Errorf("invalid previous capture identity")
	}
	t := r.Target
	if !segmentPattern.MatchString(t.Scenario) || !segmentPattern.MatchString(t.DesignID) {
		return fmt.Errorf("invalid design identity")
	}
	for _, h := range []string{t.Revision, t.RenderHash, t.HTMLSHA256, t.InputsSHA256} {
		if !hashPattern.MatchString(h) {
			return fmt.Errorf("exact target hashes are required")
		}
	}
	if t.Kind != "preview" || t.Kit == "" || len(t.Kit) > 128 || (t.Theme != "light" && t.Theme != "dark") || (t.Direction != "ltr" && t.Direction != "rtl") {
		return fmt.Errorf("resolved preview appearance is required")
	}
	if r.Width < 100 || r.Width > 4000 || r.Height < 100 || r.Height > 4000 {
		return fmt.Errorf("capture dimensions must be between 100 and 4000 pixels")
	}
	if len(r.HTML) == 0 || len(r.HTML) > 4*1024*1024 {
		return fmt.Errorf("render HTML must be between 1 byte and 4 MiB")
	}
	digest := sha256.Sum256([]byte(r.HTML))
	if hex.EncodeToString(digest[:]) != t.HTMLSHA256 {
		return fmt.Errorf("render HTML differs from exact target")
	}
	return nil
}
func requestIdentity(r Request) (string, []byte, error) {
	if err := validateRequest(r); err != nil {
		return "", nil, err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), raw, nil
}
func allowed(from, to State) bool {
	switch from {
	case Prepared:
		return to == Dispatching || to == Cancelled
	case Dispatching:
		return to == Running || to == DispatchUnknown || to == Failed
	case DispatchUnknown:
		return to == Running || to == Failed
	case Running:
		return to == Completed || to == Failed || to == CancelRequested || to == Cancelled
	case CancelRequested:
		return to == Cancelled || to == Completed || to == Failed
	}
	return false
}

func validateOperation(op Operation) error {
	if op.Version < 1 {
		return fmt.Errorf("invalid capture operation version")
	}
	switch op.State {
	case Prepared, Dispatching, DispatchUnknown:
		if op.ProducerID != "" {
			return fmt.Errorf("producer identity precedes acknowledged dispatch")
		}
	case Running, CancelRequested, Completed:
		if op.ProducerID == "" {
			return fmt.Errorf("acknowledged capture lacks producer identity")
		}
	case Cancelled, Failed:
	default:
		return fmt.Errorf("invalid capture state")
	}
	if len(op.Detail) > 4000 || len(op.ProducerID) > 200 || len(op.Artifacts) > 32 {
		return fmt.Errorf("capture result exceeds metadata limits")
	}
	if len(op.Artifacts) > 0 && op.State != Completed {
		return fmt.Errorf("artifacts require terminal producer completion")
	}
	screenshot := false
	targetEvidence := false
	for _, a := range op.Artifacts {
		if a.Reference == "" || len(a.Reference) > 2048 || a.Kind == "" || len(a.Kind) > 80 {
			return fmt.Errorf("invalid artifact reference")
		}
		screenshot = screenshot || a.Kind == "screenshot"
		if a.Evidence != nil {
			e := a.Evidence
			if targetEvidence || a.Kind != "target" || e.RenderHash != op.Request.Target.RenderHash || e.Width != float64(op.Request.Width) || e.Height != float64(op.Request.Height) || len(e.Regions) > 512 {
				return fmt.Errorf("capture target evidence mismatch")
			}
			seen := map[string]bool{}
			for _, r := range e.Regions {
				if !segmentPattern.MatchString(r.Region) || seen[r.Region] || r.Width < 0 || r.Height < 0 {
					return fmt.Errorf("invalid captured region geometry")
				}
				seen[r.Region] = true
				for _, n := range []float64{r.X, r.Y, r.Width, r.Height} {
					if math.IsNaN(n) || math.IsInf(n, 0) {
						return fmt.Errorf("non-finite capture geometry")
					}
				}
			}
			targetEvidence = true
		}
	}
	if op.State == Completed && (!screenshot || !targetEvidence) {
		return fmt.Errorf("completed capture requires screenshot and exact target evidence")
	}
	return nil
}
