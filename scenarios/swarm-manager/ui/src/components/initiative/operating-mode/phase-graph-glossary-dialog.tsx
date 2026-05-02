/**
 * PhaseGraphGlossaryDialog
 *
 * Explainer for the operating-mode phase graph. Opened by clicking the legend
 * row above the graph. Documents edge kinds, node kinds, and the underlying
 * concepts (phase / profile / skill) so new operators don't have to read the
 * registry source to understand what the visualization is showing.
 *
 * External-link CTAs (open profile in agent-manager, open skill in
 * prompt-manager) land in Phase 2 alongside the external-links service module.
 */

import { selectors } from "../../../consts/selectors";
import { Dialog } from "../../ui/dialog";

interface PhaseGraphGlossaryDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

interface GlossaryRow {
  swatch: React.ReactNode;
  label: string;
  body: string;
}

const NODE_ROWS: GlossaryRow[] = [
  {
    swatch: <span className="inline-block h-3 w-3 rounded-full bg-emerald-400" />,
    label: "start",
    body: "The phase the mode begins from on the very first round. Every operating mode declares exactly one start phase.",
  },
  {
    swatch: <span className="inline-block h-3 w-3 rounded-full bg-violet-400" />,
    label: "terminal",
    body: "A phase that closes the mode's loop. Reaching a terminal phase ends the current cycle; the operator either accepts or restarts the mode from the start phase.",
  },
  {
    swatch: <span className="inline-block h-3 w-3 rounded-full bg-cyan-400" />,
    label: "selected",
    body: "The phase you've focused in the list below. Click a node in the graph to scroll its phase card into view.",
  },
];

const EDGE_ROWS: GlossaryRow[] = [
  {
    swatch: <span className="inline-block h-0.5 w-6 rounded-sm bg-slate-400" />,
    label: "always",
    body: "An unconditional transition. When the source phase completes successfully, the next phase always becomes the source phase's downstream target.",
  },
  {
    swatch: <span className="inline-block h-0.5 w-6 rounded-sm bg-amber-400" />,
    label: "payload bool",
    body: 'A transition that fires when the source phase\'s structured result includes a specific boolean key (for example, holistic-loop\'s execute → investigate edge fires when the phase output sets payload.replan_needed = true).',
  },
  {
    swatch: <span className="inline-block h-0.5 w-6 rounded-sm bg-cyan-400" />,
    label: "progress decision",
    body: "A transition driven by an explicit progress decision in the phase output (continue / replan / complete). Used by phased-plan-drain's classify_progress phase to fan out into the next iteration.",
  },
];

const CONCEPT_ROWS: Array<{ term: string; body: string }> = [
  {
    term: "Phase",
    body: "A single node in the graph: one logical step the mode runs (investigate, plan, execute, review). Each phase has its own prompt, profile, and output contract.",
  },
  {
    term: "Profile",
    body: "An agent-manager profile that defines which model, tools, and runtime budget the agent uses while running the phase. The profile-key chip on each phase card names the profile for that phase.",
  },
  {
    term: "Skill",
    body: "A prompt-manager skill — the actual instructions the agent receives. The skill ID on each phase card resolves to the markdown the agent reads when the phase starts.",
  },
];

export function PhaseGraphGlossaryDialog({ isOpen, onClose }: PhaseGraphGlossaryDialogProps) {
  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="How to read this phase graph"
      maxWidth="max-w-2xl"
      testId={selectors.initiativeDetails.phaseGraphGlossaryDialog}
    >
      <div className="space-y-6 text-sm text-slate-300">
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
            Nodes
          </h3>
          <ul className="space-y-3">
            {NODE_ROWS.map((row) => (
              <li key={row.label} className="flex items-start gap-3">
                <span className="mt-1 flex w-16 shrink-0 items-center gap-2">
                  {row.swatch}
                  <span className="text-xs text-slate-200">{row.label}</span>
                </span>
                <p className="leading-relaxed text-slate-300">{row.body}</p>
              </li>
            ))}
          </ul>
        </section>

        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
            Edges
          </h3>
          <ul className="space-y-3">
            {EDGE_ROWS.map((row) => (
              <li key={row.label} className="flex items-start gap-3">
                <span className="mt-1 flex w-32 shrink-0 items-center gap-2">
                  {row.swatch}
                  <span className="text-xs text-slate-200">{row.label}</span>
                </span>
                <p className="leading-relaxed text-slate-300">{row.body}</p>
              </li>
            ))}
          </ul>
        </section>

        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
            Concepts
          </h3>
          <ul className="space-y-3">
            {CONCEPT_ROWS.map((row) => (
              <li key={row.term} className="grid grid-cols-[7rem_1fr] items-start gap-3">
                <span className="text-sm font-medium text-slate-100">{row.term}</span>
                <p className="leading-relaxed text-slate-300">{row.body}</p>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </Dialog>
  );
}
