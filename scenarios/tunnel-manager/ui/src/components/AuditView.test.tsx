import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuditView } from "./AuditView";

// [REQ:UI-004] Port compliance audit view
const mockFetchAudit = vi.fn();

vi.mock("../lib/api", () => ({
  fetchAudit: (...args: unknown[]) => mockFetchAudit(...args),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

const auditEmpty = { results: [], total: 0, violations: 0, compliant: 0 };

const auditWithData = {
  results: [
    { subdomain: "api", scenario_name: "my-api", expected_port: 3000, actual_port: 3000, status: "compliant", detail: "" },
    { subdomain: "web", scenario_name: "my-web", expected_port: 8080, actual_port: 9090, status: "mismatch", detail: "Port mismatch: expected 8080 got 9090" },
    { subdomain: "docs", scenario_name: "my-docs", expected_port: 4000, actual_port: null, status: "missing", detail: "No process listening" },
  ],
  total: 3,
  violations: 2,
  compliant: 1,
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("AuditView", () => {
  it("renders the audit heading", () => {
    mockFetchAudit.mockResolvedValue(auditEmpty);
    render(<AuditView />, { wrapper });
    expect(screen.getByText("Port Compliance Audit")).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    mockFetchAudit.mockReturnValue(new Promise(() => {}));
    render(<AuditView />, { wrapper });
    expect(screen.getByText(/running audit/i)).toBeInTheDocument();
  });

  it("shows empty state when no routes exist", async () => {
    mockFetchAudit.mockResolvedValue(auditEmpty);
    render(<AuditView />, { wrapper });
    expect(await screen.findByText(/no routes to audit/i)).toBeInTheDocument();
  });

  it("shows summary counts with data", async () => {
    mockFetchAudit.mockResolvedValue(auditWithData);
    render(<AuditView />, { wrapper });
    expect(await screen.findByText("1 compliant")).toBeInTheDocument();
    expect(screen.getByText("2 violation(s)")).toBeInTheDocument();
    expect(screen.getByText("3 total")).toBeInTheDocument();
  });

  it("renders audit results with subdomains", async () => {
    mockFetchAudit.mockResolvedValue(auditWithData);
    render(<AuditView />, { wrapper });
    const apiItems = await screen.findAllByText("api");
    expect(apiItems.length).toBeGreaterThanOrEqual(1);
    const webItems = screen.getAllByText("web");
    expect(webItems.length).toBeGreaterThanOrEqual(1);
  });

  it("renders status badges for results", async () => {
    mockFetchAudit.mockResolvedValue(auditWithData);
    render(<AuditView />, { wrapper });
    const compliantBadges = await screen.findAllByText("compliant");
    expect(compliantBadges.length).toBeGreaterThanOrEqual(1);
    const mismatchBadges = screen.getAllByText("mismatch");
    expect(mismatchBadges.length).toBeGreaterThanOrEqual(1);
  });

  it("shows port detail for mismatch results", async () => {
    mockFetchAudit.mockResolvedValue(auditWithData);
    render(<AuditView />, { wrapper });
    const details = await screen.findAllByText(/port mismatch/i);
    expect(details.length).toBeGreaterThanOrEqual(1);
  });

  it("shows actual port as dash when missing", async () => {
    mockFetchAudit.mockResolvedValue(auditWithData);
    render(<AuditView />, { wrapper });
    const dashes = await screen.findAllByText("—");
    expect(dashes.length).toBeGreaterThanOrEqual(1);
  });

  it("shows error state when fetch fails", async () => {
    mockFetchAudit.mockRejectedValue(new Error("Network error"));
    render(<AuditView />, { wrapper });
    expect(await screen.findByText(/failed to run audit/i)).toBeInTheDocument();
  });

  it("has retry button on error", async () => {
    mockFetchAudit.mockRejectedValue(new Error("fail"));
    render(<AuditView />, { wrapper });
    expect(await screen.findByText("Retry")).toBeInTheDocument();
  });

  it("has refresh button with aria-label", () => {
    mockFetchAudit.mockResolvedValue(auditEmpty);
    render(<AuditView />, { wrapper });
    expect(screen.getByLabelText("Refresh audit")).toBeInTheDocument();
  });

  it("refetches when refresh button clicked", async () => {
    mockFetchAudit.mockResolvedValue(auditEmpty);
    render(<AuditView />, { wrapper });
    await screen.findByText(/no routes to audit/i);
    fireEvent.click(screen.getByLabelText("Refresh audit"));
    await waitFor(() => expect(mockFetchAudit).toHaveBeenCalledTimes(2));
  });

  it("renders data-testid attributes", async () => {
    mockFetchAudit.mockResolvedValue(auditWithData);
    render(<AuditView />, { wrapper });
    await screen.findByText("1 compliant");
    expect(screen.getByTestId("audit-panel")).toBeInTheDocument();
    expect(screen.getByTestId("audit-summary-compliant")).toBeInTheDocument();
    expect(screen.getByTestId("audit-summary-violations")).toBeInTheDocument();
  });
});
