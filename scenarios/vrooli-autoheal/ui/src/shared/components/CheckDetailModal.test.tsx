import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { CheckDetailModal } from "./CheckDetailModal";

const detailMocks = vi.hoisted(() => ({
  fetchCheckHistory: vi.fn(),
  fetchConfig: vi.fn(),
  fetchDefaults: vi.fn(),
  setCheckAutoHeal: vi.fn(),
  fetchCheckActions: vi.fn(),
  executeAction: vi.fn(),
  normalizeHealthStatus: (value: unknown) => (value === "critical" || value === "warning" || value === "ok" ? value : "ok"),
}));

vi.mock("../../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../../lib/api")>("../../lib/api");
  return { ...actual, ...detailMocks };
});
vi.mock("../contexts/CheckMetadataContext", () => ({
  useCheckMetadata: () => ({
    getTitle: () => "DNS Resolution",
    getMetadata: () => ({ title: "DNS Resolution", description: "Resolves names", importance: "Required", category: "resource", intervalSeconds: 3600 }),
  }),
}));
vi.mock("../../lib/export", () => ({ exportCheckHistoryToCSV: vi.fn() }));

describe("CheckDetailModal", () => {
  it("renders history details, auto-heal controls, exports, and tabs", async () => {
    detailMocks.fetchCheckHistory.mockResolvedValue({
      checkId: "infra-dns",
      count: 2,
      history: [
        { checkId: "infra-dns", status: "critical", message: "DNS failed", timestamp: new Date().toISOString(), duration: 1000, details: { subChecks: [{ name: "System", passed: true, detail: "ready" }, { name: "External", passed: false }] } },
        { checkId: "infra-dns", status: "ok", message: "Recovered", timestamp: new Date(Date.now() - 60000).toISOString(), duration: 2000 },
      ],
    });
    detailMocks.fetchConfig.mockResolvedValue({ checks: { "infra-dns": { enabled: true, autoHeal: true } } });
    detailMocks.fetchDefaults.mockResolvedValue({ checks: { "infra-dns": { enabled: true, autoHeal: false } } });
    detailMocks.fetchCheckActions.mockResolvedValue({ actions: [{ id: "restart", name: "Restart", description: "Restart check", available: true, dangerous: true }] });
    detailMocks.setCheckAutoHeal.mockResolvedValue({ success: true });
    detailMocks.executeAction.mockResolvedValue({ success: true, message: "Healed", output: "done" });
    const onClose = vi.fn();

    renderWithProviders(<CheckDetailModal checkId="infra-dns" onClose={onClose} />);
    expect(await screen.findByTestId("check-detail-modal")).toBeInTheDocument();
    expect(await screen.findByText("DNS failed")).toBeInTheDocument();
    expect(screen.getByText("System")).toBeInTheDocument();
    fireEvent.click(screen.getByTitle("Export history to CSV"));
    fireEvent.click(screen.getByRole("switch"));
    fireEvent.click(screen.getByRole("button", { name: "Heal Now" }));
    expect(screen.getByText("Confirm Action")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(screen.getByText("Healed")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Dismiss"));
    fireEvent.click(screen.getByRole("button", { name: /History \(2\)/i }));
    expect(screen.getByText("Recovered")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalled();
  });

  it("renders loading and history failure states", async () => {
    detailMocks.fetchCheckHistory.mockImplementation(() => new Promise(() => {}));
    renderWithProviders(<CheckDetailModal checkId="infra-dns" onClose={vi.fn()} />);
    expect(await screen.findByText("Loading history...")).toBeInTheDocument();

    detailMocks.fetchCheckHistory.mockRejectedValueOnce(new Error("history unavailable"));
    renderWithProviders(<CheckDetailModal checkId="infra-dns" onClose={vi.fn()} />);
    const retry = await screen.findByText(/retry/i);
    expect(retry).toBeInTheDocument();
    fireEvent.click(retry);
  });
});
