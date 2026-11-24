import { QueryClient } from "@tanstack/react-query";
import { renderWithProviders } from "../../test-utils";
import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SealedCredentialGrants } from "./SealedCredentialGrants";
import { listCredentialGrantMetadata } from "../../api/credentialGrants";

vi.mock("../../api/credentialGrants", () => ({ listCredentialGrantMetadata: vi.fn() }));

describe("SealedCredentialGrants", () => {
  beforeEach(() => vi.mocked(listCredentialGrantMetadata).mockResolvedValue([{
    permissionId: "bridge/credential-grant/grant-1", id: "grant-1", logicalId: "service/api", field: "token",
    grantClass: "user_prompt", retention: "ephemeral", generation: 3, ackedGeneration: 3, status: "delivered",
    grantedAt: "", revokedAt: "", receiptAt: "2026-11-20T08:00:00Z", receiptAccepted: true, purgeState: "none",
    purgeReceiptAt: "", purgeAccepted: false, machineId: "machine-1", nodeId: "node-1", enrollmentGeneration: 4,
  }]));
  afterEach(() => cleanup());

  it("shows target-bound status without exposing credential values or mutation actions", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderWithProviders(<SealedCredentialGrants />, { queryClient: client });
    expect(await screen.findByText("service/api · token")).toBeTruthy();
    expect(screen.getByText("delivered")).toBeTruthy();
    expect(screen.getByText(/machine-1.*node-1.*4/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: /create|revoke/i })).toBeNull();
    expect(screen.queryByText(/secret_value/i)).toBeNull();
  });
});
