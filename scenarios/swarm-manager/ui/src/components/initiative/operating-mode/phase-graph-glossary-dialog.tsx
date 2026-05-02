/**
 * PhaseGraphGlossaryDialog
 *
 * Thin configured wrapper around <ConceptExplainerDialog>. The legend's
 * existing testId (selectors.initiativeDetails.phaseGraphGlossaryDialog) is
 * preserved here so existing call sites and tests continue to resolve. New
 * "explain a concept" surfaces should use ConceptExplainerDialog directly
 * with their own static config.
 */

import { selectors } from "../../../consts/selectors";
import {
  ConceptExplainerDialog,
  type ConceptExplainerSection,
} from "../../ui/concept-explainer-dialog";

interface PhaseGraphGlossaryDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

const SECTIONS: ConceptExplainerSection[] = [
  {
    heading: "Nodes",
    label: "start",
    swatch: <span className="inline-block h-3 w-3 rounded-full bg-emerald-400" />,
    body: "The phase the mode begins from on the very first round. Every operating mode declares exactly one start phase.",
  },
  {
    heading: "Nodes",
    label: "terminal",
    swatch: <span className="inline-block h-3 w-3 rounded-full bg-violet-400" />,
    body: "A phase that closes the mode's loop. Reaching a terminal phase ends the current cycle; the operator either accepts or restarts the mode from the start phase.",
  },
  {
    heading: "Nodes",
    label: "selected",
    swatch: <span className="inline-block h-3 w-3 rounded-full bg-cyan-400" />,
    body: "The phase you've focused in the list below. Click a node in the graph to scroll its phase card into view.",
  },
  {
    heading: "Edges",
    label: "always",
    swatch: <span className="inline-block h-0.5 w-6 rounded-sm bg-slate-400" />,
    body: "An unconditional transition. When the source phase completes successfully, the next phase always becomes the source phase's downstream target.",
  },
  {
    heading: "Edges",
    label: "payload bool",
    swatch: <span className="inline-block h-0.5 w-6 rounded-sm bg-amber-400" />,
    body: "A transition that fires when the source phase's structured result includes a specific boolean key (for example, holistic-loop's execute → investigate edge fires when the phase output sets payload.replan_needed = true).",
  },
  {
    heading: "Edges",
    label: "progress decision",
    swatch: <span className="inline-block h-0.5 w-6 rounded-sm bg-cyan-400" />,
    body: "A transition driven by an explicit progress decision in the phase output (continue / replan / complete). Used by phased-plan-drain's classify_progress phase to fan out into the next iteration.",
  },
  {
    heading: "Concepts",
    label: "Phase",
    body: "A single node in the graph: one logical step the mode runs (investigate, plan, execute, review). Each phase has its own prompt, profile, and output contract.",
  },
  {
    heading: "Concepts",
    label: "Profile",
    body: "An agent-manager profile that defines which model, tools, and runtime budget the agent uses while running the phase. The profile-key chip on each phase card names the profile for that phase.",
  },
  {
    heading: "Concepts",
    label: "Skill",
    body: "A prompt-manager skill — the actual instructions the agent receives. The skill ID on each phase card resolves to the markdown the agent reads when the phase starts.",
  },
];

export function PhaseGraphGlossaryDialog({ isOpen, onClose }: PhaseGraphGlossaryDialogProps) {
  return (
    <ConceptExplainerDialog
      isOpen={isOpen}
      onClose={onClose}
      title="How to read this phase graph"
      sections={SECTIONS}
      testId={selectors.initiativeDetails.phaseGraphGlossaryDialog}
    />
  );
}
