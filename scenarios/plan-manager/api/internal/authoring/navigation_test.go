package authoring

import (
	"testing"

	planmodel "plan-manager/internal/planmodel"

	"github.com/stretchr/testify/require"
)

func TestSelectWorkItemOrdersAuthoringStateMachine(t *testing.T) {
	sections := newSkeleton()
	prefillWorkPosture(sections)
	sess := Session{ID: "sess", Slug: "sess", Title: "Navigator", Sections: sections}

	require.Equal(t, WorkItemSection, selectWorkItem(sess, nil).Kind)
	require.Equal(t, SectionPurpose, selectWorkItem(sess, nil).Section.Key)

	fillMandatorySections(&sess)
	require.Equal(t, WorkItemGlobalContext, selectWorkItem(sess, nil).Kind)

	_, idx := sectionForTest(sess.Sections, SectionRelevantContext)
	sess.Sections[idx].Content = "NO_CONTEXT: fixture"
	sess.Sections[idx].Filled = true
	require.Equal(t, WorkItemPhase, selectWorkItem(sess, nil).Kind)

	sess.PhaseDrafts[0].References = []planmodel.Reference{{Kind: planmodel.ReferenceCode, Target: "scenarios/plan-manager/api/main.go"}}
	sess.PhaseDrafts[0].Steps = []string{"do it"}
	sess.PhaseDrafts[0].Validation = "go test ./..."
	sess.PhaseDrafts[0].Acceptance = "tests pass"
	sess.PhaseDrafts[0].RelevantContext = noteContextItemsFromLines("NO_CONTEXT: fixture", sess.PhaseDrafts[0].ID)
	require.Equal(t, WorkItemViolation, selectWorkItem(sess, []StructureViolation{{SectionKey: SectionReferences, Message: "late seam issue"}}).Kind)
	require.Equal(t, WorkItemReview, selectWorkItem(sess, nil).Kind)
}

func TestSectionCatalogSeedsSkeletonAndGuidance(t *testing.T) {
	sections := newSkeleton()

	require.Len(t, sections, len(defaultSkeleton))
	for i, spec := range defaultSkeleton {
		require.Equal(t, spec.Key, sections[i].Key)
		require.Equal(t, spec.Label, sections[i].Label)
		require.Equal(t, spec.Mandatory, sections[i].Mandatory)
		require.NotEmpty(t, spec.StepKind, "section %s must declare a guided step kind", spec.Key)
		require.NotEmpty(t, spec.Summary, "section %s must declare guidance summary", spec.Key)
		require.NotEmpty(t, spec.Instructions, "section %s must declare instructions", spec.Key)
		require.NotEmpty(t, spec.RequiredInputs, "section %s must declare required inputs", spec.Key)
		require.NotEmpty(t, spec.Placeholder, "section %s must declare a guided content placeholder", spec.Key)

		step := sectionBaseStep(spec.Key)
		require.Equal(t, spec.StepKind, step.StepKind)
		require.Equal(t, firstNonEmpty(spec.Title, spec.Label), step.Title)
		require.Equal(t, spec.Placeholder, contentPlaceholderForSection(spec.Key))
	}
}

func fillMandatorySections(sess *Session) {
	for i := range sess.Sections {
		if sess.Sections[i].Mandatory {
			sess.Sections[i].Content = "filled"
			sess.Sections[i].Filled = true
		}
	}
	sess.PhaseDrafts = []PhaseDraft{{
		ID:     "phase-1",
		Order:  1,
		Title:  "Phase",
		Intent: "Do the work",
	}}
}

func sectionForTest(sections []Section, key SectionKey) (Section, int) {
	for i, sec := range sections {
		if sec.Key == key {
			return sec, i
		}
	}
	return Section{}, -1
}
