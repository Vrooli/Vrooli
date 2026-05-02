/**
 * HowToChooseDialog
 *
 * Decision support dialog reachable from the picker (header link) and the
 * details page (Compare-to-other-modes button). Renders a top decision flow
 * (interactive yes/no traversal) and a bottom side-by-side matrix driven
 * directly from the catalog response.
 */

import { Dialog } from "../../ui/dialog";
import { selectors } from "../../../consts/selectors";
import type { InitiativeOperatingMode } from "../../../types";
import type { OperatingModeCatalogEntry } from "../../../types/operating-mode";
import { DecisionFlow } from "./decision-flow";
import { ModeMatrix } from "./mode-matrix";

export interface HowToChooseDialogProps {
  isOpen: boolean;
  onClose: () => void;
  catalog: OperatingModeCatalogEntry[];
  /**
   * Optional callback the picker entry point uses to advance its
   * `selectedModeKey`. Omitting it makes the recommendation read-only — used
   * by the details-page entry point.
   */
  onPickRecommendation?: (mode: InitiativeOperatingMode) => void;
}

export function HowToChooseDialog({
  isOpen,
  onClose,
  catalog,
  onPickRecommendation,
}: HowToChooseDialogProps) {
  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="How to choose an operating mode"
      maxWidth="max-w-4xl"
      testId={selectors.initiativeDetails.howToChooseDialog}
    >
      <div className="space-y-6 text-sm text-slate-300">
        <section className="space-y-2">
          <p className="leading-relaxed text-slate-300">
            Walk through a few questions about the work shape — the answers
            point at the right mode for this initiative. The right mode is a
            property of <em>how the work is shaped</em>, not its size.
          </p>
          <DecisionFlow
            catalog={catalog}
            onAccept={(mode) => {
              if (onPickRecommendation) {
                onPickRecommendation(mode);
              }
              onClose();
            }}
          />
        </section>

        <section className="space-y-2">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-400">
            Side-by-side
          </h3>
          <ModeMatrix catalog={catalog} />
        </section>
      </div>
    </Dialog>
  );
}
