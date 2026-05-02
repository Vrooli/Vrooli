import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PhaseCard } from "./phase-card";
import { selectors } from "../../../consts/selectors";
import { createQueryWrapper, createTestQueryClient } from "../../../test-utils/query";
import type { OperatingModeCatalogPhase } from "../../../types/operating-mode";

function basePhase(overrides: Partial<OperatingModeCatalogPhase> & { phase: string }): OperatingModeCatalogPhase {
  return {
    label: overrides.phase,
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
      requiredArtifactCount: 0,
    },
    ...overrides,
  };
}

describe("PhaseCard", () => {
  it("renders the label as the headline, with the snake_case ID and purpose", () => {
    render(
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
    render(
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
    render(
<PhaseCard
        phase={basePhase({ phase: "investigate", isStart: true })}
      />,
    );
    expect(screen.getByText("start")).toBeInTheDocument();
    expect(screen.queryByText("terminal")).not.toBeInTheDocument();

    render(
<PhaseCard
        phase={basePhase({ phase: "review", isTerminal: true })}
      />,
    );
    expect(screen.getByText("terminal")).toBeInTheDocument();
  });

  it("renders only the contract chips that are set", () => {
    render(
<PhaseCard
        phase={basePhase({
          phase: "review",
          requiresCriteria: true,
          outputContract: {
            requiresStructuredResult: true,
            requiresProgress: false,
            requiresVerdict: true,
            requiresHandoff: false,
            requiredArtifactCount: 0,
          },
        })}
      />,
    );
    expect(screen.getByText("verdict")).toBeInTheDocument();
    expect(screen.getByText("structured")).toBeInTheDocument();
    expect(screen.getByText("requires criteria")).toBeInTheDocument();
    expect(screen.queryByText("handoff")).not.toBeInTheDocument();
    expect(screen.queryByText("progress")).not.toBeInTheDocument();
  });

  it("flags required artifacts and lists optional ones", () => {
    render(
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
    render(<PhaseCard phase={basePhase({ phase: "investigate", writesRepo: false })} />);
    expect(screen.getByText("read-only")).toBeInTheDocument();
  });

  it("applies the highlight ring when highlighted is true", () => {
    const { container } = render(
      <PhaseCard phase={basePhase({ phase: "execute" })} highlighted />,
    );
    const article = container.querySelector('[data-testid="phase-card"]');
    expect(article?.className).toMatch(/ring-cyan-500/);
  });

  it("opens the profile popover when the profile chip is clicked", async () => {
    render(<PhaseCard phase={basePhase({ phase: "investigate" })} />, {
      wrapper: createQueryWrapper(),
    });
    const chip = screen.getByTestId(selectors.initiativeDetails.phaseCardProfileChip);
    expect(chip.tagName).toBe("BUTTON");
    expect(screen.queryByTestId(selectors.initiativeDetails.phaseProfilePopover)).not.toBeInTheDocument();

    await userEvent.click(chip);

    const popover = screen.getByTestId(selectors.initiativeDetails.phaseProfilePopover);
    expect(popover).toBeInTheDocument();
    expect(popover).toHaveAttribute("role", "dialog");
    expect(
      screen.getByText(/defines the model, tool access, and runtime budget/i),
    ).toBeInTheDocument();
    // The literal profile key is present inside the popover's <code> block.
    expect(popover.querySelector("code")?.textContent).toBe("swarm-manager/deep-work");
  });

  it("renders an external Agent Manager link in the popover when the agent-manager URL resolves", async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(["embedded-service-url", "agent-manager"], "https://agent.test");
    render(<PhaseCard phase={basePhase({ phase: "investigate" })} />, {
      wrapper: createQueryWrapper(queryClient),
    });
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseCardProfileChip));
    const link = screen.getByTestId(selectors.initiativeDetails.phaseProfileExternalLink);
    expect(link).toHaveAttribute(
      "href",
      "https://agent.test/profiles?profileKey=swarm-manager%2Fdeep-work",
    );
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noreferrer");
  });

  it("hides the popover external link when the agent-manager URL has not resolved", async () => {
    render(<PhaseCard phase={basePhase({ phase: "investigate" })} />, {
      wrapper: createQueryWrapper(),
    });
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.phaseCardProfileChip));
    expect(screen.queryByTestId(selectors.initiativeDetails.phaseProfileExternalLink)).toBeNull();
  });
});
