import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PhaseViewer } from "./phase-viewer";
import { contractPhaseView, simulationPhaseView } from "./phase-view";
import { selectors } from "../../../consts/selectors";
import { createQueryWrapper } from "../../../test-utils/query";
import type {
  OperatingModeCatalogPhase,
  OperatingModeRenderedPrompt,
  OperatingModeSimulationStep,
} from "../../../types/operating-mode";

vi.mock("../../../services/prompt-service", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    promptService: { getSkill: vi.fn() },
  };
});

vi.mock("../../../services", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    initiativeModeService: {
      renderSimulationPrompt: vi.fn(),
      renderLivePrompt: vi.fn(),
    },
  };
});

import { promptService } from "../../../services/prompt-service";
import { initiativeModeService } from "../../../services";

const getSkill = promptService.getSkill as unknown as ReturnType<typeof vi.fn>;
const renderSimulationPrompt = initiativeModeService.renderSimulationPrompt as unknown as ReturnType<typeof vi.fn>;

function contractPhase(overrides: Partial<OperatingModeCatalogPhase> = {}): OperatingModeCatalogPhase {
  return {
    phase: "review",
    phaseKind: "review",
    label: "Review",
    title: "Review",
    purpose: "",
    trigger: "",
    profileKey: "swarm-manager/analysis",
    writesRepo: false,
    catalogId: "cat",
    skillId: "swarm-manager-holistic-loop-review",
    activityPurpose: "",
    lockPurpose: "",
    reads: {
      base: ["OPERATOR_NOTE", "PRIOR_ROUNDS_JSON"],
      target: ["MEMBER_ITEMS_JSON", "ACCEPTANCE_CRITERIA"],
    },
    outputContract: {
      requiresStructuredResult: true,
      requiresProgress: false,
      requiresVerdict: true,
      requiresHandoff: false,
      requiresBacklogSync: false,
      requiredArtifactCount: 0,
    },
    ...overrides,
  };
}

function simulationStep(overrides: Partial<OperatingModeSimulationStep> = {}): OperatingModeSimulationStep {
  return {
    index: 0,
    phase: "investigate",
    phaseKind: "investigate",
    inputs: {
      target: { kind: "initiative", id: "sim", title: "Sim", context: { acceptance_criteria: ["Works end to end."] } },
      artifacts: [],
      priorRounds: [],
    },
    output: { handoff: { summary: "Investigated." } },
    round: {
      round: 1,
      mode: "holistic-loop",
      scopeKind: "initiative",
      scopeId: "sim",
      phase: "investigate",
      runStrategy: "single",
      agentProfileKey: "swarm-manager/deep-work",
      generatedAt: "",
      status: "completed",
    },
    transition: { from: "investigate", to: "plan", conditionKind: "always", label: "always" },
    terminal: false,
    skillId: "swarm-manager-holistic-loop-investigate",
    profileKey: "swarm-manager/deep-work",
    promptVariables: { INITIATIVE_TITLE: "Sim", MEMBER_ITEMS_JSON: "[]" },
    ...overrides,
  };
}

function rendered(overrides: Partial<OperatingModeRenderedPrompt> = {}): OperatingModeRenderedPrompt {
  return {
    mode: "holistic-loop",
    preset: "happy-path",
    stepIndex: 0,
    phase: "investigate",
    skillId: "swarm-manager-holistic-loop-investigate",
    profileKey: "swarm-manager/deep-work",
    variables: { INITIATIVE_TITLE: "Sim" },
    prompt: "You are investigating Sim. Items: execute/alpha.",
    degraded: false,
    ...overrides,
  };
}

describe("PhaseViewer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the contract source: template slots, variable-anchored reads, emit schema, declared transitions", async () => {
    getSkill.mockResolvedValue({
      id: "swarm-manager-holistic-loop-review",
      name: "Review",
      current_content: "Evaluate {{ACCEPTANCE_CRITERIA}} for {{INITIATIVE_TITLE}}.",
    });
    const view = contractPhaseView(contractPhase(), [
      { from: "review", to: "reconcile", conditionKind: "always", label: "always" },
    ]);
    render(<PhaseViewer view={view} />, { wrapper: createQueryWrapper() });

    // Instructions tab is the default and shows the unfilled template slots.
    await waitFor(() =>
      expect(screen.getByTestId(selectors.initiativeDetails.phaseViewerPrompt)).toHaveTextContent(
        "{{ACCEPTANCE_CRITERIA}}",
      ),
    );
    expect(screen.getByText(/Template with unfilled/)).toBeInTheDocument();
    const skillId = screen.getByTestId(selectors.initiativeDetails.phaseViewerSkillId);
    expect(skillId).toHaveTextContent("Skill ID");
    expect(skillId).toHaveTextContent("swarm-manager-holistic-loop-review");
    expect(skillId).toHaveAttribute("title", "prompt-manager skill read swarm-manager-holistic-loop-review");

    // Reads tab: rendered from the declared, composed contract — base group
    // and target-adapter group, one card per prompt variable.
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabReads));
    expect(screen.getByTestId("phase-reads-group-base")).toBeInTheDocument();
    expect(screen.getByTestId("phase-reads-group-target")).toBeInTheDocument();
    expect(screen.getByText("{{MEMBER_ITEMS_JSON}}")).toBeInTheDocument();
    expect(screen.getByText("{{ACCEPTANCE_CRITERIA}}")).toBeInTheDocument();
    expect(screen.getByText("{{OPERATOR_NOTE}}")).toBeInTheDocument();

    // Emits tab: verdict is a required emit for a review phase.
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabEmits));
    expect(screen.getByText("verdict")).toBeInTheDocument();
    expect(screen.getByText("required")).toBeInTheDocument();

    // Transition tab: the declared outgoing route.
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabTransition));
    expect(screen.getByText("if always, go to reconcile")).toBeInTheDocument();
  });

  it("derives the Reads tab from the phase's declared scope-specific groups (plan vs initiative)", async () => {
    getSkill.mockResolvedValue({ id: "s", name: "s", current_content: "x" });
    // An initiative-scope phase reads member items + acceptance criteria.
    const initiativeView = contractPhaseView(contractPhase());
    const { unmount } = render(<PhaseViewer view={initiativeView} />, { wrapper: createQueryWrapper() });
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabReads));
    expect(screen.getByText("{{MEMBER_ITEMS_JSON}}")).toBeInTheDocument();
    expect(screen.queryByText("{{PLAN_PATH}}")).not.toBeInTheDocument();
    unmount();

    // A plan-scope phase reads the plan path/context — a different, scope-
    // appropriate target group, not the fixed initiative vocabulary.
    const planPhase = contractPhase({
      phase: "execute",
      reads: { base: ["OPERATOR_NOTE", "PRIOR_HANDOFFS_JSON"], target: ["PLAN_PATH", "PLAN_CONTEXT_JSON"] },
    });
    const planView = contractPhaseView(planPhase);
    render(<PhaseViewer view={planView} />, { wrapper: createQueryWrapper() });
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabReads));
    expect(screen.getByText("{{PLAN_PATH}}")).toBeInTheDocument();
    expect(screen.getByText("{{PLAN_CONTEXT_JSON}}")).toBeInTheDocument();
    expect(screen.queryByText("{{MEMBER_ITEMS_JSON}}")).not.toBeInTheDocument();
    expect(screen.queryByText("{{ACCEPTANCE_CRITERIA}}")).not.toBeInTheDocument();
  });

  it("renders classification-on-transition as a built-in step on the contract Transition tab", async () => {
    getSkill.mockResolvedValue({ id: "s", name: "s", current_content: "x" });
    const drainPhase = contractPhase({
      phase: "execute",
      reads: { base: ["OPERATOR_NOTE"], target: ["PLAN_PATH"] },
      classification: { field: "progress", enum: ["continue", "complete", "blocked"], from: "handoff" },
    });
    const view = contractPhaseView(drainPhase, [
      { from: "execute", to: "execute", conditionKind: "eq", label: "on progress = continue", field: "progress", value: "continue", classified: true },
    ]);
    render(<PhaseViewer view={view} />, { wrapper: createQueryWrapper() });
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabTransition));
    expect(screen.getByTestId("phase-transition-classification-contract")).toBeInTheDocument();
    expect(screen.getByText(/Classification-on-transition/)).toBeInTheDocument();
    expect(screen.getByText("classified")).toBeInTheDocument();
  });

  it("renders the round's classification-on-transition outcome for a simulation round", async () => {
    renderSimulationPrompt.mockResolvedValue(rendered());
    const step = simulationStep({
      round: {
        round: 1,
        mode: "phased-plan-drain",
        scopeKind: "plan-manager-plan",
        scopeId: "sim",
        phase: "execute",
        runStrategy: "sequential_handoff",
        agentProfileKey: "swarm-manager/execution",
        generatedAt: "",
        status: "completed",
        transitionClassification: { outcome: "resolved", layer: "L1", classifiedField: "progress", classifiedValue: "continue" },
      },
    });
    const view = simulationPhaseView(step, "phased-plan-drain", "happy-path");
    render(<PhaseViewer view={view} />, { wrapper: createQueryWrapper() });
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabTransition));
    expect(screen.getByTestId("phase-transition-classification-outcome")).toBeInTheDocument();
    expect(screen.getByText(/via L1/)).toBeInTheDocument();
  });

  it("renders the simulation source: substituted prompt and declared reads with resolved values", async () => {
    renderSimulationPrompt.mockResolvedValue(rendered());
    const catalogPhase = contractPhase({ phase: "investigate", reads: { base: ["OPERATOR_NOTE"], target: ["MEMBER_ITEMS_JSON"] } });
    const view = simulationPhaseView(simulationStep(), "holistic-loop", "happy-path", catalogPhase);
    render(<PhaseViewer view={view} />, { wrapper: createQueryWrapper() });

    await waitFor(() =>
      expect(screen.getByTestId(selectors.initiativeDetails.phaseViewerPrompt)).toHaveTextContent(
        "You are investigating Sim",
      ),
    );
    expect(screen.getByText(/Filled with the selected preset/)).toBeInTheDocument();
    expect(renderSimulationPrompt).toHaveBeenCalledWith("holistic-loop", "happy-path", 0);

    // Reads tab shows the declared variable with its resolved simulation value.
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabReads));
    expect(screen.getByText("{{MEMBER_ITEMS_JSON}}")).toBeInTheDocument();
  });

  it("falls back to the resolved variables when the render endpoint degrades", async () => {
    renderSimulationPrompt.mockResolvedValue(
      rendered({ prompt: "", degraded: true, degradedReason: "seam unavailable", variables: { INITIATIVE_TITLE: "Sim" } }),
    );
    const view = simulationPhaseView(simulationStep(), "holistic-loop", "happy-path");
    render(<PhaseViewer view={view} />, { wrapper: createQueryWrapper() });

    await waitFor(() =>
      expect(screen.getByTestId(selectors.initiativeDetails.phaseViewerVariables)).toBeInTheDocument(),
    );
    expect(screen.queryByTestId(selectors.initiativeDetails.phaseViewerPrompt)).not.toBeInTheDocument();
    expect(screen.getByText("INITIATIVE_TITLE")).toBeInTheDocument();
  });

  it("switches tabs without refetching the prompt", async () => {
    renderSimulationPrompt.mockResolvedValue(rendered());
    const view = simulationPhaseView(simulationStep(), "holistic-loop", "happy-path");
    render(<PhaseViewer view={view} />, { wrapper: createQueryWrapper() });

    await waitFor(() => expect(renderSimulationPrompt).toHaveBeenCalledTimes(1));
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabEmits));
    expect(screen.getByText(/What this phase actually produced/)).toBeInTheDocument();
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabInstructions));
    // Still one render call — the query is cached across tab switches.
    expect(renderSimulationPrompt).toHaveBeenCalledTimes(1);
  });
});
