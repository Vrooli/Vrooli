package operatingmode

import "fmt"

// Branch coverage is the authoring-completeness signal behind the simulation
// walkthrough: every guarded or classified edge in a mode's phase graph is a
// branch an operator's mental model should be able to walk, and each one should
// be exercised by at least one example-run before the mode is trusted. This file
// enumerates a mode's branches and diffs them against the branches its
// example-runs actually walk, so the authoring surface can hand an author the
// exact list of branch example-runs still owed.

// branchKey uniquely identifies one guarded edge (a phase plus the index of the
// guard within that phase's ordered guard list).
func branchKey(phase Phase, guardIndex int) string {
	return fmt.Sprintf("%s#%d", phase, guardIndex)
}

// branchLabel renders an operator-facing description of one guarded edge:
// the source phase, the guard condition, and the destination (or "stop" for a
// guarded stop). It reuses GuardLabel so the wording matches the CLI/UI edge
// rendering exactly.
func branchLabel(phase Phase, gt GuardedTransition) string {
	dest := "stop"
	if len(gt.To) > 0 {
		dest = string(gt.To[0])
	}
	return fmt.Sprintf("%s: %s → %s", phase, GuardLabel(gt.When), dest)
}

// modeBranchCoverage enumerates a mode's guarded/classified branches and reports
// which are not walked by any of its example-runs. defs must contain every mode
// the target may delegate to (executed_by) so a composed mode's parent guards
// route correctly. The example-runs are assumed already validated (WalkExampleRun
// ran at load); this replay only records which parent guards fired.
func modeBranchCoverage(defs map[Mode]Definition, def Definition) ([]string, error) {
	covered := map[string]bool{}
	for _, run := range def.ExampleRuns {
		if err := recordRunBranchCoverage(defs, def, run, covered); err != nil {
			return nil, err
		}
	}
	var uncovered []string
	for _, phase := range orderedPhases(def) {
		guards := def.PhaseGraph.Guards[phase]
		for i, gt := range guards {
			if !covered[branchKey(phase, i)] {
				uncovered = append(uncovered, branchLabel(phase, gt))
			}
		}
	}
	return uncovered, nil
}

// recordRunBranchCoverage replays one example-run through the real guards and
// marks each parent guard index it fires. It mirrors WalkExampleRun's routing —
// including one-level executed_by delegation, where the sub-mode's continue
// self-loop fires no parent guard and only a sub-mode stop routes through the
// parent's guards — but records coverage instead of asserting the path.
func recordRunBranchCoverage(defs map[Mode]Definition, def Definition, run ExampleRun, covered map[string]bool) error {
	terminal := terminalSet(def)
	maxHops := len(run.ExpectedPath) + 1
	stepIdx := 0
	cur := def.PhaseGraph.StartPhase
	var subCur Phase

	for hop := 0; ; hop++ {
		if hop > maxHops {
			return fmt.Errorf("example-run %q did not terminate within %d hops (guards loop?)", run.ID, maxHops)
		}
		phaseDef, ok := def.PhaseGraph.Phases[cur]
		if !ok {
			return fmt.Errorf("example-run %q walked into unregistered phase %q", run.ID, cur)
		}
		if _, isTerminal := terminal[cur]; isTerminal {
			return nil
		}

		execPhaseDef := phaseDef
		var sub Definition
		if phaseDef.Delegated() {
			sub, ok = defs[phaseDef.ExecutedBy]
			if !ok {
				return fmt.Errorf("example-run %q phase %q delegates to unknown sub-mode %q", run.ID, cur, phaseDef.ExecutedBy)
			}
			if subCur == "" {
				subCur = sub.PhaseGraph.StartPhase
			}
			execPhaseDef, ok = sub.PhaseGraph.Phases[subCur]
			if !ok {
				return fmt.Errorf("example-run %q phase %q: sub-mode %q has no phase %q", run.ID, cur, sub.Mode, subCur)
			}
		}

		var output map[string]any
		if stepIdx < len(run.Steps) && Phase(run.Steps[stepIdx].Phase) == cur {
			output = run.Steps[stepIdx].Output
			stepIdx++
		}
		output, err := applyWalkClassification(run, cur, execPhaseDef, output)
		if err != nil {
			return err
		}

		if phaseDef.Delegated() {
			next, continuing, err := delegationRouteForLookup(sub, subCur, NewMapFieldLookup(output))
			if err != nil {
				return fmt.Errorf("example-run %q phase %q: %w", run.ID, cur, err)
			}
			if continuing {
				subCur = next
				continue
			}
			subCur = ""
		}

		idx, next, matched := selectGuard(def, cur, NewMapFieldLookup(output))
		if !matched {
			return fmt.Errorf("example-run %q: no guard out of %q routed onward", run.ID, cur)
		}
		covered[branchKey(cur, idx)] = true
		if len(next) == 0 {
			return nil
		}
		cur = next[0]
	}
}
