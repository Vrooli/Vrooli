// Package quality owns reproducible output-set scoring and regression gates.
// It reuses the image diff/analysis engines and delegates learned judgments to
// the local inference owner. A score is evidence, never a model promotion.
package quality

import qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"

const Version = "image-quality/1.0.0"
const Operation = "quality_evaluate"
const JudgeRole = "image.quality.judge"

// Direction is positive for metrics where larger is better and negative for
// errors. Unknown metrics are rejected instead of silently becoming a gate.
var Direction = map[string]float64{
	"reference.ssim": 1, "reference.psnr": 1, "reference.mae": -1,
	"reference.phash_similarity": 1, "dimensions_match": 1,
	"alpha.iou": 1, "alpha.mae": -1, "alpha.precision": 1, "alpha.recall": 1,
	"alpha.boundary_f1": 1, "text.exact": 1, "text.similarity": 1,
	"json.exact": 1, "json.field_accuracy": 1,
	"judge.fidelity": 1, "judge.instruction": 1, "judge.artifacts": 1,
}

var profiles = []*qualityv1.QualityProfile{
	{Id: "image", Description: "Reference-image fidelity for editing, generation, conversion, restoration and geometry previews", Metrics: []string{"reference.ssim", "reference.psnr", "reference.mae", "reference.phash_similarity", "dimensions_match"}, Rubric: "Compare the candidate with the reference and requested operation. Preserve intended content and geometry; judge clean edges, plausible details and visible artifacts. Do not reward sharpness that invents details. When the instruction requests a change, judge that change rather than penalizing every different pixel."},
	{Id: "matte", Description: "Transparency/matting quality from RGBA reference alpha", Metrics: []string{"alpha.iou", "alpha.mae", "alpha.precision", "alpha.recall", "alpha.boundary_f1", "dimensions_match"}, Rubric: "Judge foreground extraction against the reference. Inspect missing subject regions, remaining background, thin structures and edge halos. The contact sheet composites transparent pixels on a checkerboard so transparency is visible."},
	{Id: "mask", Description: "Segmentation quality from grayscale masks", Metrics: []string{"alpha.iou", "alpha.mae", "alpha.precision", "alpha.recall", "alpha.boundary_f1", "dimensions_match"}, Rubric: "Judge the candidate segmentation against the reference. Inspect coverage, excluded background, holes, thin features and boundary alignment. White means selected and black means unselected."},
	{Id: "text", Description: "OCR, captioning and other image-to-text outputs", Metrics: []string{"text.exact", "text.similarity"}, Rubric: "Judge whether the candidate text faithfully describes or transcribes the supplied source image and matches the reference facts. Penalize hallucinated objects, missed requested details and incorrect text. Accept semantically equivalent wording for descriptions; OCR must preserve exact characters."},
	{Id: "json", Description: "Structured analysis, metadata, coordinates, detections and numeric outputs", Metrics: []string{"json.exact", "json.field_accuracy"}, Rubric: "Judge the candidate structured output against the expected reference and requested operation. Check required facts, labels, counts, geometry and units. Do not invent missing evidence. Numeric fidelity and missing keys matter."},
}

func Profiles() []*qualityv1.QualityProfile {
	out := make([]*qualityv1.QualityProfile, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, &qualityv1.QualityProfile{Id: p.Id, Description: p.Description, Metrics: append([]string(nil), p.Metrics...), Rubric: p.Rubric})
	}
	return out
}

func profile(id string) *qualityv1.QualityProfile {
	for _, p := range profiles {
		if p.Id == id {
			return p
		}
	}
	return nil
}
