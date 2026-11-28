package safety

import (
	"context"
	"fmt"
	"strings"
)

// AdultAgeFloor is the default youngest apparent age at which a second opinion
// may clear an image whose face could not be read. It sits well above 18 on
// purpose: an estimate of "about 20" is not evidence of an adult. A profile may
// raise it (the shipped profiles use 25).
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
// adult under the default profile: not flagged as possibly under 18, estimated at
// AdultAgeFloor or older, and not low confidence.
func (j AgeJudgement) ClearsAdult() bool {
	return j.ClearsAdultWith(AdultAgeFloor, "medium")
}

// ClearsAdultWith applies a profile's floor and minimum confidence. Anything
// below the floor, flagged under 18, or less confident than the profile allows
// stays refused. An empty confidence is treated as low.
func (j AgeJudgement) ClearsAdultWith(minAge int, minConfidence string) bool {
	if minAge <= 0 {
		minAge = AdultAgeFloor
	}
	if j.CouldBeUnder18 || j.ApparentAge < minAge {
		return false
	}
	rank := map[string]int{"low": 1, "medium": 2, "high": 3}
	want := rank[strings.ToLower(strings.TrimSpace(minConfidence))]
	if want == 0 {
		want = 2
	}
	got := rank[strings.ToLower(strings.TrimSpace(j.Confidence))]
	// Low (or unknown) confidence is never evidence of an adult, whatever the
	// profile's minimum says. The minimum can raise the bar, never lower it below
	// medium.
	need := want
	if need < 2 {
		need = 2
	}
	return got >= need
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
	profile SecondOpinionProfile
}

// NewSecondOpinionReviewer builds the wrapper with the default profile. A nil
// judge returns the primary unchanged.
func NewSecondOpinionReviewer(primary MinorReviewer, judge AgeJudge) MinorReviewer {
	return NewSecondOpinionReviewerWithProfile(primary, judge, DefaultReviewProfile(TierLocal).SecondOpinion)
}

// NewSecondOpinionReviewerWithProfile builds the wrapper from a resolved profile.
// A disabled profile, a nil judge, or a nil primary returns the primary unchanged.
func NewSecondOpinionReviewerWithProfile(primary MinorReviewer, judge AgeJudge, profile SecondOpinionProfile) MinorReviewer {
	if primary == nil || judge == nil || !profile.Enabled {
		return primary
	}
	return &SecondOpinionReviewer{primary: primary, judge: judge, profile: profile}
}

// Health delegates to the primary reviewer so discovery sees the real state.
func (r *SecondOpinionReviewer) Health() ReviewHealth {
	if h, ok := r.primary.(reviewerHealthReporter); ok {
		return h.Health()
	}
	return ReviewHealth{Configured: true, Available: true, Detail: "configured"}
}

func (r *SecondOpinionReviewer) Available() bool {
	if a, ok := r.primary.(reviewerAvailability); ok {
		return a.Available()
	}
	return r.primary != nil
}

func (r *SecondOpinionReviewer) Review(ctx context.Context, images [][]byte) (MinorVerdict, error) {
	verdict, err := r.primary.Review(ctx, images)
	if err != nil || !verdict.Refuses() {
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
		if judgeErr != nil || !judgement.ClearsAdultWith(r.profile.MinAge, r.profile.MinConfidence) {
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
			verdict.Images[i].Action = ActionFlag
			verdict.Images[i].ReasonCode = ReasonSexualAdult
			verdict.Images[i].Detail = "cleared by second opinion"
		}
	}
	verdict.Minor = false
	verdict.Action = ActionFlag
	verdict.ReasonCode = ReasonSexualAdult
	verdict.ReasonCodes = []ReasonCode{ReasonSexualAdult}
	verdict.Detail = "passed after second opinion: " + strings.Join(cleared, "; ")
	return verdict, nil
}
