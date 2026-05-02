import { useMemo, useRef } from "react";
import { Activity, AlertTriangle, ChevronDown, ChevronRight, Play } from "lucide-react";
import { Button } from "../../ui/button";
import { Textarea } from "../../ui/textarea";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCatalogEntry,
  OperatingModeRound,
  OperatingModeWorkspace,
  OperatingModeWorkspacePhase,
} from "../../../types/operating-mode";
import { useAutoResizeTextarea } from "../../../hooks/useAutoResizeTextarea";
import { PhaseGraph, type PhaseStateMap } from "./phase-graph";
import {
  applyPhaseAction,
  type PhaseQuickActionKey,
} from "./phase-composer-envelope";
import { phaseLabel } from "./utils";

export interface PhaseComposerItem {
  ref: string;
  title?: string;
}

export interface PhaseComposerProps {
  catalogEntry: OperatingModeCatalogEntry;
  workspace: OperatingModeWorkspace;
  runningRound?: OperatingModeRound;
  items: PhaseComposerItem[];

  pendingPhase: string | null;
  onPendingPhaseChange: (phase: string | null) => void;

  selectedActions: Set<PhaseQuickActionKey>;
  onSelectedActionsChange: (next: Set<PhaseQuickActionKey>) => void;

  selectedItems: Set<string>;
  onSelectedItemsChange: (next: Set<string>) => void;

  pickerOpen: boolean;
  onPickerOpenChange: (open: boolean) => void;

  note: string;
  onNoteChange: (note: string) => void;

  phaseBusy: boolean;
  canRunPhases: boolean;
  startError?: unknown;

  onStart: (phase: string, envelopeNote: string) => void;
}

const QUICK_ACTIONS: ReadonlyArray<{
  key: PhaseQuickActionKey;
  label: string;
  hint: string;
  testId: string;
}> = [
  {
    key: "continue_from_prior",
    label: "Continue from prior",
    hint: "Use the latest round's state as input.",
    testId: selectors.initiativeDetails.phaseComposerActionContinue,
  },
  {
    key: "reset_and_reinvestigate",
    label: "Reset & re-investigate",
    hint: "Discard prior state, run from scratch.",
    testId: selectors.initiativeDetails.phaseComposerActionReset,
  },
  {
    key: "focus_on_items",
    label: "Focus on items",
    hint: "Open the picker and constrain the phase to selected items.",
    testId: selectors.initiativeDetails.phaseComposerActionFocus,
  },
  {
    key: "skip_unblock",
    label: "Skip / unblock",
    hint: "Bypass a blocked condition. Only available when the phase is currently not startable.",
    testId: selectors.initiativeDetails.phaseComposerActionSkip,
  },
  {
    key: "tighten_scope",
    label: "Tighten scope",
    hint: "Instruct the phase to narrow.",
    testId: selectors.initiativeDetails.phaseComposerActionTighten,
  },
  {
    key: "expand_scope",
    label: "Expand scope",
    hint: "Instruct the phase to broaden.",
    testId: selectors.initiativeDetails.phaseComposerActionExpand,
  },
];

export function PhaseComposer({
  catalogEntry,
  workspace,
  runningRound,
  items,
  pendingPhase,
  onPendingPhaseChange,
  selectedActions,
  onSelectedActionsChange,
  selectedItems,
  onSelectedItemsChange,
  pickerOpen,
  onPickerOpenChange,
  note,
  onNoteChange,
  phaseBusy,
  canRunPhases,
  startError,
  onStart,
}: PhaseComposerProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  useAutoResizeTextarea(textareaRef, note, { maxHeight: 240 });

  const workspacePhasesByName = useMemo(() => {
    const map = new Map<string, OperatingModeWorkspacePhase>();
    for (const phase of workspace.definition.phases) map.set(phase.phase, phase);
    return map;
  }, [workspace.definition.phases]);

  const phaseStates: PhaseStateMap = useMemo(() => {
    const out: PhaseStateMap = {};
    for (const phase of workspace.definition.phases) {
      out[phase.phase] = {
        startable: phase.startable,
        reason: phase.reason,
        isNext: phase.next,
      };
    }
    return out;
  }, [workspace.definition.phases]);

  const pendingPhaseDef = pendingPhase ? workspacePhasesByName.get(pendingPhase) : undefined;
  const pendingPhaseStartable = pendingPhaseDef?.startable === true;

  const toggleAction = (key: PhaseQuickActionKey) => {
    const next = applyPhaseAction(selectedActions, key);
    if (key === "focus_on_items" && next.has("focus_on_items") && !pickerOpen) {
      onPickerOpenChange(true);
    }
    if (key === "focus_on_items" && !next.has("focus_on_items")) {
      onSelectedItemsChange(new Set());
    }
    onSelectedActionsChange(next);
  };

  const actionEnabled = (key: PhaseQuickActionKey): boolean => {
    if (!pendingPhase) return false;
    if (key === "skip_unblock") return !pendingPhaseStartable && Boolean(pendingPhaseDef?.reason);
    return true;
  };

  const toggleItem = (ref: string) => {
    const next = new Set(selectedItems);
    if (next.has(ref)) next.delete(ref);
    else next.add(ref);
    onSelectedItemsChange(next);
  };

  const hasMeaningfulInput = note.trim().length > 0 || selectedActions.size > 0 || selectedItems.size > 0;
  const canSubmit =
    pendingPhase !== null &&
    canRunPhases &&
    !phaseBusy &&
    !runningRound &&
    pendingPhaseStartable &&
    hasMeaningfulInput;

  const handleStart = () => {
    if (!canSubmit || !pendingPhase) return;
    // Envelope is composed by the parent (via the workspace hook); we pass the
    // raw note here and let the hook decide whether to wrap it. Keeping the
    // composer envelope-agnostic means the hook can centralize the
    // buildPhaseEnvelope call and any future wire shape changes.
    onStart(pendingPhase, note);
  };

  return (
    <div
      className="space-y-3 rounded-xl border border-white/10 bg-slate-800/30 p-4"
      data-testid={selectors.initiativeDetails.phaseComposer}
    >
      {runningRound && (
        <div
          className="flex items-start gap-2 rounded-lg border border-cyan-500/30 bg-cyan-500/10 p-3 text-sm text-cyan-200"
          data-testid={selectors.initiativeDetails.phaseComposerActiveBanner}
        >
          <Activity className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
          <span>
            Round {runningRound.round} running in phase {phaseLabel(runningRound.phase)}.
            Wait or cancel from the timeline below to start another.
          </span>
        </div>
      )}

      <PhaseGraph
        entry={catalogEntry}
        mode="composer"
        rounds={workspace.rounds}
        phaseStates={phaseStates}
        selectedPhaseId={pendingPhase}
        onSelectPhase={(p) => onPendingPhaseChange(p)}
      />

      {pendingPhase && pendingPhaseDef && (
        <div
          className="flex flex-wrap items-center gap-2 rounded-lg border border-slate-800 bg-slate-950/40 p-3 text-xs text-slate-300"
          data-testid={selectors.initiativeDetails.phaseComposerSelectedPhaseStrip}
        >
          <span className="font-semibold text-slate-200">{phaseLabel(pendingPhase)}</span>
          <Chip>Profile: {pendingPhaseDef.profileKey}</Chip>
          {pendingPhaseDef.writesRepo ? (
            <Chip className="border-emerald-500/30 bg-emerald-500/10 text-emerald-300">writes repo</Chip>
          ) : (
            <Chip>read-only</Chip>
          )}
          {pendingPhaseDef.requiresCriteria && (
            <Chip className="border-amber-500/30 bg-amber-500/10 text-amber-300">requires criteria</Chip>
          )}
          {!pendingPhaseStartable && pendingPhaseDef.reason && (
            <span className="ml-1 flex items-center gap-1 text-amber-300">
              <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
              {pendingPhaseDef.reason}
            </span>
          )}
          {pendingPhaseDef.outputArtifacts && pendingPhaseDef.outputArtifacts.length > 0 && (
            <div className="mt-1 w-full text-[11px]">
              <span className="text-slate-500">Outputs: </span>
              <span className="font-mono text-slate-300">
                {pendingPhaseDef.outputArtifacts.map((a) => a.path).join(", ")}
              </span>
            </div>
          )}
        </div>
      )}

      <div className="flex flex-wrap gap-2">
        {QUICK_ACTIONS.map((action) => {
          const active = selectedActions.has(action.key);
          const enabled = actionEnabled(action.key);
          return (
            <button
              key={action.key}
              type="button"
              data-testid={action.testId}
              disabled={!enabled || phaseBusy}
              aria-pressed={active}
              onClick={() => toggleAction(action.key)}
              title={action.hint}
              className={`rounded-md border px-2.5 py-1.5 text-xs font-medium transition-colors ${
                active
                  ? "border-cyan-400/60 bg-cyan-500/20 text-cyan-200"
                  : "border-slate-700 bg-slate-800/60 text-slate-300 hover:border-slate-500"
              } disabled:cursor-not-allowed disabled:opacity-40`}
            >
              {action.label}
            </button>
          );
        })}
      </div>

      {selectedActions.has("focus_on_items") && items.length > 0 && (
        <div
          className="rounded-lg border border-slate-700 bg-slate-800/40"
          data-testid={selectors.initiativeDetails.phaseComposerItemPicker}
        >
          <button
            type="button"
            onClick={() => onPickerOpenChange(!pickerOpen)}
            aria-expanded={pickerOpen}
            data-testid={selectors.initiativeDetails.phaseComposerItemPickerToggle}
            className="flex w-full items-center justify-between px-3 py-2 text-left text-xs text-slate-200 hover:bg-slate-800/60"
          >
            <span>Target items — {selectedItems.size} of {items.length} selected</span>
            {pickerOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
          </button>
          {pickerOpen && (
            <ul className="max-h-48 space-y-0.5 overflow-y-auto border-t border-slate-700 p-2">
              {items.map((item) => (
                <li key={item.ref}>
                  <label
                    className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 text-[12px] text-slate-300 hover:bg-slate-800/60"
                    data-testid={selectors.initiativeDetails.phaseComposerItemPickerItem}
                    data-ref={item.ref}
                  >
                    <input
                      type="checkbox"
                      checked={selectedItems.has(item.ref)}
                      onChange={() => toggleItem(item.ref)}
                      disabled={phaseBusy}
                      className="h-3.5 w-3.5 accent-cyan-400"
                    />
                    <code className="text-cyan-300/80">{item.ref}</code>
                    {item.title && <span className="truncate text-slate-400">{item.title}</span>}
                  </label>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <Textarea
        ref={textareaRef}
        className="min-h-20"
        value={note}
        onChange={(event) => onNoteChange(event.target.value)}
        placeholder={pendingPhase ? "Optional note for the next phase run." : "Pick a phase above to start."}
        data-testid={selectors.initiativeDetails.phaseComposerNote}
      />

      <div className="flex items-center justify-end gap-2">
        <Button
          type="button"
          size="sm"
          onClick={handleStart}
          disabled={!canSubmit}
          data-testid={selectors.initiativeDetails.phaseComposerStart}
        >
          <Play className="mr-1.5 h-4 w-4" />
          {phaseBusy ? "Starting…" : pendingPhase ? `Start ${phaseLabel(pendingPhase)}` : "Start phase"}
        </Button>
      </div>

      {Boolean(startError) && (
        <p className="text-sm text-red-300">
          {startError instanceof Error ? startError.message : "Failed to start phase."}
        </p>
      )}
    </div>
  );
}

function Chip({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <span
      className={`rounded-full border px-2 py-0.5 text-[11px] ${
        className ?? "border-slate-700/80 bg-slate-900/60 text-slate-300"
      }`}
    >
      {children}
    </span>
  );
}
