import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { OperatingModesTab } from "./OperatingModesTab";
import { selectors } from "../../../../consts/selectors";

const catalogMock = vi.fn();

vi.mock("../../../../services", () => ({
  initiativeModeService: {
    catalog: () => catalogMock(),
  },
}));

function renderTab(
  onItemClick = vi.fn(),
  options: { searchQuery?: string; onClearSearch?: () => void } = {},
) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return {
    onItemClick,
    ...render(
      <QueryClientProvider client={queryClient}>
        <OperatingModesTab
          searchQuery={options.searchQuery ?? ""}
          onItemClick={onItemClick}
          onClearSearch={options.onClearSearch}
        />
      </QueryClientProvider>,
    ),
  };
}

describe("OperatingModesTab", () => {
  beforeEach(() => {
    catalogMock.mockReset();
  });

  it("renders modes with usage counts", async () => {
    catalogMock.mockResolvedValue({
      modes: [
        {
          mode: "holistic-loop",
          label: "Holistic Loop",
          description: "Looped investigation cycles",
          usageCount: 4,
          scopeKind: "initiative",
          runStrategy: "operator_gated_loop",
          workspaceTabId: "operating-mode",
          capabilities: { supportsPhases: true, canStartPhases: true, canCompleteItems: false, canApplyBacklogSyncProposals: false, requiresAcceptanceCriteria: false, supportsArtifacts: true, supportsHandoffs: false, usesItemExecutionFlow: false },
          default: false,
          switchable: true,
          supportsPhases: true,
          phases: [],
        },
      ],
    });

    renderTab();

    await waitFor(() => {
      expect(screen.getByText("Holistic Loop")).toBeInTheDocument();
    });
    expect(screen.getByText(/4 init\./)).toBeInTheDocument();
    expect(screen.getByText("Looped investigation cycles")).toBeInTheDocument();
  });

  it("invokes onItemClick with the operatingMode prefix", async () => {
    catalogMock.mockResolvedValue({
      modes: [
        {
          mode: "phased-plan-drain",
          label: "Phased Plan Drain",
          usageCount: 0,
          scopeKind: "initiative",
          runStrategy: "sequential_handoff",
          workspaceTabId: "operating-mode",
          capabilities: { supportsPhases: true, canStartPhases: true, canCompleteItems: false, canApplyBacklogSyncProposals: false, requiresAcceptanceCriteria: false, supportsArtifacts: true, supportsHandoffs: false, usesItemExecutionFlow: false },
          default: false,
          switchable: true,
          supportsPhases: true,
          phases: [],
        },
      ],
    });

    const onItemClick = vi.fn();
    renderTab(onItemClick);

    const button = await screen.findByTestId("sidebar-operating-mode-item");
    fireEvent.click(button);

    expect(onItemClick).toHaveBeenCalledWith("operatingMode/phased-plan-drain");
  });

  it("shows an empty state when no modes are returned", async () => {
    catalogMock.mockResolvedValue({ modes: [] });
    renderTab();
    await waitFor(() => {
      expect(screen.getByText("No operating modes registered.")).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.sidebar.emptyState)).toBeInTheDocument();
    // Hint copy is rendered when no query is active.
    expect(
      screen.getByText("Modes appear here as the system learns new methodologies."),
    ).toBeInTheDocument();
  });

  it("renders a query-aware empty state with a working Clear search button", async () => {
    catalogMock.mockResolvedValue({
      modes: [
        {
          mode: "holistic-loop",
          label: "Holistic Loop",
          description: "Looped investigation cycles",
          usageCount: 1,
          scopeKind: "initiative",
          runStrategy: "operator_gated_loop",
          workspaceTabId: "operating-mode",
          capabilities: { supportsPhases: true, canStartPhases: true, canCompleteItems: false, canApplyBacklogSyncProposals: false, requiresAcceptanceCriteria: false, supportsArtifacts: true, supportsHandoffs: false, usesItemExecutionFlow: false },
          default: false,
          switchable: true,
          supportsPhases: true,
          phases: [],
        },
      ],
    });
    const onClearSearch = vi.fn();
    renderTab(vi.fn(), { searchQuery: "asdf", onClearSearch });
    await waitFor(() => {
      expect(screen.getByText('No matches for "asdf"')).toBeInTheDocument();
    });
    await userEvent.click(screen.getByTestId(selectors.sidebar.emptyStateClear));
    expect(onClearSearch).toHaveBeenCalledTimes(1);
  });
});
