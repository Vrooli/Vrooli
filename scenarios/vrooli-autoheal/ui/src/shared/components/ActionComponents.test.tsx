import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { ActionButtons } from "./ActionButtons";
import { ActionHistory, ActionHistoryCompact } from "./ActionHistory";

const actionMocks = vi.hoisted(() => ({
  fetchCheckActions: vi.fn(),
  executeAction: vi.fn(),
  fetchActionHistory: vi.fn(),
}));
vi.mock("../../lib/api", () => actionMocks);

describe("recovery action components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    actionMocks.fetchCheckActions.mockResolvedValue({
      actions: [
        { id: "start", name: "Start", description: "Start resource", available: true, dangerous: false },
        { id: "restart", name: "Restart", description: "Restart resource", available: true, dangerous: true },
        { id: "stop", name: "Stop", description: "Stop resource", available: false, dangerous: true },
        { id: "inspect", name: "Inspect", description: "Inspect resource", available: false, dangerous: false },
      ],
    });
    actionMocks.executeAction.mockResolvedValue({ success: true, actionId: "start", message: "Started" });
    actionMocks.fetchActionHistory.mockResolvedValue({
      logs: [
        { id: 1, actionId: "restart", checkId: "resource-postgres", success: true, message: "Restarted", durationMs: 250, timestamp: new Date().toISOString() },
        { id: 2, actionId: "stop", checkId: "resource-redis", success: false, message: "Failed", error: "permission denied", durationMs: 1200, timestamp: new Date(Date.now() - 86400000).toISOString() },
      ],
    });
  });

  it("renders action states, confirms dangerous actions, and reports results", async () => {
    renderWithProviders(<ActionButtons checkId="resource-postgres" category="resource" />);
    expect(await screen.findByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(screen.getByText("Stop")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start" }));
    await waitFor(() => expect(actionMocks.executeAction).toHaveBeenCalledWith("resource-postgres", "start"));

    fireEvent.click(screen.getByRole("button", { name: "Restart" }));
    expect(screen.getByText("Confirm Action")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "Restart" }));
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(actionMocks.executeAction).toHaveBeenCalledWith("resource-postgres", "restart"));
  });

  it("handles non-resource and empty action states", async () => {
    const { container } = renderWithProviders(<ActionButtons checkId="infra-dns" category="infrastructure" />);
    expect(container).toBeEmptyDOMElement();
    actionMocks.fetchCheckActions.mockResolvedValueOnce({ actions: [] });
    renderWithProviders(<ActionButtons checkId="resource-empty" category="resource" />);
    await waitFor(() => expect(actionMocks.fetchCheckActions).toHaveBeenCalledWith("resource-empty"));
  });

  it("renders history, compact history, empty, loading, and error states", async () => {
    renderWithProviders(<ActionHistory checkId="resource-postgres" limit={1} />);
    expect(await screen.findByText("Restarted")).toBeInTheDocument();
    expect(screen.getByText(/showing 1 of 2/i)).toBeInTheDocument();
    renderWithProviders(<ActionHistory checkId="resource-postgres" />);
    expect(await screen.findByText(/permission denied/i)).toBeInTheDocument();
    renderWithProviders(<ActionHistoryCompact checkId="resource-postgres" />);
    expect(await screen.findByText(/last action/i)).toBeInTheDocument();

    actionMocks.fetchActionHistory.mockResolvedValueOnce({ logs: [] });
    renderWithProviders(<ActionHistory checkId="empty" />);
    expect(await screen.findByText(/no actions have been executed/i)).toBeInTheDocument();
    actionMocks.fetchActionHistory.mockRejectedValueOnce(new Error("failed"));
    renderWithProviders(<ActionHistory checkId="error" />);
    expect(await screen.findByText(/failed to load action history/i)).toBeInTheDocument();
  });
});
