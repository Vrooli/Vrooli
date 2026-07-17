import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DecisionFlow } from "./decision-flow";
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

describe("DecisionFlow", () => {
  it("traverses 'no' → 'yes' (items not coupled, items stable) and lands on the member-item strategy", async () => {
    render(<DecisionFlow catalog={FULL_CATALOG} />);
    await userEvent.click(screen.getByRole("button", { name: /^No$/i }));
    await userEvent.click(screen.getByRole("button", { name: /^Yes$/i }));
    expect(screen.getByText("Recommended")).toBeInTheDocument();
    // Presented as the strategy; the underlying wire value stays item-level.
    expect(screen.getByText("Member-item workflow")).toBeInTheDocument();
  });

  it("traverses 'yes' → 'no' (items coupled, plan unstable) and lands on holistic-loop", async () => {
    render(<DecisionFlow catalog={FULL_CATALOG} />);
    await userEvent.click(screen.getByRole("button", { name: /^Yes$/i }));
    await userEvent.click(screen.getByRole("button", { name: /^No$/i }));
    expect(screen.getByText("Recommended")).toBeInTheDocument();
    expect(screen.getByText("Holistic Loop")).toBeInTheDocument();
  });

  it("traverses 'yes' → 'yes' (items coupled, plan stable) and lands on phased-plan-drain", async () => {
    render(<DecisionFlow catalog={FULL_CATALOG} />);
    await userEvent.click(screen.getByRole("button", { name: /^Yes$/i }));
    await userEvent.click(screen.getByRole("button", { name: /^Yes$/i }));
    expect(screen.getByText("Recommended")).toBeInTheDocument();
    expect(screen.getByText("Phased Plan Drain")).toBeInTheDocument();
  });

  it("calls onAccept when the operator picks the recommendation", async () => {
    const onAccept = vi.fn();
    render(<DecisionFlow catalog={FULL_CATALOG} onAccept={onAccept} />);
    await userEvent.click(screen.getByRole("button", { name: /^No$/i }));
    await userEvent.click(screen.getByRole("button", { name: /^Yes$/i }));
    await userEvent.click(screen.getByRole("button", { name: /Pick this mode/i }));
    expect(onAccept).toHaveBeenCalledWith("item-level");
  });

  it("renders an error chip when the config references a mode not in the catalog", () => {
    // Catalog missing `phased-plan-drain` → the config still references it, so
    // a visible error chip surfaces. Failure is loud, not silent.
    const partialCatalog = FULL_CATALOG.filter((entry) => entry.mode !== "phased-plan-drain");
    render(<DecisionFlow catalog={partialCatalog} />);
    expect(screen.getByText(/Decision flow references unknown mode/i)).toBeInTheDocument();
    expect(screen.getByText("phased-plan-drain")).toBeInTheDocument();
  });
});
