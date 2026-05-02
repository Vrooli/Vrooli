import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ModePickerDialog } from "./mode-picker-dialog";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCapabilities,
  OperatingModeCatalogEntry,
} from "../../../types/operating-mode";

const itemExecutionCapabilities: OperatingModeCapabilities = {
  supportsPhases: false,
  canStartPhases: false,
  canCompleteItems: false,
  canApplyBacklogSyncProposals: false,
  requiresAcceptanceCriteria: false,
  supportsArtifacts: false,
  supportsHandoffs: false,
  usesItemExecutionFlow: true,
};

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

function makeMode(overrides: Partial<OperatingModeCatalogEntry> & { mode: string; label: string }): OperatingModeCatalogEntry {
  return {
    description: `${overrides.label} description`,
    usageCount: 1,
    scopeKind: "initiative",
    runStrategy: "operator_gated_loop",
    workspaceTabId: "operating-mode",
    capabilities: initiativeModeCapabilities,
    default: false,
    switchable: true,
    supportsPhases: true,
    phases: [],
    ...overrides,
  };
}

const catalog: OperatingModeCatalogEntry[] = [
  makeMode({
    mode: "item-level",
    label: "Item Level",
    capabilities: itemExecutionCapabilities,
    scopeKind: "backlog_item",
    runStrategy: "existing_item_flow",
    default: true,
    supportsPhases: false,
  }),
  makeMode({
    mode: "holistic-loop",
    label: "Holistic Loop",
    capabilities: initiativeModeCapabilities,
  }),
  makeMode({
    mode: "phased-plan-drain",
    label: "Phased Plan Drain",
    capabilities: { ...initiativeModeCapabilities, supportsHandoffs: true },
    runStrategy: "sequential_handoff",
  }),
];

describe("ModePickerDialog", () => {
  it("does not render its content when closed", () => {
    render(
      <ModePickerDialog
        isOpen={false}
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        onConfirm={() => {}}
      />,
    );
    expect(screen.queryByTestId(selectors.initiativeDetails.modePicker)).toBeNull();
  });

  it("renders a card per switchable mode and pre-selects the current one", () => {
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="holistic-loop"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        onConfirm={() => {}}
      />,
    );
    const cards = screen.getAllByTestId(selectors.initiativeDetails.modePickerCard);
    expect(cards).toHaveLength(3);
    const holisticCard = cards.find((card) => card.textContent?.includes("Holistic Loop"));
    expect(holisticCard).toHaveAttribute("aria-pressed", "true");
  });

  it("disables Switch button when the current mode is selected", () => {
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="holistic-loop"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        onConfirm={() => {}}
      />,
    );
    expect(screen.getByTestId(selectors.initiativeDetails.modePickerConfirm)).toBeDisabled();
  });

  it("enables Switch and confirms with cancelAck=false for non-item-execution exits", async () => {
    const onConfirm = vi.fn();
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="holistic-loop"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        onConfirm={onConfirm}
      />,
    );
    const cards = screen.getAllByTestId(selectors.initiativeDetails.modePickerCard);
    await userEvent.click(cards.find((c) => c.textContent?.includes("Phased Plan Drain"))!);
    const confirm = screen.getByTestId(selectors.initiativeDetails.modePickerConfirm);
    expect(confirm).toBeEnabled();
    await userEvent.click(confirm);
    expect(onConfirm).toHaveBeenCalledWith("phased-plan-drain", false);
  });

  it("requires the override checkbox when leaving item-execution flow", async () => {
    const onConfirm = vi.fn();
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        onConfirm={onConfirm}
      />,
    );
    const cards = screen.getAllByTestId(selectors.initiativeDetails.modePickerCard);
    await userEvent.click(cards.find((c) => c.textContent?.includes("Holistic Loop"))!);
    const confirm = screen.getByTestId(selectors.initiativeDetails.modePickerConfirm);
    expect(confirm).toBeDisabled();
    const ack = screen.getByTestId(selectors.initiativeDetails.modePickerOverrideAck);
    await userEvent.click(ack);
    expect(confirm).toBeEnabled();
    await userEvent.click(confirm);
    expect(onConfirm).toHaveBeenCalledWith("holistic-loop", true);
  });

  it("does not show the amber notice when switching between two non-item-execution modes", async () => {
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="holistic-loop"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        onConfirm={() => {}}
      />,
    );
    const cards = screen.getAllByTestId(selectors.initiativeDetails.modePickerCard);
    await userEvent.click(cards.find((c) => c.textContent?.includes("Phased Plan Drain"))!);
    expect(screen.queryByTestId(selectors.initiativeDetails.modePickerOverrideAck)).toBeNull();
  });

  it("renders the compare panel with capability deltas for the selected mode", async () => {
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        onConfirm={() => {}}
      />,
    );
    const cards = screen.getAllByTestId(selectors.initiativeDetails.modePickerCard);
    await userEvent.click(cards.find((c) => c.textContent?.includes("Holistic Loop"))!);
    const compare = screen.getByTestId(selectors.initiativeDetails.modePickerComparePanel);
    expect(compare).toHaveTextContent(/Adds phase graph/i);
    expect(compare).toHaveTextContent(/Removes existing item execution flow/i);
  });

  it("surfaces mutation errors", () => {
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        mutationError={new Error("boom")}
        onConfirm={() => {}}
      />,
    );
    expect(screen.getByText("boom")).toBeInTheDocument();
  });

  it("disables Cancel and Switch while mutating", () => {
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="holistic-loop"
        catalog={catalog}
        catalogLoading={false}
        isMutating
        onConfirm={() => {}}
      />,
    );
    expect(screen.getByTestId(selectors.initiativeDetails.modePickerCancel)).toBeDisabled();
    expect(screen.getByTestId(selectors.initiativeDetails.modePickerConfirm)).toBeDisabled();
    expect(screen.getByText("Switching…")).toBeInTheDocument();
  });
});
