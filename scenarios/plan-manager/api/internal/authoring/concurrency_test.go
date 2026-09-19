package authoring_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"plan-manager/internal/authoring"
	planmodel "plan-manager/internal/planmodel"

	"github.com/stretchr/testify/require"
)

// TestConcurrentSectionWritesBothSurvive is the lost-write regression guard for
// the session-lock discipline: two concurrent SubmitSection calls on different
// keys must both persist. Before withSessionLock, SubmitSection did a bare
// load-modify-save, so pipelined CLI calls could silently drop one section.
// Run with -race.
func TestConcurrentSectionWritesBothSurvive(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})

	for i := 0; i < 25; i++ {
		sess, _, err := svc.StartSession(ctx, fmt.Sprintf("Concurrent sections %d", i), "", "")
		require.NoError(t, err)

		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _, _, errs[0] = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Purpose written concurrently.")
		}()
		go func() {
			defer wg.Done()
			_, _, _, errs[1] = svc.SubmitSection(ctx, sess.ID, authoring.SectionProblemStatement, "Problem written concurrently.")
		}()
		wg.Wait()
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])

		final, _, err := svc.GetSession(ctx, sess.ID)
		require.NoError(t, err)
		requireSectionContent(t, final, authoring.SectionPurpose, "Purpose written concurrently.")
		requireSectionContent(t, final, authoring.SectionProblemStatement, "Problem written concurrently.")
	}
}

// TestConcurrentSectionAndPhaseFieldWritesBothSurvive covers the exact reported
// mechanism: a section write racing a phase-field write (the latter was locked,
// the former was not) must not lose either mutation.
func TestConcurrentSectionAndPhaseFieldWritesBothSurvive(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})

	for i := 0; i < 25; i++ {
		sess, _, err := svc.StartSession(ctx, fmt.Sprintf("Concurrent mixed %d", i), "", "")
		require.NoError(t, err)
		_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Race phase", "Survive the race.")
		require.NoError(t, err)

		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _, _, errs[0] = svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Purpose during phase write.")
		}()
		go func() {
			defer wg.Done()
			_, _, _, errs[1] = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldSteps, "Step one\nStep two")
		}()
		wg.Wait()
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])

		final, _, err := svc.GetSession(ctx, sess.ID)
		require.NoError(t, err)
		requireSectionContent(t, final, authoring.SectionPurpose, "Purpose during phase write.")
		require.Len(t, final.PhaseDrafts, 1)
		require.Equal(t, "Step one\nStep two", strings.Join(final.PhaseDrafts[0].Steps, "\n"))
	}
}

// TestConcurrentContextItemWritesBothSurvive locks in the same guarantee for
// the context path: two concurrent global context submissions both land.
func TestConcurrentContextItemWritesBothSurvive(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})

	for i := 0; i < 25; i++ {
		sess, _, err := svc.StartSession(ctx, fmt.Sprintf("Concurrent context %d", i), "", "")
		require.NoError(t, err)

		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		for j := 0; j < 2; j++ {
			go func(j int) {
				defer wg.Done()
				item := planmodel.RelevantContextItem{
					Kind:        planmodel.RelevantContextDoc,
					Label:       fmt.Sprintf("Doc %d", j),
					Reason:      "concurrency fixture",
					Instruction: "read",
					Target:      fmt.Sprintf("docs/concepts/DOC-%d.md", j),
				}
				_, _, _, _, errs[j] = svc.SubmitRelevantContextItem(ctx, sess.ID, "", item)
			}(j)
		}
		wg.Wait()
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])

		final, _, err := svc.GetSession(ctx, sess.ID)
		require.NoError(t, err)
		require.Len(t, final.RelevantContext, 2, "both concurrent context submissions must survive")
	}
}

func requireSectionContent(t *testing.T, sess authoring.Session, key authoring.SectionKey, want string) {
	t.Helper()
	for _, section := range sess.Sections {
		if section.Key == key {
			require.Equal(t, want, section.Content, "section %s content", key)
			return
		}
	}
	t.Fatalf("section %s not found", key)
}
