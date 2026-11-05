package authoring

type WorkItemKind string

const (
	WorkItemFinalized     WorkItemKind = "finalized"
	WorkItemSection       WorkItemKind = "section"
	WorkItemGlobalContext WorkItemKind = "global_context"
	WorkItemPhase         WorkItemKind = "phase"
	WorkItemViolation     WorkItemKind = "violation"
	WorkItemReview        WorkItemKind = "review"
)

// WorkItem is the explicit authoring wizard navigation result. It is the single
// place that encodes the order a session resolves: finalized -> mandatory
// sections -> global context -> phase drafts -> violations -> final review.
type WorkItem struct {
	Kind       WorkItemKind
	Section    Section
	Phase      PhaseDraft
	Violations []StructureViolation
	Step       GuidedStep
	Ready      bool
}

func selectWorkItem(sess Session, violations []StructureViolation) WorkItem {
	if sess.Finalized {
		return WorkItem{
			Kind: WorkItemFinalized,
			Step: stepForFinalizedPlan(sess, sess.PlanID, sess.Slug),
		}
	}
	if key := firstUnfilledMandatory(sess.Sections); key != "" {
		if sec, ok := sectionByKey(sess.Sections, key); ok {
			return WorkItem{
				Kind:    WorkItemSection,
				Section: sec,
				Step:    stepForSection(sess, sec),
			}
		}
	}
	if !globalContextResolved(sess) || !globalSkillContextResolved(sess) {
		sec := Section{Key: SectionRelevantContext, Label: "Relevant context"}
		if found, ok := sectionByKey(sess.Sections, SectionRelevantContext); ok {
			sec = found
		}
		return WorkItem{
			Kind:    WorkItemGlobalContext,
			Section: sec,
			Step:    stepForGlobalContextCheckpoint(sess),
		}
	}
	if id := nextIncompletePhaseID(sess.PhaseDrafts); id != "" {
		if phase, ok := findDraft(sess.PhaseDrafts, id); ok {
			return WorkItem{
				Kind:       WorkItemPhase,
				Phase:      phase,
				Violations: phaseViolations(phase),
				Step:       stepForPhase(sess, phase),
			}
		}
	}
	if len(violations) > 0 {
		return WorkItem{
			Kind:       WorkItemViolation,
			Violations: violations,
			Step:       stepForValidation(sess, false, violations),
		}
	}
	return WorkItem{
		Kind:  WorkItemReview,
		Step:  stepForReview(sess),
		Ready: true,
	}
}
