package sceneprep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
)

// AssessRepair inspects a retained mask before any model is loaded. It is a
// read-only operation and its risk signals cannot accept or reject an asset.
func (s *Service) AssessRepair(ctx context.Context, req *v1.AssessRepairRequest) (*v1.RepairAssessment, error) {
	if req == nil || !ownedRef(req.MaskRef) {
		return nil, fmt.Errorf("%w: an owned mask reference is required", ErrInvalid)
	}
	r, _, err := s.Blobs.Get(ctx, req.MaskRef)
	if err != nil {
		return nil, fmt.Errorf("%w: mask unavailable", ErrNotFound)
	}
	raw, err := io.ReadAll(io.LimitReader(r, (32<<20)+1))
	closeErr := r.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(raw) > 32<<20 {
		return nil, fmt.Errorf("%w: encoded mask exceeds 32 MiB", ErrInvalid)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 16_000_000 {
		return nil, fmt.Errorf("%w: mask dimensions are invalid or exceed 16 million pixels", ErrInvalid)
	}
	mask, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: mask decode failed", ErrInvalid)
	}
	p := req.GetInpaint()
	options := RepairAssessmentOptions{CropToMask: p.GetCropToMask(), ContextPixels: p.GetContextPixels(), MaxEdgePx: p.GetMaxEdgePx()}
	a, err := AssessRepairMask(mask, options)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return assessmentProto(a), nil
}

func assessmentProto(a RepairAssessment) *v1.RepairAssessment {
	return &v1.RepairAssessment{MaskAreaFraction: a.MaskAreaFraction, RiskLevel: a.RiskLevel, RiskSignals: a.RiskSignals, ContextPixels: a.ContextPixels, ContextClippedSides: a.ContextClippedSides, InferenceScale: a.InferenceScale}
}

// assessBundle measures actual retained removal-mask bytes. Crop options are
// descriptive caller-retained repair provenance, not a model attestation.
func assessBundle(bundle *Bundle) (*v1.RepairAssessment, error) {
	b := bundle.Manifest.Background
	if b.RemovalMask == nil {
		return nil, nil
	}
	r, err := bundle.OpenArtifact(b.RemovalMask.Path)
	if err != nil {
		return nil, err
	}
	mask, _, err := image.Decode(r)
	closeErr := r.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	options := RepairAssessmentOptions{}
	if b.RepairProvenance != nil {
		if b.RepairProvenance.SizeBytes > 64<<10 {
			return nil, fmt.Errorf("repair provenance exceeds 64 KiB")
		}
		r, err := bundle.OpenArtifact(b.RepairProvenance.Path)
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(io.LimitReader(r, (64<<10)+1))
		closeErr := r.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		var p struct {
			InpaintCrop struct {
				CropToMask    bool   `json:"crop_to_mask"`
				ContextPixels uint32 `json:"context_pixels"`
				MaxEdgePx     uint32 `json:"max_edge_px"`
			} `json:"inpaint_crop"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("repair provenance JSON: %w", err)
		}
		options = RepairAssessmentOptions{CropToMask: p.InpaintCrop.CropToMask, ContextPixels: p.InpaintCrop.ContextPixels, MaxEdgePx: p.InpaintCrop.MaxEdgePx}
	}
	a, err := AssessRepairMask(mask, options)
	if err != nil {
		return nil, err
	}
	return assessmentProto(a), nil
}
