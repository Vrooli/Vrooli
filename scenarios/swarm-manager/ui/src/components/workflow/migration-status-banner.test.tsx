import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { MigrationStatusBanner } from "./migration-status-banner";
import { agentOperationsService } from "../../services";
import type { WorkflowMigrationStatus } from "../../types/agent-operations";

vi.mock("../../services", () => ({
  agentOperationsService: {
    getMigrationStatus: vi.fn(),
  },
}));

const mockedStatus = vi.mocked(agentOperationsService.getMigrationStatus);

function status(overrides: Partial<WorkflowMigrationStatus>): WorkflowMigrationStatus {
  return {
    state: "not-started",
    epoch: 0,
    stagedCount: 0,
    promotedCount: 0,
    quarantinedCount: 0,
    startedAt: "",
    updatedAt: "",
    reportPath: "",
    documentFound: false,
    ...overrides,
  };
}

describe("MigrationStatusBanner", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders nothing in the not-started steady state", async () => {
    mockedStatus.mockResolvedValue(status({}));
    renderWithProviders(<MigrationStatusBanner />, { withRouter: false });
    await waitFor(() => expect(mockedStatus).toHaveBeenCalled());
    expect(screen.queryByTestId("workflow-migration-banner")).not.toBeInTheDocument();
  });

  it("renders nothing once promoted (steady state)", async () => {
    mockedStatus.mockResolvedValue(
      status({ state: "promoted", promotedCount: 10, documentFound: true, epoch: 1 }),
    );
    renderWithProviders(<MigrationStatusBanner />, { withRouter: false });
    await waitFor(() => expect(mockedStatus).toHaveBeenCalled());
    expect(screen.queryByTestId("workflow-migration-banner")).not.toBeInTheDocument();
  });

  it("renders the staged notice with counts", async () => {
    mockedStatus.mockResolvedValue(
      status({ state: "staged", stagedCount: 12, documentFound: true, epoch: 3 }),
    );
    renderWithProviders(<MigrationStatusBanner />, { withRouter: false });
    const banner = await screen.findByTestId("workflow-migration-banner");
    expect(banner).toHaveAttribute("data-state", "staged");
    expect(banner).toHaveTextContent("Workflow migration staged");
    expect(banner).toHaveTextContent("Epoch 3: 12 documents staged");
    // Informational partial-migration notice — polite live region.
    expect(banner).toHaveAttribute("role", "status");
  });

  it("renders the quarantined warning", async () => {
    mockedStatus.mockResolvedValue(
      status({ state: "quarantined", quarantinedCount: 1, documentFound: true, epoch: 2 }),
    );
    renderWithProviders(<MigrationStatusBanner />, { withRouter: false });
    const banner = await screen.findByTestId("workflow-migration-banner");
    expect(banner).toHaveAttribute("data-state", "quarantined");
    expect(banner).toHaveTextContent("Workflow migration quarantined");
    expect(banner).toHaveTextContent("Epoch 2: 1 document quarantined");
    // Quarantine means some items may run on legacy records — assertive alert.
    expect(banner).toHaveAttribute("role", "alert");
  });

  it("renders nothing while the status query is pending (no phantom banner)", () => {
    mockedStatus.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<MigrationStatusBanner />, { withRouter: false });
    expect(screen.queryByTestId("workflow-migration-banner")).not.toBeInTheDocument();
  });

  it("renders nothing when the status query fails (banner is advisory, never blocking)", async () => {
    mockedStatus.mockRejectedValue(new Error("migration status unavailable"));
    renderWithProviders(<MigrationStatusBanner />, { withRouter: false });
    await waitFor(() => expect(mockedStatus).toHaveBeenCalled());
    expect(screen.queryByTestId("workflow-migration-banner")).not.toBeInTheDocument();
  });
});
