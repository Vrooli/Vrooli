import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PhaseCard } from "./phase-card";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeCatalogPhase } from "../../../types/operating-mode";

function basePhase(overrides: Partial<OperatingModeCatalogPhase> & { phase: string }): OperatingModeCatalogPhase {
  return {
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
  it("renders the title, snake_case ID, and purpose", () => {
    render(
      <PhaseCard
        phase={basePhase({
          phase: "investigate",
          title: "Holistic Loop Investigate",
          purpose: "Investigate the initiative.",
        })}
      />,
    );
    expect(screen.getByText("Holistic Loop Investigate")).toBeInTheDocument();
    expect(screen.getByText("investigate")).toBeInTheDocument();
    expect(screen.getByText("Investigate the initiative.")).toBeInTheDocument();
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

  it("exposes a profile-info disclosure with the descriptive copy", async () => {
    render(<PhaseCard phase={basePhase({ phase: "investigate" })} />);
    const details = screen.getByTestId(selectors.initiativeDetails.phaseCardProfileInfo);
    expect(details.tagName).toBe("DETAILS");
    expect(details).not.toHaveAttribute("open");
    const summary = details.querySelector("summary");
    expect(summary).not.toBeNull();
    await userEvent.click(summary!);
    expect(details).toHaveAttribute("open");
    expect(
      screen.getByText(/Different profiles vary the model, tool access, and runtime budget/),
    ).toBeInTheDocument();
  });
});
