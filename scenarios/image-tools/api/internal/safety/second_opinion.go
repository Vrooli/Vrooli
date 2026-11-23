package safety

import (
	"context"
	"fmt"
	"strings"
)

// AdultAgeFloor is the youngest apparent age at which a second opinion may clear an
// image whose face could not be read. It sits well above 18 on purpose: an estimate of
// "about 20" is not evidence of an adult.
const AdultAgeFloor = 25

// AgeJudgement is one face-independent age reading of an image.
type AgeJudgement struct {
	// ApparentAge is the estimate for the youngest person visible.
	ApparentAge int
	// CouldBeUnder18 is the judge's own flag that the person might be a minor.
	CouldBeUnder18 bool
	// Confidence is "high", "medium" or "low".
	Confidence string
	// Detail is a short note for the job record (never the image).
	Detail string
}

// ClearsAdult reports whether the reading is confident enough to call the subject an
// adult: not flagged as possibly under 18, estimated at AdultAgeFloor or older, and
// not low confidence. Anything else stays refused.
func (j AgeJudgement) ClearsAdult() bool {
	return !j.CouldBeUnder18 && j.ApparentAge >= AdultAgeFloor && !strings.EqualFold(j.Confidence, "low")
}

// AgeJudge reads an image's subject age without relying on a face (the whole body
// counts). An error means "could not judge" and never clears an image.
type AgeJudge interface {
	Judge(ctx context.Context, image []byte) (AgeJudgement, error)
}

// SecondOpinionReviewer wraps a reviewer and lets a second signal clear the one refusal
// the first cannot decide: a sexual image with no readable face. A face-based refusal (a
// minor-appearing face) is never overridden, and an image the judge cannot read confidently
// as an adult stays refused. The floor is still the primary reviewer's: this only
// replaces "could not establish the age" with a measured age where one exists.
type SecondOpinionReviewer struct {
	primary MinorReviewer
	judge   AgeJudge
}

// NewSecondOpinionReviewer builds the wrapper. A nil judge returns the primary unchanged.
func NewSecondOpinionReviewer(primary MinorReviewer, judge AgeJudge) MinorReviewer {
	if primary == nil || judge == nil {
		return primary
	}
	return &SecondOpinionReviewer{primary: primary, judge: judge}
}

func (r *SecondOpinionReviewer) Review(ctx context.Context, images [][]byte) (MinorVerdict, error) {
	verdict, err := r.primary.Review(ctx, images)
	if err != nil || !verdict.Minor {
		return verdict, err
	}
	// Clear a refusal only when the per-image findings explain every refused image
	// as "age not established"; any other refusal reason (or no findings) stands.
	if len(verdict.Images) != len(images) {
		return verdict, nil
	}
	cleared := make([]string, 0, len(images))
	for i, finding := range verdict.Images {
		if !finding.Refuse {
			continue
		}
		if !finding.Unassessable || finding.MinorFace {
			return verdict, nil
		}
		judgement, judgeErr := r.judge.Judge(ctx, images[i])
		if judgeErr != nil || !judgement.ClearsAdult() {
			note := "second opinion could not clear it"
			if judgeErr != nil {
				note += ": " + judgeErr.Error()
			} else {
				note += fmt.Sprintf(": apparent age %d, confidence %s", judgement.ApparentAge, judgement.Confidence)
			}
			verdict.Detail = strings.TrimSpace(verdict.Detail + "; image " + fmt.Sprint(i) + " " + note)
			return verdict, nil
		}
		cleared = append(cleared, fmt.Sprintf("image %d: no readable face, second opinion read an adult (apparent age %d, %s confidence)", i, judgement.ApparentAge, judgement.Confidence))
	}
	for i := range verdict.Images {
		if verdict.Images[i].Refuse {
			verdict.Images[i].Refuse = false
		}
	}
	verdict.Minor = false
	verdict.Detail = "passed after second opinion: " + strings.Join(cleared, "; ")
	return verdict, nil
}
