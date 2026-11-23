package safety

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fixedReviewer struct{ verdict MinorVerdict }

func (f fixedReviewer) Review(context.Context, [][]byte) (MinorVerdict, error) { return f.verdict, nil }

type fixedJudge struct {
	judgement AgeJudgement
	err       error
	calls     int
}

func (f *fixedJudge) Judge(context.Context, []byte) (AgeJudgement, error) {
	f.calls++
	return f.judgement, f.err
}

func faceless() MinorVerdict {
	return MinorVerdict{Minor: true, Sexual: true, Detail: "image 0: no readable face", Images: []MinorImageFinding{
		{NSFWScore: 1, Sexual: true, Unassessable: true, Refuse: true},
	}}
}

func TestSecondOpinionClearsOnlyAConfidentAdultWithoutAFace(t *testing.T) {
	cases := []struct {
		name      string
		judgement AgeJudgement
		err       error
		wantMinor bool
	}{
		{"confident adult clears", AgeJudgement{ApparentAge: 31, Confidence: "high"}, nil, false},
		{"at the floor clears", AgeJudgement{ApparentAge: AdultAgeFloor, Confidence: "medium"}, nil, false},
		{"just under the floor stays refused", AgeJudgement{ApparentAge: AdultAgeFloor - 1, Confidence: "high"}, nil, true},
		{"flagged possibly under 18 stays refused", AgeJudgement{ApparentAge: 30, CouldBeUnder18: true, Confidence: "high"}, nil, true},
		{"low confidence stays refused", AgeJudgement{ApparentAge: 40, Confidence: "low"}, nil, true},
		{"judge error stays refused", AgeJudgement{}, errors.New("gateway down"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			judge := &fixedJudge{judgement: tc.judgement, err: tc.err}
			got, err := NewSecondOpinionReviewer(fixedReviewer{faceless()}, judge).Review(context.Background(), [][]byte{{1}})
			if err != nil {
				t.Fatal(err)
			}
			if got.Minor != tc.wantMinor {
				t.Fatalf("Minor = %v, want %v (%s)", got.Minor, tc.wantMinor, got.Detail)
			}
			if !got.Sexual {
				t.Fatal("clearing the age concern must not hide that the image is sexual")
			}
			if !tc.wantMinor && !strings.Contains(got.Detail, "second opinion") {
				t.Fatalf("a cleared image must say why: %q", got.Detail)
			}
		})
	}
}

func TestSecondOpinionNeverOverridesAMinorAppearingFace(t *testing.T) {
	verdict := MinorVerdict{Minor: true, Sexual: true, Images: []MinorImageFinding{
		{NSFWScore: 1, Sexual: true, MinorFace: true, Refuse: true},
	}}
	judge := &fixedJudge{judgement: AgeJudgement{ApparentAge: 40, Confidence: "high"}}
	got, _ := NewSecondOpinionReviewer(fixedReviewer{verdict}, judge).Review(context.Background(), [][]byte{{1}})
	if !got.Minor || judge.calls != 0 {
		t.Fatalf("a face-based refusal must stand without asking the judge: minor=%v calls=%d", got.Minor, judge.calls)
	}
}

func TestSecondOpinionRefusesWholeRequestIfAnyImageIsNotCleared(t *testing.T) {
	verdict := MinorVerdict{Minor: true, Sexual: true, Images: []MinorImageFinding{
		{Sexual: true, Unassessable: true, Refuse: true},
		{Sexual: true, MinorFace: true, Refuse: true},
	}}
	judge := &fixedJudge{judgement: AgeJudgement{ApparentAge: 35, Confidence: "high"}}
	got, _ := NewSecondOpinionReviewer(fixedReviewer{verdict}, judge).Review(context.Background(), [][]byte{{1}, {2}})
	if !got.Minor {
		t.Fatal("one uncleared image must keep the request refused")
	}
}

func TestSecondOpinionLeavesPassingAndMissingFindingsAlone(t *testing.T) {
	judge := &fixedJudge{}
	pass := MinorVerdict{Sexual: true, Images: []MinorImageFinding{{Sexual: true}}}
	if got, _ := NewSecondOpinionReviewer(fixedReviewer{pass}, judge).Review(context.Background(), [][]byte{{1}}); got.Minor || judge.calls != 0 {
		t.Fatalf("a passing review must not consult the judge: %+v calls=%d", got, judge.calls)
	}
	noFindings := MinorVerdict{Minor: true, Sexual: true}
	if got, _ := NewSecondOpinionReviewer(fixedReviewer{noFindings}, judge).Review(context.Background(), [][]byte{{1}}); !got.Minor || judge.calls != 0 {
		t.Fatalf("a refusal without per-image findings must stand: %+v calls=%d", got, judge.calls)
	}
	if NewSecondOpinionReviewer(fixedReviewer{pass}, nil) == nil {
		t.Fatal("a nil judge must return the primary unchanged")
	}
}
