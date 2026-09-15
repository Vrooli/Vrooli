package candidates

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"brand-manager/internal/assets"
	"brand-manager/internal/generation"
	"brand-manager/internal/imagetools"

	"github.com/google/uuid"
)

const (
	maxConcepts   = 8
	maxVariations = 4
)

// AssetStore stores and reads brand assets (satisfied by assets.Service).
type AssetStore interface {
	Upload(ctx context.Context, in assets.UploadInput) (assets.Asset, error)
	Download(ctx context.Context, id string) (assets.Content, error)
}

// MarkSetter records a picked mark on the brand (wired to brands.Service).
type MarkSetter interface {
	SetMarkAsset(ctx context.Context, brandID, assetID string) error
}

// Backend is the image-tools surface candidates needs (satisfied by
// imagetools.Client).
type Backend interface {
	Generate(ctx context.Context, req generation.ImageGenerateRequest) (generation.ImageOutput, error)
	Edit(ctx context.Context, req generation.ImageEditRequest) (generation.ImageOutput, error)
	RemoveBackground(ctx context.Context, req generation.ImageRemoveBackgroundRequest) (generation.ImageOutput, error)
	Vectorize(ctx context.Context, src []byte, opts imagetools.VectorizeOptions) ([]byte, error)
	RemoveObject(ctx context.Context, src, mask []byte) (generation.ImageOutput, error)
}

// Service is the candidates application layer.
type Service struct {
	store   Store
	assets  AssetStore
	backend Backend
	marks   MarkSetter
}

// NewService constructs the candidates service.
func NewService(store Store, assetStore AssetStore, backend Backend, marks MarkSetter) *Service {
	return &Service{store: store, assets: assetStore, backend: backend, marks: marks}
}

// Import adopts an existing asset as a proposed candidate.
func (s *Service) Import(ctx context.Context, brandID, assetID, concept, parentID, note string) (Candidate, error) {
	if strings.TrimSpace(brandID) == "" || strings.TrimSpace(assetID) == "" {
		return Candidate{}, errors.New("candidates: brand_id and asset_id are required")
	}
	content, err := s.assets.Download(ctx, assetID)
	if err != nil {
		return Candidate{}, err
	}
	return s.store.Create(ctx, Candidate{
		BrandID:   brandID,
		AssetID:   assetID,
		MediaType: content.MimeType,
		Concept:   concept,
		Origin:    OriginImported,
		ParentID:  parentID,
		Status:    StatusProposed,
		Note:      note,
	})
}

// List returns the brand's candidates.
func (s *Service) List(ctx context.Context, brandID string, status Status, limit, offset int) ([]Candidate, error) {
	return s.store.List(ctx, brandID, status, limit, offset)
}

// Get returns one candidate.
func (s *Service) Get(ctx context.Context, id string) (Candidate, error) {
	return s.store.Get(ctx, id)
}

// Reject marks a candidate rejected.
func (s *Service) Reject(ctx context.Context, id, note string) (Candidate, error) {
	return s.store.UpdateStatus(ctx, id, StatusRejected, note)
}

// Restore returns a rejected/superseded candidate to proposed.
func (s *Service) Restore(ctx context.Context, id string) (Candidate, error) {
	return s.store.UpdateStatus(ctx, id, StatusProposed, "")
}

// Refine derives a new proposed candidate from a parent without mutating it.
func (s *Service) Refine(ctx context.Context, candidateID string, action RefineAction) (Candidate, error) {
	parent, err := s.store.Get(ctx, candidateID)
	if err != nil {
		return Candidate{}, err
	}
	content, err := s.assets.Download(ctx, parent.AssetID)
	if err != nil {
		return Candidate{}, err
	}

	switch {
	case action.Vectorize != nil:
		svg, verr := s.backend.Vectorize(ctx, content.Bytes, toImageToolsOptions(*action.Vectorize))
		if verr != nil {
			return Candidate{}, verr
		}
		asset, aerr := s.storeAsset(ctx, parent.BrandID, svg, "image/svg+xml", "svg")
		if aerr != nil {
			return Candidate{}, aerr
		}
		return s.store.Create(ctx, childCandidate(parent, asset, OriginVectorized, ""))
	case action.MaskAssetID != "":
		mask, merr := s.assets.Download(ctx, action.MaskAssetID)
		if merr != nil {
			return Candidate{}, merr
		}
		out, oerr := s.backend.RemoveObject(ctx, content.Bytes, mask.Bytes)
		if oerr != nil {
			return Candidate{}, oerr
		}
		asset, aerr := s.storeOutput(ctx, parent.BrandID, out)
		if aerr != nil {
			return Candidate{}, aerr
		}
		return s.store.Create(ctx, childCandidate(parent, asset, OriginObjectRemoved, ""))
	case action.RemoveBackground:
		out, oerr := s.backend.RemoveBackground(ctx, generation.ImageRemoveBackgroundRequest{Source: content.Bytes})
		if oerr != nil {
			return Candidate{}, oerr
		}
		asset, aerr := s.storeOutput(ctx, parent.BrandID, out)
		if aerr != nil {
			return Candidate{}, aerr
		}
		return s.store.Create(ctx, childCandidate(parent, asset, OriginBackgroundRemoved, ""))
	case action.Instruction != "":
		out, oerr := s.backend.Edit(ctx, generation.ImageEditRequest{Source: content.Bytes, Instruction: action.Instruction})
		if oerr != nil {
			return Candidate{}, oerr
		}
		asset, aerr := s.storeOutput(ctx, parent.BrandID, out)
		if aerr != nil {
			return Candidate{}, aerr
		}
		return s.store.Create(ctx, childCandidate(parent, asset, OriginEdited, action.Instruction))
	default:
		return Candidate{}, errors.New("candidates: refine requires an instruction, mask, background removal or vectorize")
	}
}

// Pick promotes a candidate to the brand's canonical mark. A raster pick is
// vectorized first and the VECTORIZED child is the one actually picked.
func (s *Service) Pick(ctx context.Context, candidateID string) (Candidate, string, bool, error) {
	cand, err := s.store.Get(ctx, candidateID)
	if err != nil {
		return Candidate{}, "", false, err
	}
	vectorized := false
	if !isVector(cand.MediaType) {
		content, derr := s.assets.Download(ctx, cand.AssetID)
		if derr != nil {
			return Candidate{}, "", false, derr
		}
		svg, verr := s.backend.Vectorize(ctx, content.Bytes, imagetools.VectorizeOptions{
			KeepColors:                 []string{"#ffffff", "#22d3ee"},
			ClipToLargestRoundedRegion: true,
			TolerancePx:                0.8,
		})
		if verr != nil {
			return Candidate{}, "", false, verr
		}
		asset, aerr := s.storeAsset(ctx, cand.BrandID, svg, "image/svg+xml", "svg")
		if aerr != nil {
			return Candidate{}, "", false, aerr
		}
		child, cerr := s.store.Create(ctx, childCandidate(cand, asset, OriginVectorized, "auto-vectorized on pick"))
		if cerr != nil {
			return Candidate{}, "", false, cerr
		}
		cand = child
		vectorized = true
	}

	// Supersede the current pick first so the unique partial index never sees two.
	if err := s.store.SupersedePicked(ctx, cand.BrandID, cand.ID); err != nil {
		return Candidate{}, "", false, err
	}
	picked, err := s.store.UpdateStatus(ctx, cand.ID, StatusPicked, "")
	if err != nil {
		return Candidate{}, "", false, err
	}
	if err := s.marks.SetMarkAsset(ctx, cand.BrandID, cand.AssetID); err != nil {
		return Candidate{}, "", false, err
	}
	return picked, cand.AssetID, vectorized, nil
}

// Explore fans out one generation per concept and variation.
func (s *Service) Explore(ctx context.Context, in ExploreInput) ([]Candidate, []string, error) {
	if strings.TrimSpace(in.BrandID) == "" {
		return nil, nil, errors.New("candidates: brand_id is required")
	}
	concepts := in.Concepts
	if len(concepts) == 0 {
		concepts = []string{"logo mark"}
	}
	if len(concepts) > maxConcepts {
		concepts = concepts[:maxConcepts]
	}
	variations := in.Variations
	if variations <= 0 {
		variations = 1
	}
	if variations > maxVariations {
		variations = maxVariations
	}

	roles := []string{"image.generate.logo"}
	if in.PreferVector {
		roles = []string{"image.vector.default", "image.generate.logo"}
	}

	var out []Candidate
	var warnings []string
	for _, concept := range concepts {
		for v := 0; v < variations; v++ {
			prompt := markPrompt(in.Brief, concept)
			var output generation.ImageOutput
			var roleUsed string
			var genErr error
			for _, role := range roles {
				output, genErr = s.backend.Generate(ctx, generation.ImageGenerateRequest{
					Prompt:         prompt,
					Width:          512,
					Height:         512,
					OpenRouterRole: role,
					AllowBYOK:      in.AllowBYOK,
					QualityPolicy:  in.QualityPolicy,
					FallbackPolicy: in.FallbackPolicy,
				})
				roleUsed = role
				if genErr == nil {
					break
				}
			}
			if genErr != nil {
				warnings = append(warnings, fmt.Sprintf("concept %q variation %d: %v", concept, v+1, genErr))
				continue
			}
			asset, aerr := s.storeOutput(ctx, in.BrandID, output)
			if aerr != nil {
				return out, warnings, aerr
			}
			cand, cerr := s.store.Create(ctx, Candidate{
				BrandID:   in.BrandID,
				AssetID:   asset.ID,
				MediaType: output.MimeType,
				Concept:   concept,
				Prompt:    prompt,
				Role:      roleUsed,
				Model:     output.ModelID,
				Origin:    OriginGenerated,
				Status:    StatusProposed,
			})
			if cerr != nil {
				return out, warnings, cerr
			}
			warnings = append(warnings, output.Warnings...)
			out = append(out, cand)
		}
	}
	return out, warnings, nil
}

func (s *Service) storeOutput(ctx context.Context, brandID string, out generation.ImageOutput) (assets.Asset, error) {
	ext := extForMime(out.MimeType)
	filename := "candidate-" + uuid.NewString()[:8] + "." + ext
	return s.assets.Upload(ctx, assets.UploadInput{BrandID: brandID, Filename: filename, MimeType: out.MimeType, Content: out.Data})
}

func (s *Service) storeAsset(ctx context.Context, brandID string, data []byte, mime, ext string) (assets.Asset, error) {
	filename := "candidate-" + uuid.NewString()[:8] + "." + ext
	return s.assets.Upload(ctx, assets.UploadInput{BrandID: brandID, Filename: filename, MimeType: mime, Content: data})
}

func childCandidate(parent Candidate, asset assets.Asset, origin Origin, note string) Candidate {
	return Candidate{
		BrandID:   parent.BrandID,
		AssetID:   asset.ID,
		MediaType: asset.MimeType,
		Concept:   parent.Concept,
		Prompt:    parent.Prompt,
		Role:      parent.Role,
		Model:     parent.Model,
		Origin:    origin,
		ParentID:  parent.ID,
		Status:    StatusProposed,
		Note:      note,
	}
}

func markPrompt(brief, concept string) string {
	brief = strings.TrimSpace(brief)
	concept = strings.TrimSpace(concept)
	if brief == "" {
		brief = "a product brand"
	}
	return fmt.Sprintf("%s mark for %s, vector line art, flat colours, centred, on a transparent background. Draw only the mark — no tile, no frame, no drop shadow, no text.", concept, brief)
}

func isVector(mediaType string) bool {
	m := strings.ToLower(mediaType)
	return strings.Contains(m, "svg")
}

func extForMime(mime string) string {
	switch strings.ToLower(mime) {
	case "image/svg+xml":
		return "svg"
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	default:
		return "png"
	}
}

func toImageToolsOptions(o VectorizeOptions) imagetools.VectorizeOptions {
	return imagetools.VectorizeOptions{
		Colors:                     o.Colors,
		KeepColors:                 o.KeepColors,
		DropBackgroundLayers:       o.DropBackgroundLayers,
		ClipToLargestRoundedRegion: o.ClipToLargestRoundedRegion,
		InsetPx:                    o.InsetPx,
		TolerancePx:                o.TolerancePx,
		Smoothing:                  o.Smoothing,
		MinAreaPx:                  o.MinAreaPx,
	}
}
