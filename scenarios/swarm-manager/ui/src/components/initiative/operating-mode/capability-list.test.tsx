import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { CapabilityList } from "./capability-list";
import type { OperatingModeCapabilities } from "../../../types/operating-mode";

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

describe("CapabilityList", () => {
  it("renders enabled capabilities in compact variant", () => {
    render(
      <CapabilityList
        capabilities={caps({ supportsPhases: true, canStartPhases: true, supportsHandoffs: true })}
      />,
    );
    expect(screen.getByText("Phase graph")).toBeInTheDocument();
    expect(screen.getByText("Phase start controls")).toBeInTheDocument();
    expect(screen.getByText("Round handoffs")).toBeInTheDocument();
    expect(screen.queryByText("Phase artifacts")).not.toBeInTheDocument();
  });

  it("renders enabled capabilities in full variant", () => {
    render(
      <CapabilityList
        capabilities={caps({ supportsPhases: true, supportsArtifacts: true })}
        variant="full"
      />,
    );
    expect(screen.getByText("Phase graph")).toBeInTheDocument();
    expect(screen.getByText("Phase artifacts")).toBeInTheDocument();
  });

  it("renders the empty message when no capabilities are enabled", () => {
    render(<CapabilityList capabilities={caps()} emptyMessage="Nothing here" />);
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
  });

  it("uses capabilityLabel for translation (no raw camelCase keys leak through)", () => {
    render(
      <CapabilityList capabilities={caps({ canApplyBacklogSyncProposals: true })} variant="full" />,
    );
    expect(screen.getByText("Apply backlog sync proposals")).toBeInTheDocument();
    expect(screen.queryByText("canApplyBacklogSyncProposals")).not.toBeInTheDocument();
  });
});
