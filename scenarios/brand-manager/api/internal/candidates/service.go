package candidates

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"brand-manager/internal/assets"
	"brand-manager/internal/generation"
	"brand-manager/internal/imagetools"
	"brand-manager/internal/render"

	"github.com/google/uuid"
)

const (
	maxConcepts   = 8
	maxVariations = 4
	// exploreConcurrency bounds parallel generations in one explore round. Cloud
	// jobs run on image-tools' network lane, so a round costs about one job's
	// latency instead of the sum of every job.
	exploreConcurrency = 4
	// exploreBudget bounds one explore round once it is detached from the
	// caller's request.
	exploreBudget = 15 * time.Minute
	// conceptEdge is the square size concepts are rendered at.
	conceptEdge = 1024
	// referenceEdge is the square size a vector style reference is rendered at.
	referenceEdge = 1024
)

// OpenRouter roles for explore and refine. Concepts are rendered by an
// illustration model and traced to a vector at pick. The vector role draws flat
// geometric marks that look unfinished next to a rendered concept, so it is used
// only when a caller asks for a deliberately flat direction (PreferVector).
const (
	roleConcept        = "image.generate.default"
	roleConceptQuality = "image.generate.quality"
	roleReference      = "image.edit.default"
	roleReferenceAlt   = "image.edit.identity"
	roleVector         = "image.vector.default"
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
	Rasterize(ctx context.Context, svg []byte, width, height int, background string) ([]byte, error)
}

// BrandInfo is what candidates needs to know about a brand: its name and
// description for prompts, its picked mark for style references, and its
// container style for rendering a reference and choosing pick colours.
type BrandInfo struct {
	ID          string
	Name        string
	Description string
	MarkAssetID string
	Style       render.Style
}

// BrandContext resolves a brand by id or slug (wired to the brands and styles
// services).
type BrandContext interface {
	Resolve(ctx context.Context, ref string) (BrandInfo, error)
}

// Service is the candidates application layer.
type Service struct {
	store   Store
	assets  AssetStore
	backend Backend
	marks   MarkSetter
	brands  BrandContext
}

// NewService constructs the candidates service. brands may be nil: explore then
// writes a generic prompt and style references are unavailable.
func NewService(store Store, assetStore AssetStore, backend Backend, marks MarkSetter, brands BrandContext) *Service {
	return &Service{store: store, assets: assetStore, backend: backend, marks: marks, brands: brands}
}

// Import adopts an existing asset as a proposed candidate. It is idempotent per
// (brand, asset): importing the same image again returns the existing candidate
// instead of a second record for it.
func (s *Service) Import(ctx context.Context, brandID, assetID, concept, parentID, note string) (Candidate, error) {
	if strings.TrimSpace(brandID) == "" || strings.TrimSpace(assetID) == "" {
		return Candidate{}, errors.New("candidates: brand_id and asset_id are required")
	}
	content, err := s.assets.Download(ctx, assetID)
	if err != nil {
		return Candidate{}, err
	}
	existing, found, err := s.store.FindByAsset(ctx, brandID, assetID)
	if err != nil {
		return Candidate{}, err
	}
	if found {
		return existing, nil
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
		source, rerr := s.rasterIfVector(ctx, content, render.Style{})
		if rerr != nil {
			return Candidate{}, rerr
		}
		out, oerr := s.backend.Edit(ctx, generation.ImageEditRequest{
			Source:         source,
			Instruction:    action.Instruction,
			OpenRouterRole: roleReference,
			AllowBYOK:      generation.BrandImageAllowBYOK(""),
			QualityPolicy:  generation.BrandImageQualityPolicy(""),
			FallbackPolicy: generation.BrandImageFallbackPolicy(""),
		})
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
			KeepColors:                 s.markColors(ctx, cand.BrandID),
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

// Explore renders every concept × variation as a proposed candidate. Concepts
// render in parallel on a context detached from the caller, and each candidate is
// stored the moment its image lands, so a dropped client connection loses no
// generation already paid for. A style reference (another brand's approved mark,
// or an asset) makes every concept match that reference's rendering.
func (s *Service) Explore(ctx context.Context, in ExploreInput) ([]Candidate, []string, error) {
	if strings.TrimSpace(in.BrandID) == "" {
		return nil, nil, errors.New("candidates: brand_id is required")
	}
	concepts := in.Concepts
	if len(concepts) == 0 {
		concepts = []string{""}
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

	info := s.brandInfo(ctx, in.BrandID)
	reference, referenceNote, err := s.styleReference(ctx, in)
	if err != nil {
		return nil, nil, err
	}
	roles := exploreRoles(in, reference != nil)

	work, cancel := context.WithTimeout(context.WithoutCancel(ctx), exploreBudget)
	defer cancel()

	type slot struct {
		concept   string
		variation int
		result    exploreResult
	}
	var slots []*slot
	for _, concept := range concepts {
		for v := 1; v <= variations; v++ {
			slots = append(slots, &slot{concept: concept, variation: v})
		}
	}
	sem := make(chan struct{}, exploreConcurrency)
	var wg sync.WaitGroup
	for _, sl := range slots {
		wg.Add(1)
		go func(sl *slot) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			sl.result = s.exploreOne(work, in, info, reference, referenceNote, roles, sl.concept, sl.variation)
		}(sl)
	}
	wg.Wait()

	var out []Candidate
	var warnings []string
	var storeErr error
	for _, sl := range slots {
		warnings = append(warnings, sl.result.warnings...)
		switch {
		case sl.result.storeErr != nil:
			if storeErr == nil {
				storeErr = sl.result.storeErr
			}
		case sl.result.genErr != nil:
			warnings = append(warnings, fmt.Sprintf("concept %q variation %d: %v", sl.concept, sl.variation, sl.result.genErr))
		default:
			out = append(out, sl.result.cand)
		}
	}
	return out, dedupeStrings(warnings), storeErr
}

// exploreResult is one concept × variation outcome. A generation error becomes a
// warning; a storage error fails the round.
type exploreResult struct {
	cand     Candidate
	warnings []string
	genErr   error
	storeErr error
}

func (s *Service) exploreOne(ctx context.Context, in ExploreInput, info BrandInfo, reference []byte, referenceNote string, roles []string, concept string, variation int) exploreResult {
	prompt := conceptPrompt(info, in.Brief, concept, reference != nil)
	var (
		output   generation.ImageOutput
		roleUsed string
		genErr   error
	)
	for _, role := range roles {
		roleUsed = role
		if reference != nil {
			output, genErr = s.backend.Edit(ctx, generation.ImageEditRequest{
				Source:         reference,
				Instruction:    prompt,
				OpenRouterRole: role,
				AllowBYOK:      generation.BrandImageAllowBYOK(in.FallbackPolicy),
				QualityPolicy:  generation.BrandImageQualityPolicy(in.QualityPolicy),
				FallbackPolicy: generation.BrandImageFallbackPolicy(in.FallbackPolicy),
			})
		} else {
			output, genErr = s.backend.Generate(ctx, generation.ImageGenerateRequest{
				Prompt:         prompt,
				Width:          conceptEdge,
				Height:         conceptEdge,
				OpenRouterRole: role,
				AllowBYOK:      generation.BrandImageAllowBYOK(in.FallbackPolicy),
				QualityPolicy:  generation.BrandImageQualityPolicy(in.QualityPolicy),
				FallbackPolicy: generation.BrandImageFallbackPolicy(in.FallbackPolicy),
			})
		}
		if genErr == nil {
			break
		}
	}
	if genErr != nil {
		return exploreResult{genErr: genErr}
	}
	warnings := append([]string(nil), output.Warnings...)
	if strings.HasPrefix(output.Tier, "local") {
		warnings = append(warnings, fmt.Sprintf("concept %q variation %d rendered on the local model %s, which ignores the logo role; allow the cloud tier for concept-quality renders", concept, variation, output.ModelID))
	}
	asset, err := s.storeOutput(ctx, in.BrandID, output)
	if err != nil {
		return exploreResult{warnings: warnings, storeErr: err}
	}
	label := strings.TrimSpace(concept)
	if label == "" {
		label = "open concept"
	}
	cand, err := s.store.Create(ctx, Candidate{
		BrandID:   in.BrandID,
		AssetID:   asset.ID,
		MediaType: output.MimeType,
		Concept:   label,
		Prompt:    prompt,
		Role:      roleUsed,
		Model:     output.ModelID,
		Origin:    OriginGenerated,
		Status:    StatusProposed,
		Note:      referenceNote,
	})
	if err != nil {
		return exploreResult{warnings: warnings, storeErr: err}
	}
	return exploreResult{cand: cand, warnings: warnings}
}

// exploreRoles returns the roles to try in order: the caller's role; the edit
// roles when a style reference conditions the render; the vector role for a
// deliberately flat direction; otherwise the illustration roles.
func exploreRoles(in ExploreInput, hasReference bool) []string {
	if role := strings.TrimSpace(in.Role); role != "" {
		return []string{role}
	}
	if hasReference {
		return []string{roleReference, roleReferenceAlt}
	}
	if in.PreferVector {
		return []string{roleVector, roleConcept}
	}
	return []string{roleConcept, roleConceptQuality}
}

// conceptPrompt writes the render prompt for one concept. Without a reference it
// describes a finished icon in the brand's container palette. With a reference
// it asks for the reference's rendering with only the subject changed, which is
// how a product line stays one family.
func conceptPrompt(info BrandInfo, brief, concept string, withReference bool) string {
	name := strings.TrimSpace(info.Name)
	if name == "" {
		name = "the product"
	}
	about := strings.TrimSpace(brief)
	if about == "" {
		about = strings.TrimSpace(info.Description)
	}
	if about != "" {
		about = " Product: " + strings.TrimSuffix(about, ".") + "."
	}
	subject := strings.TrimSpace(concept)
	if subject == "" {
		subject = "a distinctive symbol for " + name
	}
	accent := firstNonEmpty(info.Style.AccentColor, "#22d3ee")
	if withReference {
		return fmt.Sprintf("Create a new app icon for %s that belongs to the same product family as the reference icon.%s "+
			"Keep everything about the reference's style: the same rounded-square tile and dark gradient, the same framing and margins, "+
			"the same luminous line weight and soft glow, the same glowing %s star points, the same level of polish and detail. "+
			"Change only the subject, which is: %s. The result must look like a sibling of the reference, not a copy of it. "+
			"No text, no letters, no watermark, no mouse pointer.", name, about, accent, subject)
	}
	top := firstNonEmpty(info.Style.BackgroundTop, "#15243c")
	bottom := firstNonEmpty(info.Style.BackgroundBottom, "#0b1728")
	return fmt.Sprintf("A finished, premium app icon for %s.%s Subject: %s. "+
		"Render it with depth and polish, like a flagship product icon: the subject centred on a rounded-square tile with a smooth dark gradient from %s to %s, "+
		"drawn in luminous white with %s highlights and a soft, refined glow, crisp and balanced with generous margins so it reads at small sizes. "+
		"Not flat clip art, not a sketch. No text, no letters, no watermark, no mouse pointer, nothing outside the tile.", name, about, subject, top, bottom, accent)
}

// brandInfo resolves a brand's prompt and style context, best effort: an
// unknown brand or a missing context yields just the id.
func (s *Service) brandInfo(ctx context.Context, brandID string) BrandInfo {
	if s.brands == nil {
		return BrandInfo{ID: brandID}
	}
	info, err := s.brands.Resolve(ctx, brandID)
	if err != nil {
		return BrandInfo{ID: brandID}
	}
	return info
}

// markColors are the colours a raster pick's vectorize keeps: white line art and
// the brand's container accent.
func (s *Service) markColors(ctx context.Context, brandID string) []string {
	info := s.brandInfo(ctx, brandID)
	return []string{"#ffffff", firstNonEmpty(info.Style.AccentColor, "#22d3ee")}
}

// styleReference returns the raster the concepts are conditioned on, with a note
// recorded on each candidate, or nil when explore has no reference.
func (s *Service) styleReference(ctx context.Context, in ExploreInput) ([]byte, string, error) {
	if ref := strings.TrimSpace(in.StyleReferenceBrand); ref != "" {
		if s.brands == nil {
			return nil, "", errors.New("candidates: style references need the brand context")
		}
		info, err := s.brands.Resolve(ctx, ref)
		if err != nil {
			return nil, "", fmt.Errorf("candidates: style reference brand %q: %w", ref, err)
		}
		png, err := s.brandReference(ctx, info)
		if err != nil {
			return nil, "", fmt.Errorf("candidates: style reference brand %q: %w", ref, err)
		}
		return png, "style reference: " + info.Name + "'s approved mark", nil
	}
	if id := strings.TrimSpace(in.StyleReferenceAssetID); id != "" {
		content, err := s.assets.Download(ctx, id)
		if err != nil {
			return nil, "", fmt.Errorf("candidates: style reference asset %q: %w", id, err)
		}
		png, err := s.rasterIfVector(ctx, content, render.Style{})
		if err != nil {
			return nil, "", fmt.Errorf("candidates: style reference asset %q: %w", id, err)
		}
		return png, "style reference: asset " + id, nil
	}
	return nil, "", nil
}

// brandReference returns the truest raster of a brand's approved look: the
// nearest raster ancestor of its picked candidate (the rendered concept the
// vector was traced from), else the picked mark composed in the brand's
// container style and rasterized.
func (s *Service) brandReference(ctx context.Context, info BrandInfo) ([]byte, error) {
	picked, err := s.store.List(ctx, info.ID, StatusPicked, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(picked) > 0 {
		cur := picked[0]
		for depth := 0; depth < 8; depth++ {
			if !isVector(cur.MediaType) {
				if content, derr := s.assets.Download(ctx, cur.AssetID); derr == nil {
					return content.Bytes, nil
				}
				break
			}
			if cur.ParentID == "" {
				break
			}
			parent, perr := s.store.Get(ctx, cur.ParentID)
			if perr != nil {
				break
			}
			cur = parent
		}
	}
	if strings.TrimSpace(info.MarkAssetID) == "" {
		return nil, errors.New("the brand has no picked mark")
	}
	mark, err := s.assets.Download(ctx, info.MarkAssetID)
	if err != nil {
		return nil, err
	}
	return s.rasterIfVector(ctx, mark, info.Style)
}

// rasterIfVector returns raster bytes unchanged and renders an SVG at
// referenceEdge, composed in style when a style is given.
func (s *Service) rasterIfVector(ctx context.Context, content assets.Content, style render.Style) ([]byte, error) {
	if !isVector(content.MimeType) {
		return content.Bytes, nil
	}
	svg := content.Bytes
	if style.BackgroundTop != "" {
		composed, err := render.Compose(svg, nil, style, render.Rounded, referenceEdge, false)
		if err != nil {
			return nil, err
		}
		svg = composed
	}
	return s.backend.Rasterize(ctx, svg, referenceEdge, referenceEdge, "")
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
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
