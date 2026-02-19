import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import RecoveryPage from "./RecoveryPage";

// [REQ:RECOVER-006] [REQ:RECOVER-007] Recovery page tests

const mockFetchRecoveryState = vi.fn();
const mockFetchRecoveryEvents = vi.fn();
const mockTriggerRecovery = vi.fn();
const mockResetCircuit = vi.fn();

vi.mock("../lib/api", () => ({
  fetchRecoveryState: (...args: unknown[]) => mockFetchRecoveryState(...args),
  fetchRecoveryEvents: (...args: unknown[]) => mockFetchRecoveryEvents(...args),
  triggerRecovery: (...args: unknown[]) => mockTriggerRecovery(...args),
  resetCircuit: (...args: unknown[]) => mockResetCircuit(...args),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

const idleState = {
  status: "idle",
  circuit_open: false,
  consecutive_failures: 0,
  backoff_retries: 0,
};

const circuitOpenState = {
  status: "circuit_open",
  circuit_open: true,
  consecutive_failures: 5,
  backoff_retries: 3,
  last_recovery_at: "2026-02-19T08:00:00Z",
};

const recoveryEvents = [
  { id: 1, trigger_type: "ready_failure", action: "systemctl_restart", outcome: "success", details: "recovered", created_at: "2026-02-19T07:00:00Z" },
  { id: 2, trigger_type: "ha_connection_loss", action: "systemctl_restart", outcome: "failure", details: "timeout", created_at: "2026-02-19T07:30:00Z" },
  { id: 3, trigger_type: "manual", action: "systemctl_restart", outcome: "skipped", details: "circuit open", created_at: "2026-02-19T08:00:00Z" },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockFetchRecoveryEvents.mockResolvedValue([]);
});

describe("RecoveryPage", () => {
  it("renders recovery state heading", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("Recovery State")).toBeDefined();
  });

  it("renders recovery events heading", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("Recovery Events")).toBeDefined();
  });

  it("shows idle status", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("idle")).toBeDefined();
  });

  it("shows circuit breaker closed status", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("CLOSED")).toBeDefined();
  });

  it("shows circuit breaker open status", async () => {
    mockFetchRecoveryState.mockResolvedValue(circuitOpenState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("OPEN")).toBeDefined();
  });

  it("shows consecutive failures count", async () => {
    mockFetchRecoveryState.mockResolvedValue(circuitOpenState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("5")).toBeDefined();
  });

  it("shows trigger recovery button", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("Trigger Recovery")).toBeDefined();
  });

  it("shows reset circuit button when circuit is open", async () => {
    mockFetchRecoveryState.mockResolvedValue(circuitOpenState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("Reset Circuit")).toBeDefined();
  });

  it("hides reset circuit button when circuit is closed", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    render(<RecoveryPage />, { wrapper });
    await screen.findByText("Trigger Recovery");
    expect(screen.queryByText("Reset Circuit")).toBeNull();
  });

  it("shows recovery state error", async () => {
    mockFetchRecoveryState.mockRejectedValue(new Error("fail"));
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("Failed to load recovery state.")).toBeDefined();
  });

  it("shows last recovery timestamp", async () => {
    mockFetchRecoveryState.mockResolvedValue(circuitOpenState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText(/Last recovery:/)).toBeDefined();
  });

  it("shows no recovery events message", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    mockFetchRecoveryEvents.mockResolvedValue([]);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("No recovery events")).toBeDefined();
  });

  it("shows recovery events with trigger labels", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    mockFetchRecoveryEvents.mockResolvedValue(recoveryEvents);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("Ready Failure")).toBeDefined();
    expect(screen.getByText("HA Connection Loss")).toBeDefined();
    expect(screen.getByText("Manual")).toBeDefined();
  });

  it("shows recovery event outcome badges", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    mockFetchRecoveryEvents.mockResolvedValue(recoveryEvents);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("success")).toBeDefined();
    expect(screen.getByText("failure")).toBeDefined();
    expect(screen.getByText("skipped")).toBeDefined();
  });

  it("has refresh buttons for both panels", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    mockFetchRecoveryEvents.mockResolvedValue([]);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByLabelText("Refresh recovery state")).toBeDefined();
    expect(screen.getByLabelText("Refresh recovery events")).toBeDefined();
  });

  it("shows backoff retries count", async () => {
    mockFetchRecoveryState.mockResolvedValue(circuitOpenState);
    render(<RecoveryPage />, { wrapper });
    expect(await screen.findByText("3")).toBeDefined();
  });

  it("shows confirmation dialog when trigger recovery is clicked", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    render(<RecoveryPage />, { wrapper });
    const triggerBtn = await screen.findByText("Trigger Recovery");
    fireEvent.click(triggerBtn);
    expect(await screen.findByTestId("confirm-dialog")).toBeDefined();
    expect(screen.getByText(/restart the cloudflared tunnel/)).toBeDefined();
  });

  it("shows confirmation dialog when reset circuit is clicked", async () => {
    mockFetchRecoveryState.mockResolvedValue(circuitOpenState);
    render(<RecoveryPage />, { wrapper });
    const resetBtn = await screen.findByText("Reset Circuit");
    fireEvent.click(resetBtn);
    expect(await screen.findByText("Reset Circuit Breaker")).toBeDefined();
    expect(screen.getByText(/close the circuit breaker/)).toBeDefined();
  });

  it("calls triggerRecovery when confirmed", async () => {
    mockFetchRecoveryState.mockResolvedValue(idleState);
    mockTriggerRecovery.mockResolvedValue({});
    render(<RecoveryPage />, { wrapper });
    const triggerBtn = await screen.findByText("Trigger Recovery");
    fireEvent.click(triggerBtn);
    const confirmBtn = await screen.findByTestId("confirm-dialog-confirm");
    fireEvent.click(confirmBtn);
    await waitFor(() => expect(mockTriggerRecovery).toHaveBeenCalledWith(false));
  });
});
