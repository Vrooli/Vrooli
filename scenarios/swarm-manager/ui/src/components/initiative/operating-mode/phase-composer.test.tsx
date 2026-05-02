import { describe, expect, it, vi } from "vitest";
import { useState } from "react";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PhaseComposer, type PhaseComposerItem } from "./phase-composer";
import type { PhaseQuickActionKey } from "./phase-composer-envelope";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCapabilities,
  OperatingModeCatalogEntry,
  OperatingModeRound,
  OperatingModeWorkspace,
} from "../../../types/operating-mode";

vi.mock("./phase-graph", () => ({
  PhaseGraph: ({
    selectedPhaseId,
    onSelectPhase,
    phaseStates,
  }: {
    selectedPhaseId?: string | null;
    onSelectPhase?: (phase: string) => void;
    phaseStates?: Record<string, { startable?: boolean; reason?: string; isNext?: boolean }>;
  }) => (
    <div data-testid="phase-graph-mock">
      <span data-testid="phase-graph-selected">{selectedPhaseId ?? ""}</span>
      {Object.entries(phaseStates ?? {}).map(([phase, state]) => (
        <button
          key={phase}
          type="button"
          data-testid={`phase-graph-node-${phase}`}
          data-startable={state.startable === true}
          onClick={() => {
            if (state.startable === false) return;
            onSelectPhase?.(phase);
          }}
        >
          {phase}
        </button>
      ))}
    </div>
  ),
}));

const initiativeModeCapabilities: OperatingModeCapabilities = {
  supportsPhases: true,
  canStartPhases: true,
  canCompleteItems: true,
  canApplyBacklogSyncProposals: true,
  requiresAcceptanceCriteria: true,
  supportsArtifacts: true,
  supportsHandoffs: false,
  usesItemExecutionFlow: false,
};

function makeCatalogEntry(): OperatingModeCatalogEntry {
  return {
    mode: "holistic-loop",
    label: "Holistic Loop",
    description: "Investigate → plan → execute → review.",
    usageCount: 1,
    scopeKind: "initiative",
    runStrategy: "operator_gated_loop",
    workspaceTabId: "operating-mode",
    capabilities: initiativeModeCapabilities,
    default: false,
    switchable: true,
    supportsPhases: true,
    phases: [],
  };
}

function makeWorkspace(overrides?: Partial<OperatingModeWorkspace>): OperatingModeWorkspace {
  return {
    initiativeName: "init-a",
    mode: "holistic-loop",
    definition: {
      mode: "holistic-loop",
      label: "Holistic Loop",
      scopeKind: "initiative",
      capabilities: initiativeModeCapabilities,
      runStrategy: "operator_gated_loop",
      terminal: ["review"],
      transitions: { investigate: ["plan"] },
      phases: [
        {
          phase: "investigate",
          activityPurpose: "holistic_loop_investigate",
          profileKey: "swarm-manager/deep-work",
          writesRepo: false,
          startable: true,
          next: true,
          requiresCriteria: false,
        },
        {
          phase: "execute",
          activityPurpose: "holistic_loop_execute",
          profileKey: "swarm-manager/deep-work",
          writesRepo: true,
          startable: false,
          reason: "Plan must complete first",
          requiresCriteria: false,
        },
      ],
    },
    artifacts: [],
    rounds: [],
    ...overrides,
  };
}

const items: PhaseComposerItem[] = [
  { ref: "execute/do-thing", title: "Do thing" },
  { ref: "fix/typo", title: "Fix typo" },
];

interface ComposerHarnessState {
  pendingPhase: string | null;
  selectedActions: Set<PhaseQuickActionKey>;
  selectedItems: Set<string>;
  pickerOpen: boolean;
  note: string;
}

function makeState(): ComposerHarnessState {
  return {
    pendingPhase: null,
    selectedActions: new Set(),
    selectedItems: new Set(),
    pickerOpen: false,
    note: "",
  };
}

function renderComposer(opts?: {
  state?: ComposerHarnessState;
  runningRound?: OperatingModeRound;
  phaseBusy?: boolean;
  canRunPhases?: boolean;
  startError?: unknown;
  onStart?: (phase: string, note: string) => void;
}) {
  const state = opts?.state ?? makeState();
  const onStart = opts?.onStart ?? vi.fn();
  function Harness() {
    const [s, setS] = useState(state);
    return (
      <PhaseComposer
        catalogEntry={makeCatalogEntry()}
        workspace={makeWorkspace()}
        runningRound={opts?.runningRound}
        items={items}
        pendingPhase={s.pendingPhase}
        onPendingPhaseChange={(p) => setS((prev) => ({ ...prev, pendingPhase: p }))}
        selectedActions={s.selectedActions}
        onSelectedActionsChange={(next) => setS((prev) => ({ ...prev, selectedActions: next }))}
        selectedItems={s.selectedItems}
        onSelectedItemsChange={(next) => setS((prev) => ({ ...prev, selectedItems: next }))}
        pickerOpen={s.pickerOpen}
        onPickerOpenChange={(open) => setS((prev) => ({ ...prev, pickerOpen: open }))}
        note={s.note}
        onNoteChange={(note) => setS((prev) => ({ ...prev, note }))}
        phaseBusy={opts?.phaseBusy ?? false}
        canRunPhases={opts?.canRunPhases ?? true}
        startError={opts?.startError}
        onStart={onStart}
      />
    );
  }
  return { ...render(<Harness />), onStart };
}

describe("PhaseComposer", () => {
  it("disables Start until a phase is selected", () => {
    renderComposer();
    expect(screen.getByTestId(selectors.initiativeDetails.phaseComposerStart)).toBeDisabled();
  });

  it("enables Start once a startable phase is selected and a note is typed", async () => {
    const onStart = vi.fn();
    renderComposer({ onStart });
    await userEvent.click(screen.getByTestId("phase-graph-node-investigate"));
    await userEvent.type(
      screen.getByTestId(selectors.initiativeDetails.phaseComposerNote),
      "deep dive",
    );
    const startBtn = screen.getByTestId(selectors.initiativeDetails.phaseComposerStart);
    expect(startBtn).toBeEnabled();
    await userEvent.click(startBtn);
    expect(onStart).toHaveBeenCalledWith("investigate", "deep dive");
  });

  it("does not select a phase when clicking a disabled node", async () => {
    renderComposer();
    await userEvent.click(screen.getByTestId("phase-graph-node-execute"));
    expect(screen.getByTestId("phase-graph-selected")).toHaveTextContent("");
  });

  it("makes tighten_scope and expand_scope mutually exclusive", async () => {
    renderComposer();
    await userEvent.click(screen.getByTestId("phase-graph-node-investigate"));
    const tighten = screen.getByTestId(selectors.initiativeDetails.phaseComposerActionTighten);
    const expand = screen.getByTestId(selectors.initiativeDetails.phaseComposerActionExpand);
    await userEvent.click(tighten);
    expect(tighten).toHaveAttribute("aria-pressed", "true");
    await userEvent.click(expand);
    expect(tighten).toHaveAttribute("aria-pressed", "false");
    expect(expand).toHaveAttribute("aria-pressed", "true");
  });

  it("opens the item picker when focus_on_items is selected", async () => {
    renderComposer();
    await userEvent.click(screen.getByTestId("phase-graph-node-investigate"));
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseComposerActionFocus));
    expect(screen.getByTestId(selectors.initiativeDetails.phaseComposerItemPicker)).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.initiativeDetails.phaseComposerItemPickerItem)).toHaveLength(2);
  });

  it("disables skip_unblock when the selected phase is startable", async () => {
    renderComposer();
    await userEvent.click(screen.getByTestId("phase-graph-node-investigate"));
    expect(screen.getByTestId(selectors.initiativeDetails.phaseComposerActionSkip)).toBeDisabled();
  });

  it("blocks Start while a round is running", async () => {
    const runningRound: OperatingModeRound = {
      round: 5,
      mode: "holistic-loop",
      scopeKind: "initiative",
      scopeId: "init-a",
      phase: "investigate",
      runStrategy: "operator_gated_loop",
      agentProfileKey: "swarm-manager/deep-work",
      generatedAt: "2026-04-30T00:00:00Z",
      runId: "run-1",
      status: "agent_running",
    };
    renderComposer({ runningRound });
    expect(
      screen.getByTestId(selectors.initiativeDetails.phaseComposerActiveBanner),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByTestId("phase-graph-node-investigate"));
    await userEvent.type(
      screen.getByTestId(selectors.initiativeDetails.phaseComposerNote),
      "go",
    );
    expect(screen.getByTestId(selectors.initiativeDetails.phaseComposerStart)).toBeDisabled();
  });

  it("shows the selected-phase metadata strip with profile + write-mode chip", async () => {
    renderComposer();
    await userEvent.click(screen.getByTestId("phase-graph-node-investigate"));
    const strip = screen.getByTestId(selectors.initiativeDetails.phaseComposerSelectedPhaseStrip);
    const utils = within(strip);
    expect(utils.getByText(/Profile: swarm-manager\/deep-work/)).toBeInTheDocument();
    expect(utils.getByText("read-only")).toBeInTheDocument();
  });

  it("renders the start error message", () => {
    renderComposer({ startError: new Error("nope") });
    expect(screen.getByText("nope")).toBeInTheDocument();
  });
});
