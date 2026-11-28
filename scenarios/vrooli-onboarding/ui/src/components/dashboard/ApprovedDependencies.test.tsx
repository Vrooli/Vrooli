import { screen } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { renderWithProviders } from "../../test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApprovedDependencies } from "./ApprovedDependencies";
import { listApprovedDependencies } from "../../api/approvedDependencies";

vi.mock("../../api/approvedDependencies", () => ({ listApprovedDependencies: vi.fn() }));

describe("ApprovedDependencies", () => {
  beforeEach(() => vi.mocked(listApprovedDependencies).mockResolvedValue({
    principal: "operator-1", source: ".vrooli/dependencies/approved-dependencies.json", source_revision: "abc123",
    source_digest: "sha256:0123456789abcdef", schema_version: "1", version: "2026-11-20", mode: "reviewed",
    guidance: "Approval changes remain in dependency review.",
    records: [{ ecosystem: "npm", package_name: "example", version_range: "^1.2.0", state: "approved", rationale: "Used by the app", approved_by: "maintainer" }],
  }));

  it("shows source-bound read-only approvals without approval controls", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderWithProviders(<ApprovedDependencies />, { queryClient: client });
    expect(await screen.findByText("example")).toBeInTheDocument();
    expect(screen.getByText("Approval changes remain in dependency review.")).toBeInTheDocument();
    expect(screen.getByText(/reviewed/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /approve|edit/i })).not.toBeInTheDocument();
  });
});
