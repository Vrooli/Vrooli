/**
 * DecisionFlow
 *
 * Renders an interactive yes/no traversal of the static decision-flow config
 * and lands the operator on a recommended mode. Modes referenced in the
 * config are validated against the live catalog at render time — unknown
 * modes render a visible error chip rather than silently failing. This is
 * greenfield-acceptable: failure is loud, not silent.
 */

import { useMemo, useState } from "react";
import { ArrowLeft, RotateCcw } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import type { InitiativeOperatingMode } from "../../../types";
import type { OperatingModeCatalogEntry } from "../../../types/operating-mode";
import { Button } from "../../ui/button";
import { OperatingModeCard } from "./operating-mode-card";
import {
  DECISION_FLOW,
  DECISION_FLOW_ROOT_ID,
  findQuestion,
  type DecisionFlowNodeRef,
} from "./decision-flow.config";

export interface DecisionFlowProps {
  catalog: OperatingModeCatalogEntry[];
  /**
   * Called when the operator picks the recommendation. The picker entry point
   * uses this to advance its own selectedModeKey; the details-page entry
   * point can navigate to the mode detail.
   */
  onAccept?: (mode: InitiativeOperatingMode) => void;
}

interface UnknownModeChipProps {
  mode: string;
}

function UnknownModeChip({ mode }: UnknownModeChipProps) {
  return (
    <p className="rounded-md border border-red-500/40 bg-red-500/10 px-3 py-2 text-sm text-red-200">
      Decision flow references unknown mode <code className="font-mono">{mode}</code> — not in the catalog.
      Update <code className="font-mono">decision-flow.config.ts</code> to match registered modes.
    </p>
  );
}

export function DecisionFlow({ catalog, onAccept }: DecisionFlowProps) {
  const catalogByMode = useMemo(() => {
    const map = new Map<string, OperatingModeCatalogEntry>();
    for (const entry of catalog) {
      map.set(entry.mode, entry);
    }
    return map;
  }, [catalog]);

  const referencedModes = useMemo(() => {
    const set = new Set<string>();
    for (const question of DECISION_FLOW) {
      for (const ref of [question.yes, question.no]) {
        if (ref.kind === "mode") set.add(ref.mode);
      }
    }
    return Array.from(set);
  }, []);

  const unknownReferences = referencedModes.filter((mode) => !catalogByMode.has(mode));

  const [trail, setTrail] = useState<string[]>([DECISION_FLOW_ROOT_ID]);
  const [terminal, setTerminal] = useState<DecisionFlowNodeRef | null>(null);

  const reset = () => {
    setTrail([DECISION_FLOW_ROOT_ID]);
    setTerminal(null);
  };

  const back = () => {
    if (terminal) {
      setTerminal(null);
      return;
    }
    if (trail.length <= 1) return;
    setTrail((prev) => prev.slice(0, -1));
  };

  const followAnswer = (answer: "yes" | "no") => {
    const currentId = trail[trail.length - 1];
    if (!currentId) return;
    const current = findQuestion(currentId);
    if (!current) return;
    const next = current[answer];
    if (next.kind === "mode") {
      setTerminal(next);
      return;
    }
    setTrail((prev) => [...prev, next.id]);
  };

  const currentId = trail[trail.length - 1] ?? null;
  const currentQuestion = currentId ? findQuestion(currentId) : null;

  return (
    <div
      className="space-y-3"
      data-testid={selectors.initiativeDetails.howToChooseDecisionFlow}
    >
      {unknownReferences.map((mode) => (
        <UnknownModeChip key={mode} mode={mode} />
      ))}

      {terminal && terminal.kind === "mode" ? (
        <RecommendationCard
          mode={terminal.mode}
          entry={catalogByMode.get(terminal.mode)}
          onAccept={onAccept}
          onBack={back}
          onReset={reset}
        />
      ) : currentQuestion ? (
        <div className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/40 p-4">
          <p className="text-sm font-medium text-slate-100">{currentQuestion.question}</p>
          {currentQuestion.hint ? (
            <p className="text-xs text-slate-400">{currentQuestion.hint}</p>
          ) : null}
          <div className="flex flex-wrap gap-2">
            <Button type="button" size="sm" onClick={() => followAnswer("yes")}>
              Yes
            </Button>
            <Button type="button" size="sm" variant="outline" onClick={() => followAnswer("no")}>
              No
            </Button>
            {trail.length > 1 ? (
              <Button type="button" size="sm" variant="ghost" onClick={back}>
                <ArrowLeft className="mr-1 h-3.5 w-3.5" /> Back
              </Button>
            ) : null}
          </div>
        </div>
      ) : (
        <p className="text-sm italic text-slate-500">
          Decision flow has no questions configured.
        </p>
      )}
    </div>
  );
}

function RecommendationCard({
  mode,
  entry,
  onAccept,
  onBack,
  onReset,
}: {
  mode: InitiativeOperatingMode;
  entry: OperatingModeCatalogEntry | undefined;
  onAccept?: (mode: InitiativeOperatingMode) => void;
  onBack: () => void;
  onReset: () => void;
}) {
  if (!entry) {
    return <UnknownModeChip mode={mode} />;
  }
  return (
    <div className="space-y-3 rounded-lg border border-cyan-500/30 bg-cyan-500/5 p-4">
      <div>
        <p className="text-[11px] font-semibold uppercase tracking-wide text-cyan-300">
          Recommended
        </p>
        <p className="mt-1 text-sm text-slate-300">
          Based on your answers, this is likely the right mode for the work shape.
        </p>
      </div>
      <OperatingModeCard mode={entry} selected />
      <div className="flex flex-wrap gap-2">
        {onAccept ? (
          <Button type="button" size="sm" onClick={() => onAccept(mode)}>
            Pick this mode
          </Button>
        ) : null}
        <Button type="button" size="sm" variant="ghost" onClick={onBack}>
          <ArrowLeft className="mr-1 h-3.5 w-3.5" /> Back
        </Button>
        <Button type="button" size="sm" variant="ghost" onClick={onReset}>
          <RotateCcw className="mr-1 h-3.5 w-3.5" /> Start over
        </Button>
      </div>
    </div>
  );
}
