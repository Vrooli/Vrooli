package verdicts

import modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"

// ToProto renders a verdict for the wire. The models and AI discovery handlers
// both serve verdicts, so the conversion lives with the type.
func ToProto(v Verdict) *modelsv1.Verdict {
	out := &modelsv1.Verdict{Operation: v.Operation, Grade: GradeToProto(v.Grade), Reason: v.Reason}
	for _, e := range v.Evidence {
		out.Evidence = append(out.Evidence, &modelsv1.VerdictEvidence{Kind: e.Kind, Summary: e.Summary, Ref: e.Ref, Date: e.Date})
	}
	return out
}

// GradeToProto maps a grade to its wire enum; anything unknown is untested.
func GradeToProto(g Grade) modelsv1.VerdictGrade {
	switch g {
	case Recommended:
		return modelsv1.VerdictGrade_VERDICT_GRADE_RECOMMENDED
	case Works:
		return modelsv1.VerdictGrade_VERDICT_GRADE_WORKS
	case Experimental:
		return modelsv1.VerdictGrade_VERDICT_GRADE_EXPERIMENTAL
	case NotRecommended:
		return modelsv1.VerdictGrade_VERDICT_GRADE_NOT_RECOMMENDED
	case Broken:
		return modelsv1.VerdictGrade_VERDICT_GRADE_BROKEN
	default:
		return modelsv1.VerdictGrade_VERDICT_GRADE_UNTESTED
	}
}
