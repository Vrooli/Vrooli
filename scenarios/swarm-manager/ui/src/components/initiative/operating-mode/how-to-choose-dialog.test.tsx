import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { HowToChooseDialog } from "./how-to-choose-dialog";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCapabilities,
  OperatingModeCatalogEntry,
} from "../../../types/operating-mode";

function caps(overrides: Partial<OperatingModeCapabilities> = {}): OperatingModeCapabilities {
  return {
    supportsPhases: false,
    canStartPhases: false,
    canCompleteItems: false,
    canApplyBacklogSyncProposals: false,
    requiresAcceptanceCriteria: false,
    supportsArtifacts: false,
    supportsHandoffs: false,
    usesItemExecutionFlow: false,
    ...overrides,
  };
}

function makeMode(
  overrides: Partial<OperatingModeCatalogEntry> & { mode: string; label: string },
): OperatingModeCatalogEntry {
  return {
    description: `${overrides.label} description`,
    bestFor: [`${overrides.label} best for`],
    notFor: [`${overrides.label} not for`],
    tradeoffs: [`${overrides.label} tradeoff`],
    usageCount: 0,
    targetKind: "initiative",
    runStrategy: "operator_gated_loop",
    workspaceTabId: "operating-mode",
    capabilities: caps(),
    default: false,
    switchable: true,
    supportsPhases: true,
    phases: [],
    ...overrides,
  };
}

const FULL_CATALOG: OperatingModeCatalogEntry[] = [
  makeMode({ mode: "item-level", label: "Item Level", default: true }),
  makeMode({ mode: "holistic-loop", label: "Holistic Loop" }),
  makeMode({ mode: "phased-plan-drain", label: "Phased Plan Drain" }),
];

describe("HowToChooseDialog", () => {
  it("does not render when closed", () => {
    render(
      <HowToChooseDialog isOpen={false} onClose={() => {}} catalog={FULL_CATALOG} />,
    );
    expect(screen.queryByTestId(selectors.initiativeDetails.howToChooseDialog)).toBeNull();
  });

  it("renders both the decision flow and the matrix when open", () => {
    render(<HowToChooseDialog isOpen onClose={() => {}} catalog={FULL_CATALOG} />);
    expect(screen.getByTestId(selectors.initiativeDetails.howToChooseDialog)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.initiativeDetails.howToChooseDecisionFlow)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.initiativeDetails.howToChooseMatrix)).toBeInTheDocument();
  });

  it("calls onPickRecommendation and onClose when a recommendation is accepted", async () => {
    const onPickRecommendation = vi.fn();
    const onClose = vi.fn();
    render(
      <HowToChooseDialog
        isOpen
        onClose={onClose}
        catalog={FULL_CATALOG}
        onPickRecommendation={onPickRecommendation}
      />,
    );
    // No → Yes lands on item-level
    await userEvent.click(screen.getByRole("button", { name: /^No$/i }));
    await userEvent.click(screen.getByRole("button", { name: /^Yes$/i }));
    await userEvent.click(screen.getByRole("button", { name: /Pick this mode/i }));
    expect(onPickRecommendation).toHaveBeenCalledWith("item-level");
    expect(onClose).toHaveBeenCalled();
  });
});
