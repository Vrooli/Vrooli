import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PhaseCard } from "./phase-card";
import { selectors } from "../../../consts/selectors";
import { createQueryWrapper, createTestQueryClient } from "../../../test-utils/query";
import type { OperatingModeCatalogEntry, OperatingModeCatalogPhase } from "../../../types/operating-mode";

vi.mock("../../../services/prompt-service", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    promptService: { getSkill: vi.fn() },
  };
});

import { promptService } from "../../../services/prompt-service";

const getSkill = promptService.getSkill as unknown as ReturnType<typeof vi.fn>;

function basePhase(overrides: Partial<OperatingModeCatalogPhase> & { phase: string }): OperatingModeCatalogPhase {
  return {
    label: overrides.phase,
    phaseKind: "investigate",
    title: overrides.phase,
    purpose: "",
    trigger: "",
    profileKey: "swarm-manager/deep-work",
    writesRepo: false,
    catalogId: "",
    skillId: "",
    activityPurpose: "",
    lockPurpose: "",
    outputContract: {
      requiresStructuredResult: true,
      requiresProgress: false,
      requiresVerdict: false,
      requiresHandoff: false,
      requiresBacklogSync: false,
      requiredArtifactCount: 0,
    },
    ...overrides,
  };
}

function renderCard(ui: React.ReactElement, queryClient = createTestQueryClient()) {
  return render(ui, { wrapper: createQueryWrapper(queryClient) });
}

describe("PhaseCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getSkill.mockResolvedValue({ id: "skill", name: "Skill", current_content: "template body" });
  });

  it("renders the label as the headline, with the snake_case ID and purpose", () => {
    renderCard(
      <PhaseCard
        phase={basePhase({
          phase: "investigate",
          label: "Investigate",
          title: "Holistic Loop Investigate",
          purpose: "Investigate the initiative.",
        })}
      />,
    );
    expect(screen.getByRole("heading", { name: "Investigate" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Holistic Loop Investigate" })).not.toBeInTheDocument();
    expect(screen.getByText("investigate")).toBeInTheDocument();
    expect(screen.getByText("Investigate the initiative.")).toBeInTheDocument();
  });

  it("falls back to title when label is empty (legacy API payload)", () => {
    renderCard(
      <PhaseCard
        phase={basePhase({
          phase: "investigate",
          label: "",
          title: "Holistic Loop Investigate",
        })}
      />,
    );
    expect(screen.getByRole("heading", { name: "Holistic Loop Investigate" })).toBeInTheDocument();
  });

  it("renders start and terminal markers", () => {
    renderCard(<PhaseCard phase={basePhase({ phase: "investigate", isStart: true })} />);
    expect(screen.getByText("start")).toBeInTheDocument();

    renderCard(<PhaseCard phase={basePhase({ phase: "review", isTerminal: true })} />);
    expect(screen.getAllByText("terminal").length).toBeGreaterThan(0);
  });

  it("composes the shared PhaseViewer in contract source with the four concern tabs", async () => {
    renderCard(
      <PhaseCard
        phase={basePhase({
          phase: "review",
          requiresCriteria: true,
          reads: { base: ["OPERATOR_NOTE"], target: ["MEMBER_ITEMS_JSON", "ACCEPTANCE_CRITERIA"] },
          outputContract: {
            requiresStructuredResult: true,
            requiresProgress: false,
            requiresVerdict: true,
            requiresHandoff: false,
            requiresBacklogSync: false,
            requiredArtifactCount: 0,
          },
        })}
        transitions={[{ from: "review", to: "reconcile", conditionKind: "always", label: "always" }]}
      />,
    );
    const viewer = screen.getByTestId(selectors.initiativeDetails.phaseViewer);
    expect(viewer).toHaveAttribute("data-source", "contract");
    expect(screen.getByText("requires criteria")).toBeInTheDocument();

    // Reads tab: each card names its backing prompt variable, from data.
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabReads));
    expect(screen.getByText("{{MEMBER_ITEMS_JSON}}")).toBeInTheDocument();
    expect(screen.getByText("{{ACCEPTANCE_CRITERIA}}")).toBeInTheDocument();

    // Emits tab: verdict is a required emit for a review phase.
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabEmits));
    expect(screen.getByText("verdict")).toBeInTheDocument();
    expect(screen.getByText("required")).toBeInTheDocument();

    // Transition tab: the declared outgoing route.
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabTransition));
    expect(screen.getByText("if always, go to reconcile")).toBeInTheDocument();
  });

  it("shows terminal transition copy when no outgoing transition exists", async () => {
    renderCard(<PhaseCard phase={basePhase({ phase: "reconcile", isTerminal: true })} />);
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerTabTransition));
    expect(screen.getByText("No outgoing transition; this phase is terminal.")).toBeInTheDocument();
  });

  it("renders the contract template body in the Instructions tab", async () => {
    getSkill.mockResolvedValue({
      id: "swarm-manager-holistic-loop-investigate",
      name: "Investigate",
      current_content: "Investigate {{INITIATIVE_TITLE}} thoroughly.",
    });
    renderCard(
      <PhaseCard phase={basePhase({ phase: "investigate", skillId: "swarm-manager-holistic-loop-investigate" })} />,
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.initiativeDetails.phaseViewerPrompt)).toHaveTextContent(
        "{{INITIATIVE_TITLE}}",
      ),
    );
  });

  it("flags required artifacts and lists optional ones", () => {
    renderCard(
      <PhaseCard
        phase={basePhase({
          phase: "investigate",
          outputArtifacts: [
            { path: "modes/holistic-loop/findings.md", contentType: "text/markdown", required: true },
            { path: "modes/holistic-loop/notes.md", contentType: "text/markdown", required: false },
          ],
        })}
      />,
    );
    expect(screen.getByText("modes/holistic-loop/findings.md")).toBeInTheDocument();
    expect(screen.getByText("modes/holistic-loop/notes.md")).toBeInTheDocument();
    expect(screen.getByText("required")).toBeInTheDocument();
  });

  it("renders the read-only chip when writes_repo is false", () => {
    renderCard(<PhaseCard phase={basePhase({ phase: "investigate", writesRepo: false })} />);
    expect(screen.getByText("read-only")).toBeInTheDocument();
  });

  it("applies the highlight ring when highlighted is true", () => {
    const { container } = renderCard(<PhaseCard phase={basePhase({ phase: "execute" })} highlighted />);
    const article = container.querySelector('[data-testid="phase-card"]');
    expect(article?.className).toMatch(/ring-cyan-500/);
  });

  it("opens the profile popover from the Instructions tab profile chip", async () => {
    renderCard(<PhaseCard phase={basePhase({ phase: "investigate" })} />);
    const chip = screen.getByTestId(selectors.initiativeDetails.phaseViewerProfileChip);
    expect(chip.tagName).toBe("BUTTON");
    expect(screen.queryByTestId(selectors.initiativeDetails.phaseProfilePopover)).not.toBeInTheDocument();

    await userEvent.click(chip);

    const popover = screen.getByTestId(selectors.initiativeDetails.phaseProfilePopover);
    expect(popover).toHaveAttribute("role", "dialog");
    expect(popover.querySelector("code")?.textContent).toBe("swarm-manager/deep-work");
  });

  it("renders an external Agent Manager link in the popover when the agent-manager URL resolves", async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(["embedded-service-url", "agent-manager"], "https://agent.test");
    renderCard(<PhaseCard phase={basePhase({ phase: "investigate" })} />, queryClient);
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseViewerProfileChip));
    const link = screen.getByTestId(selectors.initiativeDetails.phaseProfileExternalLink);
    expect(link).toHaveAttribute("href", "https://agent.test/profiles?profileKey=swarm-manager%2Fdeep-work");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders a delegated phase's composed sub-mode graph inline", () => {
    const subMode: OperatingModeCatalogEntry = {
      mode: "phased-plan-drain",
      label: "Phased Plan Drain",
      bestFor: [],
      notFor: [],
      tradeoffs: [],
      usageCount: 0,
      targetKind: "plan-execution",
      runStrategy: "sequential_handoff",
      workspaceTabId: "",
      capabilities: {
        supportsPhases: true,
        canStartPhases: true,
        canCompleteItems: false,
        canApplyBacklogSyncProposals: false,
        requiresAcceptanceCriteria: false,
        supportsArtifacts: true,
        supportsHandoffs: true,
      },
      default: false,
      switchable: true,
      supportsPhases: true,
      phases: [
        basePhase({
          phase: "execute",
          isStart: true,
          classification: { field: "progress", enum: ["continue", "complete", "blocked"], from: "handoff" },
        }),
      ],
    };
    renderCard(
      <PhaseCard
        phase={basePhase({ phase: "execute", executedBy: "phased-plan-drain" })}
        subModes={{ "phased-plan-drain": subMode }}
      />,
    );
    // The delegated marker and the composed sub-mode's phases render inline.
    expect(screen.getByText("delegated")).toBeInTheDocument();
    const composed = screen.getByTestId("composed-sub-mode-graph");
    expect(composed).toHaveAttribute("data-sub-mode", "phased-plan-drain");
    expect(screen.getByTestId("composed-phase-execute")).toBeInTheDocument();
    expect(within(composed).getByText(/target: Plan execution/)).toBeInTheDocument();
    expect(within(composed).getByText(/classifies/)).toBeInTheDocument();
  });
});
