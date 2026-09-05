package authoring

import (
	"reflect"
	"testing"

	planmodel "plan-manager/internal/planmodel"

	"github.com/stretchr/testify/require"
)

// TestSessionToPlanCoversEveryAuthoredPlanField makes the authoring->plan
// projection executable.
//
// sessionToPlan hand-copies roughly twenty authored Plan fields out of session
// sections, and phaseDraftsToPlanPhases hand-copies seventeen more onto each
// phase. Nothing connected those hand-copies to the Plan struct, so a field
// added to Plan and classified authored could be silently unreachable from the
// authoring wizard while still rendering and still affecting the content hash.
// That had already happened twice: FinalValidationCommands and plan-level
// RisksHazards were both unsettable by the wizard.
//
// This test fails the moment a new authored field is added to Plan without a
// corresponding section and projection line.
func TestSessionToPlanCoversEveryAuthoredPlanField(t *testing.T) {
	sess := fullyPopulatedSession(t)

	got, err := sessionToPlan(sess)
	require.NoError(t, err)

	typ := reflect.TypeOf(planmodel.Plan{})
	value := reflect.ValueOf(got)
	var unreachable []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if planmodel.PlanFieldClasses[field.Name] != planmodel.FieldClassAuthored {
			continue
		}
		if value.Field(i).IsZero() {
			unreachable = append(unreachable, field.Name)
		}
	}
	require.Emptyf(t, unreachable,
		"authored Plan field(s) %v cannot be produced by the authoring wizard: every authored field needs a section in section_catalog.go and a line in sessionToPlan",
		unreachable)
}

// fullyPopulatedSession fills every catalogued section and one complete phase
// draft, so a zero value in the projected plan means "unreachable", never
// "not supplied by this test".
func fullyPopulatedSession(t *testing.T) Session {
	t.Helper()
	sections := make([]Section, 0, len(defaultSkeleton))
	for _, spec := range defaultSkeleton {
		sections = append(sections, Section{Key: spec.Key, Content: sectionContentFor(spec.Key)})
	}
	return Session{
		ID:          "11111111-2222-4333-8444-555555555555",
		Title:       "Drift probe",
		Slug:        "drift-probe",
		Sections:    sections,
		PhaseDrafts: []PhaseDraft{fullyPopulatedPhaseDraft()},
	}
}

// sectionContentFor returns content that satisfies each section's own grammar.
// A section whose parser expects structure gets structure; everything else gets
// plain prose.
func sectionContentFor(key SectionKey) string {
	switch key {
	case SectionAssumptions:
		return "Execution runs on a captured baseline.\nThe baseline is captured first -> re-capture before executing"
	case SectionDecisions:
		return "Use the shared renderer grammar -> avoids a second drifting copy"
	case SectionDefinitions:
		return "Mirror — the rendered markdown view of a plan"
	case SectionValidationStrategy:
		return "Run the focused suites per phase.\n\n**Final validation commands:**\n\n- `vrooli scenario test plan-manager`"
	case SectionAcceptanceBoundary:
		return "- Allowed: scenarios/plan-manager/**"
	case SectionRegressionAnchor:
		return "- Strategy: change_boundary\n- Baseline: drift-probe-baseline"
	case SectionPhases:
		return "### Phase 1 - Drift probe phase\n\nProbe the projection.\n"
	case SectionReferences:
		return "- [CODE: scenarios/plan-manager/api/internal/authoring/plan_projection.go]"
	case SectionRelevantContext, SectionRequiredReading:
		return "NO_CONTEXT: drift probe supplies no external context."
	default:
		return "Drift probe content for " + string(key) + "."
	}
}

func fullyPopulatedPhaseDraft() PhaseDraft {
	draft := PhaseDraft{}
	populateExportedFields(reflect.ValueOf(&draft).Elem())
	return draft
}

func populateExportedFields(value reflect.Value) {
	switch value.Kind() {
	case reflect.String:
		value.SetString("drift")
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(1)
	case reflect.Float32, reflect.Float64:
		value.SetFloat(1)
	case reflect.Slice:
		item := reflect.New(value.Type().Elem()).Elem()
		populateExportedFields(item)
		value.Set(reflect.Append(value, item))
	case reflect.Map:
		value.Set(reflect.MakeMap(value.Type()))
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		populateExportedFields(value.Elem())
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if value.Type().Field(i).PkgPath == "" {
				populateExportedFields(value.Field(i))
			}
		}
	}
}
