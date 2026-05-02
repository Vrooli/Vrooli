import { describe, expect, it, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ModePickerDialog } from "./mode-picker-dialog";
import { ApiError } from "../../../lib/api-client";
import { selectors } from "../../../consts/selectors";
import { createQueryWrapper, createTestQueryClient } from "../../../test-utils/query";
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
    bestFor: [`${overrides.label} best for`],
    notFor: [`${overrides.label} not for`],
    tradeoffs: [`${overrides.label} tradeoff`],
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

  it("submits with cancel=false on the first attempt when leaving item-execution flow", async () => {
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
    // No upfront cancellation ack is shown — the server's 409 response is the
    // source of truth for when cancellation is required.
    expect(screen.queryByTestId(selectors.initiativeDetails.modePickerOverrideAck)).toBeNull();
    const confirm = screen.getByTestId(selectors.initiativeDetails.modePickerConfirm);
    expect(confirm).toBeEnabled();
    await userEvent.click(confirm);
    expect(onConfirm).toHaveBeenCalledWith("holistic-loop", false);
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

  it("surfaces a 409 active-item-executions conflict as a preview list with ack-then-confirm", async () => {
    const onConfirm = vi.fn();
    const conflictError = new ApiError("http", "active executions", {
      status: 409,
      code: "active_item_executions",
      details: {
        initiative_name: "initiative-a",
        from_mode: "item-level",
        to_mode: "holistic-loop",
        active_item_executions: [
          { item_ref: "fix:auth-cookie", run_id: "run-aaaa-bbbb", status: "running" },
          { item_ref: "feat:onboarding", run_id: "run-cccc-dddd", status: "running" },
        ],
      },
    });

    const wrapper = createQueryWrapper();
    const { rerender } = render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        onConfirm={onConfirm}
      />,
      { wrapper },
    );
    const cards = screen.getAllByTestId(selectors.initiativeDetails.modePickerCard);
    await userEvent.click(cards.find((c) => c.textContent?.includes("Holistic Loop"))!);
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.modePickerConfirm));
    expect(onConfirm).toHaveBeenLastCalledWith("holistic-loop", false);

    // Server returns 409. Parent forwards the error as `mutationError`.
    rerender(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        mutationError={conflictError}
        onConfirm={onConfirm}
      />,
    );

    const list = await screen.findByTestId(selectors.initiativeDetails.cancellationItemList);
    expect(within(list).getByText("fix:auth-cookie")).toBeInTheDocument();
    expect(within(list).getByText("feat:onboarding")).toBeInTheDocument();
    expect(within(list).getByText(/2 item executions currently running/i)).toBeInTheDocument();
    // Raw error message is suppressed in favour of the rendered preview.
    expect(screen.queryByText("active executions")).toBeNull();

    const confirm = screen.getByTestId(selectors.initiativeDetails.modePickerConfirm);
    expect(confirm).toBeDisabled();
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.modePickerOverrideAck));
    expect(confirm).toBeEnabled();
    expect(confirm).toHaveTextContent(/Cancel executions and switch/i);

    await userEvent.click(confirm);
    expect(onConfirm).toHaveBeenLastCalledWith("holistic-loop", true);
  });

  it("renders run-id links via the agent-manager external URL when resolved", async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(["embedded-service-url", "agent-manager"], "https://agent.test");

    const conflictError = new ApiError("http", "active executions", {
      status: 409,
      code: "active_item_executions",
      details: {
        initiative_name: "initiative-a",
        from_mode: "item-level",
        to_mode: "holistic-loop",
        active_item_executions: [{ item_ref: "fix:auth-cookie", run_id: "run-aaaa-bbbb", status: "running" }],
      },
    });

    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        mutationError={conflictError}
        onConfirm={() => {}}
      />,
      { wrapper: createQueryWrapper(queryClient) },
    );

    const list = await screen.findByTestId(selectors.initiativeDetails.cancellationItemList);
    const runLink = within(list).getByRole("link");
    expect(runLink).toHaveAttribute("href", "https://agent.test/runs/run-aaaa-bbbb");
    expect(runLink).toHaveAttribute("target", "_blank");
    expect(runLink).toHaveAttribute("rel", "noreferrer");
  });

  it("clears the conflict preview when the user picks a different target mode", async () => {
    const conflictError = new ApiError("http", "active executions", {
      status: 409,
      code: "active_item_executions",
      details: {
        initiative_name: "initiative-a",
        from_mode: "item-level",
        to_mode: "holistic-loop",
        active_item_executions: [{ item_ref: "fix:auth-cookie", run_id: "r" }],
      },
    });
    const wrapper = createQueryWrapper();
    const { rerender } = render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        mutationError={conflictError}
        onConfirm={() => {}}
      />,
      { wrapper },
    );
    expect(await screen.findByTestId(selectors.initiativeDetails.cancellationItemList)).toBeInTheDocument();

    rerender(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        // Conflict was tied to the previous selection; switching to a new
        // target mode discards the stale preview.
        mutationError={undefined}
        onConfirm={() => {}}
      />,
    );
    const cards = screen.getAllByTestId(selectors.initiativeDetails.modePickerCard);
    await userEvent.click(cards.find((c) => c.textContent?.includes("Phased Plan Drain"))!);
    expect(screen.queryByTestId(selectors.initiativeDetails.cancellationItemList)).toBeNull();
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

  it("does not render the catalog Retry button when there is no catalog error", () => {
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        onRetryCatalog={() => {}}
        isMutating={false}
        onConfirm={() => {}}
      />,
    );
    expect(screen.queryByTestId(selectors.initiativeDetails.modePickerRetry)).toBeNull();
  });

  it("renders the catalog Retry button on catalog error and calls onRetryCatalog when clicked", async () => {
    const onRetryCatalog = vi.fn();
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={[]}
        catalogLoading={false}
        catalogError={new Error("network down")}
        onRetryCatalog={onRetryCatalog}
        isMutating={false}
        onConfirm={() => {}}
      />,
    );
    expect(screen.getByText("network down")).toBeInTheDocument();
    const retry = screen.getByTestId(selectors.initiativeDetails.modePickerRetry);
    expect(retry).toBeEnabled();
    await userEvent.click(retry);
    expect(onRetryCatalog).toHaveBeenCalledTimes(1);
  });

  it("disables the Retry button and shows a spinner while the catalog is fetching", () => {
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={[]}
        catalogLoading={false}
        catalogError={new Error("network down")}
        catalogFetching
        onRetryCatalog={() => {}}
        isMutating={false}
        onConfirm={() => {}}
      />,
    );
    const retry = screen.getByTestId(selectors.initiativeDetails.modePickerRetry);
    expect(retry).toBeDisabled();
    expect(retry).toHaveTextContent(/Retrying…/);
  });

  it("renders the criteria pre-warning chip when target requires criteria and current does not", async () => {
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
    expect(
      screen.getByTestId(selectors.initiativeDetails.modePickerCriteriaPrewarning),
    ).toBeInTheDocument();
  });

  it("hides the criteria pre-warning when current already requires criteria", async () => {
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
    expect(
      screen.queryByTestId(selectors.initiativeDetails.modePickerCriteriaPrewarning),
    ).toBeNull();
  });

  it("renders the How to choose link when an opener is provided", async () => {
    const onOpenHowToChoose = vi.fn();
    render(
      <ModePickerDialog
        isOpen
        onClose={() => {}}
        currentMode="item-level"
        catalog={catalog}
        catalogLoading={false}
        isMutating={false}
        onConfirm={() => {}}
        onOpenHowToChoose={onOpenHowToChoose}
      />,
    );
    const link = screen.getByTestId(selectors.initiativeDetails.modePickerHowToChooseLink);
    await userEvent.click(link);
    expect(onOpenHowToChoose).toHaveBeenCalled();
  });

  it("hides the How to choose link when no opener is provided", () => {
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
    expect(
      screen.queryByTestId(selectors.initiativeDetails.modePickerHowToChooseLink),
    ).toBeNull();
  });

  it("renders the guidance callouts for the selected mode in a detail block below the card grid", async () => {
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
    // The detail block already renders for the initial selection (current
    // mode). Picking another card swaps the detail block to that mode.
    const cards = screen.getAllByTestId(selectors.initiativeDetails.modePickerCard);
    await userEvent.click(cards.find((c) => c.textContent?.includes("Holistic Loop"))!);
    const callouts = screen.getByTestId(selectors.initiativeDetails.modePickerGuidanceCallouts);
    expect(callouts).toBeInTheDocument();
    // Detail block sits *outside* every card so the 3-column callout grid can
    // breathe at the dialog's full width instead of being squeezed into a
    // single card column.
    for (const card of cards) {
      expect(card.contains(callouts)).toBe(false);
    }
  });

  it("renders the selected mode description and label in the detail block", async () => {
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
    expect(screen.getByText("About Holistic Loop")).toBeInTheDocument();
    // Description shows in the card *and* the detail block — both copies are
    // expected and useful (card stays self-explanatory, detail block reads
    // unclamped).
    expect(screen.getAllByText("Holistic Loop description").length).toBeGreaterThanOrEqual(1);
  });
});
