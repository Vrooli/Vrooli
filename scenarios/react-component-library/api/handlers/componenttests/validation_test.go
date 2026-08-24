package componenttests

import (
	"testing"

	domain "react-component-library/internal/componenttests"

	"github.com/stretchr/testify/require"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestComponentTestsAssessmentWithFailuresIsJSONSerializable(t *testing.T) {
	_, err := protojson.Marshal(componentTestsAssessment("react-component-library", false, []string{"react-component-library:Textarea"}))
	require.NoError(t, err)
}

func TestComponentTestsAssessmentUsesCanonicalPresentationProjection(t *testing.T) {
	clean := componentTestsAssessment("react-component-library", true, nil)
	require.Equal(t, "v1", clean.GetPresentation().GetContractVersion())
	require.Equal(t, "L1", clean.GetPresentation().GetCurrentLevel())
	require.Equal(t, "Contracts validated", clean.GetPresentation().GetCurrentLevelLabel())
	require.Equal(t, "Contracts validated", clean.GetPresentation().GetNorthStar())
	require.True(t, clean.GetPresentation().GetAtMaximum())
	require.Empty(t, clean.GetPresentation().GetDocumentationTopics())

	failed := componentTestsAssessment("react-component-library", false, []string{"zeta", "alpha"})
	require.Equal(t, []string{"COMPONENT_TEST_FAILED"}, failed.GetPresentation().GetBlockingFindingCodes())
	require.Equal(t, []string{
		"component-tests maturity next move",
		"component-tests COMPONENT_TEST_FAILED canonical fix",
	}, failed.GetPresentation().GetDocumentationTopics())
	require.False(t, failed.GetPresentation().GetAtMaximum())
	require.Len(t, failed.GetPresentation().GetCapabilities(), 1)
	findings := failed.GetPresentation().GetCapabilities()[0].GetFindings()
	require.Len(t, findings, 1)
	require.Equal(t, int32(2), findings[0].GetCount())
	require.Equal(t, []string{"alpha", "zeta"}, findings[0].GetLocations())
	require.Equal(t, commonv1.FixAffordance_FIX_AFFORDANCE_MANUAL, findings[0].GetFixAffordance())
}

func TestComponentValidationWorkersLeaveBASSessionHeadroom(t *testing.T) {
	// BAS defaults to ten active browser sessions. Component validation can run
	// beside other capture consumers, so the catalog provider must not consume
	// the entire driver budget by itself.
	require.Less(t, componentValidationWorkers, 10)
	require.Greater(t, componentValidationWorkers, 0)
}

func TestHasReusablePassedReportRequiresExactImmutableClosure(t *testing.T) {
	reports := []domain.Report{
		{RootLibraryID: "rcl:Card", RootVersion: "1.2.0", IncludeClosure: false, Verdict: domain.VerdictPassed},
		{RootLibraryID: "rcl:Card", RootVersion: "1.1.0", IncludeClosure: true, Verdict: domain.VerdictPassed},
		{RootLibraryID: "rcl:Card", RootVersion: "1.2.0", IncludeClosure: true, Verdict: domain.VerdictFailed},
	}
	require.False(t, hasReusablePassedReport(reports, "rcl:Card", "1.2.0"))
	reports = append(reports, domain.Report{RootLibraryID: "rcl:Card", RootVersion: "1.2.0", IncludeClosure: true, Verdict: domain.VerdictPassed})
	require.True(t, hasReusablePassedReport(reports, "rcl:Card", "1.2.0"))
}
