import { screen } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { renderWithProviders } from "../../test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { StandingGrantMetadata } from "./StandingGrantMetadata";
import { listStandingGrantMetadata } from "../../api/standingGrants";

vi.mock("../../api/standingGrants", () => ({ listStandingGrantMetadata: vi.fn() }));

describe("StandingGrantMetadata", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows owner-projected metadata and identifies the panel as read only", async () => {
    vi.mocked(listStandingGrantMetadata).mockResolvedValue({ grants: [{
      permissionId: "agent-manager/standing-grant/opaque", id: "grant-1", subject: "setup-worker", approver: "operator",
      operations: ["run.create"], targets: ["agent-manager:run/*"], state: "approved", maxConcurrentRuns: 2,
      maxRunBudgetUsd: 5, approvedAt: "2026-11-20T10:00:00Z", expiresAt: "2026-11-21T12:00:00Z",
    }], scopes: [{ permissionId: "agent-manager/standing-grant/scope", scope: "agent-manager:read", effect: "read", runEligible: true, command: "permissions/list" }] });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderWithProviders(<StandingGrantMetadata />, { queryClient: client });
    expect(await screen.findByText("setup-worker")).toBeInTheDocument();
    expect(screen.getByText(/Read only/)).toBeInTheDocument();
    expect(screen.getByText(/run.create/)).toBeInTheDocument();
    expect(screen.getByText("agent-manager:read")).toBeInTheDocument();
    expect(screen.getByText(/Declared Agent Manager scopes/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /revoke|approve/i })).not.toBeInTheDocument();
  });
});
