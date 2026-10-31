package deployability

import (
	"errors"
	"strings"
	"testing"
)

func TestParsePlatformStatusAcceptsEveryAuthoredToken(t *testing.T) {
	cases := []struct {
		raw           string
		status        PlatformStatus
		qualification Qualification
	}{
		{raw: "supported", status: StatusSupported, qualification: QualificationQualified},
		{raw: "build-verified", status: StatusBuildVerified, qualification: QualificationBuildVerified},
		{raw: "experimental", status: StatusExperimental, qualification: QualificationUnqualified},
		{raw: "unqualified", status: StatusUnqualified, qualification: QualificationUnqualified},
		{raw: "partial", status: StatusPartial, qualification: QualificationDegraded},
		{raw: "unsupported", status: StatusUnsupported, qualification: QualificationIneligible},
		{raw: "not_implemented", status: StatusNotImplemented, qualification: QualificationDegraded},
		{raw: "not_applicable", status: StatusNotApplicable, qualification: QualificationIneligible},
		{raw: "  Build-Verified ", status: StatusBuildVerified, qualification: QualificationBuildVerified},
	}
	for _, testCase := range cases {
		t.Run(testCase.raw, func(t *testing.T) {
			status, err := ParsePlatformStatus(testCase.raw)
			if err != nil {
				t.Fatalf("expected %q to parse, got %v", testCase.raw, err)
			}
			if status != testCase.status {
				t.Fatalf("expected status %q, got %q", testCase.status, status)
			}
			if got := status.Qualification(); got != testCase.qualification {
				t.Fatalf("expected qualification %q, got %q", testCase.qualification, got)
			}
			if strings.TrimSpace(status.Qualification().Reason()) == "" {
				t.Fatalf("qualification %q carries no reason", status.Qualification())
			}
		})
	}
}

func TestParsePlatformStatusRejectsUnknownTokens(t *testing.T) {
	// "available" and "bundled" are authored under tier_feasibility artifacts,
	// a different vocabulary entirely. Seeing one as a platform status means a
	// manifest mixed the two, which must be an error rather than a downgrade.
	for _, token := range []string{"", "available", "bundled", "yes", "SUPPORTED?"} {
		t.Run(token, func(t *testing.T) {
			status, err := ParsePlatformStatus(token)
			if err == nil {
				t.Fatalf("expected %q to be rejected, got status %q", token, status)
			}
			if status != "" {
				t.Fatalf("rejected token resolved to status %q", status)
			}
			var unknown UnknownPlatformStatusError
			if !errors.As(err, &unknown) {
				t.Fatalf("expected UnknownPlatformStatusError, got %T", err)
			}
			if !strings.Contains(err.Error(), token) {
				t.Fatalf("error %q does not name the offending token %q", err, token)
			}
		})
	}
}

func TestQualificationLadderIsOrdered(t *testing.T) {
	ascending := []Qualification{
		QualificationUndeclared,
		QualificationIneligible,
		QualificationDegraded,
		QualificationUnqualified,
		QualificationBuildVerified,
		QualificationQualified,
	}
	for index := 1; index < len(ascending); index++ {
		lower, higher := ascending[index-1], ascending[index]
		if !(lower.Rank() < higher.Rank()) {
			t.Fatalf("%q (%d) is not ranked below %q (%d)", lower, lower.Rank(), higher, higher.Rank())
		}
		if lower.AtLeast(higher) {
			t.Fatalf("%q must not satisfy an at-least-%q floor", lower, higher)
		}
		if !higher.AtLeast(lower) {
			t.Fatalf("%q must satisfy an at-least-%q floor", higher, lower)
		}
	}
	if !QualificationQualified.AtLeast(QualificationBuildVerified) {
		t.Fatal("qualified must clear a build-verified floor")
	}
	if QualificationUnqualified.AtLeast(QualificationBuildVerified) {
		t.Fatal("unqualified must not clear a build-verified floor")
	}
	for _, rung := range ascending {
		if strings.TrimSpace(rung.Reason()) == "" {
			t.Fatalf("rung %q carries no human reason", rung)
		}
	}
}

func TestPlatformStatusesCoverTheWholeVocabulary(t *testing.T) {
	if len(PlatformStatuses()) != len(platformStatusQualifications) {
		t.Fatalf("PlatformStatuses() lists %d of %d vocabulary members", len(PlatformStatuses()), len(platformStatusQualifications))
	}
	for _, status := range PlatformStatuses() {
		if _, err := ParsePlatformStatus(string(status)); err != nil {
			t.Fatalf("listed status %q does not parse: %v", status, err)
		}
	}
}
