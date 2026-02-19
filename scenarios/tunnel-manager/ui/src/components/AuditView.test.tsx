import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuditView } from "./AuditView";

// [REQ:UI-004] Port compliance audit view
vi.mock("../lib/api", () => ({
  fetchAudit: vi.fn().mockResolvedValue({ results: [], total: 0, violations: 0, compliant: 0 }),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("AuditView", () => {
  it("renders the audit heading", () => {
    render(<AuditView />, { wrapper });
    expect(screen.getByText("Port Compliance Audit")).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    render(<AuditView />, { wrapper });
    expect(screen.getByText(/running audit/i)).toBeInTheDocument();
  });
});
