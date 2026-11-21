package composition

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type Provenance struct {
	ModelID           string `json:"model_id"`
	LicenseLane       string `json:"license_lane"`
	AppliedRung       string `json:"applied_rung"`
	Seed              int64  `json:"seed"`
	CaptionAsAuthored string `json:"caption_as_authored"`
	CaptionAsSent     string `json:"caption_as_sent"`
}

type Take struct {
	ID           string     `json:"id"`
	JobID        string     `json:"job_id"`
	StyleID      string     `json:"style_id"`
	PoolState    string     `json:"pool_state"`
	BlobRef      string     `json:"blob_ref"`
	ReservedBy   string     `json:"reserved_by,omitempty"`
	ReservedAt   *time.Time `json:"reserved_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	Provenance   Provenance `json:"provenance"`
	TimesOffered int        `json:"times_offered"`
}

type BatchRequest struct {
	JobID    string
	StyleID  string
	Caption  string
	Takes    int
	Duration int
	Seed     int64
}

var ErrInvalidBatch = errors.New("invalid composition batch")

func PlanBatch(req BatchRequest) ([]Take, error) {
	if req.JobID == "" || req.StyleID == "" || req.Caption == "" || req.Takes < 1 || req.Takes > 100 || req.Duration < 1 || req.Duration > 240 {
		return nil, ErrInvalidBatch
	}
	now := time.Now().UTC()
	rng := rand.New(rand.NewSource(req.Seed))
	out := make([]Take, req.Takes)
	for i := range out {
		// ACE's effective seed must survive its numeric boundary exactly.
		// The batch seed remains int64; each actual take receives a uint32.
		seed := int64(rng.Uint32())
		out[i] = Take{
			ID: fmt.Sprintf("%s-take-%02d", req.JobID, i+1), JobID: req.JobID, StyleID: req.StyleID, PoolState: "available", CreatedAt: now,
			Provenance: Provenance{ModelID: "", LicenseLane: "", AppliedRung: "", Seed: seed, CaptionAsAuthored: req.Caption, CaptionAsSent: req.Caption},
		}
	}
	return out, nil
}
