package planlog

import (
	"testing"

	planmodel "plan-manager/internal/planmodel"

	"github.com/stretchr/testify/require"
)

func TestExplicitScenarioQABugReferenceRecognizesStoredSourceShapes(t *testing.T) {
	tests := []struct {
		name  string
		entry Entry
		want  string
	}{
		{
			name: "published reference in detail",
			entry: Entry{
				Type:   planmodel.LogEntryBugReport,
				Detail: "Published scenario-qa bug knw-1788689623135002151.",
			},
			want: "knw-1788689623135002151",
		},
		{
			name: "filed reference in detail",
			entry: Entry{
				Type:   planmodel.LogEntryBugReport,
				Detail: "Scenario QA bug knw-1788947949411237588 filed; evidence retained.",
			},
			want: "knw-1788947949411237588",
		},
		{
			name: "reference in structured bug description",
			entry: Entry{
				Type: planmodel.LogEntryBugReport,
				Bug: planmodel.BugReportPayload{
					Description: "Scenario QA bug knw-1788802962375904482. Out of scope for this brief.",
				},
			},
			want: "knw-1788802962375904482",
		},
		{
			name: "second structured bug description",
			entry: Entry{
				Type: planmodel.LogEntryBugReport,
				Bug: planmodel.BugReportPayload{
					Description: "Scenario QA bug knw-1788802962758069116. Out of scope for this brief.",
				},
			},
			want: "knw-1788802962758069116",
		},
		{
			name: "generic knowledge id is not enough",
			entry: Entry{
				Type:   planmodel.LogEntryBugReport,
				Detail: "Diagnostic mentions knw-1234 in a test fixture.",
			},
		},
		{
			name: "malformed knowledge id is not enough",
			entry: Entry{
				Type:   planmodel.LogEntryBugReport,
				Detail: "Scenario QA bug knw-123x.",
			},
		},
		{
			name: "non-bug report is not blocked",
			entry: Entry{
				Type:   planmodel.LogEntryRecord,
				Detail: "Scenario QA bug knw-1234.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, explicitScenarioQABugReference(tt.entry))
		})
	}
}
