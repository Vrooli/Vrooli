import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { ModeMatrix } from "./mode-matrix";
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

describe("ModeMatrix", () => {
  it("renders one column per catalog mode and rows for each aspect", () => {
    const catalog = [
      makeMode({ mode: "item-level", label: "Item Level", targetKind: "initiative", default: true }),
      makeMode({ mode: "holistic-loop", label: "Holistic Loop", capabilities: caps({ supportsPhases: true }) }),
      makeMode({ mode: "phased-plan-drain", label: "Phased Plan Drain", capabilities: caps({ supportsHandoffs: true }) }),
    ];
    render(<ModeMatrix catalog={catalog} />);
    // The legacy item-level entry presents as the member-item workflow
    // strategy (explicit label mapping, lib/member-item-strategy.ts).
    expect(screen.getByText("Member-item workflow")).toBeInTheDocument();
    expect(screen.getByText("Holistic Loop")).toBeInTheDocument();
    expect(screen.getByText("Phased Plan Drain")).toBeInTheDocument();
    // Required rows
    expect(screen.getByRole("rowheader", { name: "Target" })).toBeInTheDocument();
    expect(screen.getByRole("rowheader", { name: "Run strategy" })).toBeInTheDocument();
    expect(screen.getByRole("rowheader", { name: "Best for" })).toBeInTheDocument();
    expect(screen.getByRole("rowheader", { name: "Not for" })).toBeInTheDocument();
    expect(screen.getByRole("rowheader", { name: "Tradeoffs" })).toBeInTheDocument();
    expect(screen.getByRole("rowheader", { name: "Capabilities" })).toBeInTheDocument();
    // Decision metadata is reflected per column
    expect(screen.getByText("Item Level best for")).toBeInTheDocument();
    expect(screen.getByText("Phased Plan Drain tradeoff")).toBeInTheDocument();
  });

  it("renders a hint when the catalog is empty", () => {
    render(<ModeMatrix catalog={[]} />);
    expect(screen.getByText(/No modes are registered/i)).toBeInTheDocument();
  });
});
