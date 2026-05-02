import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { OperatingModeCard } from "./operating-mode-card";
import type {
  OperatingModeCapabilities,
  OperatingModeCatalogEntry,
} from "../../../types/operating-mode";

const baseCapabilities: OperatingModeCapabilities = {
  supportsPhases: false,
  canStartPhases: false,
  canCompleteItems: false,
  canApplyBacklogSyncProposals: false,
  requiresAcceptanceCriteria: false,
  supportsArtifacts: false,
  supportsHandoffs: false,
  usesItemExecutionFlow: true,
};

function makeMode(overrides: Partial<OperatingModeCatalogEntry> = {}): OperatingModeCatalogEntry {
  return {
    mode: "item-level",
    label: "Item Level",
    description: "Each backlog item runs through the existing flow.",
    bestFor: ["Right-sized items"],
    notFor: ["Coupled work"],
    tradeoffs: ["Highest parallelism"],
    usageCount: 3,
    scopeKind: "backlog_item",
    runStrategy: "existing_item_flow",
    workspaceTabId: "info",
    capabilities: baseCapabilities,
    default: true,
    switchable: true,
    supportsPhases: false,
    phases: [],
    ...overrides,
  };
}

describe("OperatingModeCard", () => {
  it("renders label, usage badge, description, and scope/strategy line", () => {
    render(<OperatingModeCard mode={makeMode()} data-testid="card" />);
    expect(screen.getByText("Item Level")).toBeInTheDocument();
    expect(screen.getByText("3 init.")).toBeInTheDocument();
    expect(screen.getByText("Each backlog item runs through the existing flow.")).toBeInTheDocument();
    expect(screen.getByText(/Backlog item · Existing item flow · default/)).toBeInTheDocument();
  });

  it("omits the description block when none is set", () => {
    render(<OperatingModeCard mode={makeMode({ description: undefined })} data-testid="card" />);
    expect(screen.queryByText(/Each backlog item/)).not.toBeInTheDocument();
  });

  it("omits the default suffix when not the default mode", () => {
    render(<OperatingModeCard mode={makeMode({ default: false })} data-testid="card" />);
    expect(screen.queryByText(/· default/)).not.toBeInTheDocument();
  });

  it("renders as a button when onClick is provided", async () => {
    const onClick = vi.fn();
    render(<OperatingModeCard mode={makeMode()} onClick={onClick} data-testid="card" />);
    const card = screen.getByTestId("card");
    expect(card.tagName).toBe("BUTTON");
    await userEvent.click(card);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("renders as a non-interactive container without onClick", () => {
    render(<OperatingModeCard mode={makeMode()} data-testid="card" />);
    const card = screen.getByTestId("card");
    expect(card.tagName).not.toBe("BUTTON");
  });

  it("applies the selected ring class when selected", () => {
    const { container } = render(
      <OperatingModeCard mode={makeMode()} selected onClick={() => {}} data-testid="card" />,
    );
    const inner = container.querySelector(".ring-2");
    expect(inner).not.toBeNull();
  });

  it("aria-pressed reflects selected state on the interactive variant", () => {
    render(<OperatingModeCard mode={makeMode()} selected onClick={() => {}} data-testid="card" />);
    expect(screen.getByTestId("card")).toHaveAttribute("aria-pressed", "true");
  });

  it("uses single-line description clamp in compact mode", () => {
    const { container } = render(
      <OperatingModeCard mode={makeMode()} compact data-testid="card" />,
    );
    expect(container.querySelector(".line-clamp-1")).not.toBeNull();
  });

  it("uses two-line description clamp by default", () => {
    const { container } = render(<OperatingModeCard mode={makeMode()} data-testid="card" />);
    expect(container.querySelector("p.line-clamp-2.text-xs")).not.toBeNull();
  });

  it("keeps the uniform compact shape when selected (decision-support detail renders below the grid)", () => {
    const { container } = render(
      <OperatingModeCard mode={makeMode()} selected onClick={() => {}} data-testid="card" />,
    );
    // Selected cards stay tight so they don't push the row to a tall narrow
    // column. Callouts and full description live in the picker's detail
    // block below the grid, not inside the card itself.
    expect(container.querySelector("p.line-clamp-2.text-xs")).not.toBeNull();
    expect(
      screen.queryByTestId("initiative-mode-picker-guidance-callouts"),
    ).toBeNull();
  });
});
