import { QueryClient } from "@tanstack/react-query";
import { renderWithProviders } from "../../test-utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CredentialGrantActions } from "./CredentialGrantActions";
import { listCredentialGrantActions, writeCredentialGrantAction } from "../../api/credentialGrants";

vi.mock("../../api/credentialGrants", () => ({
  listCredentialGrantActions: vi.fn(),
  writeCredentialGrantAction: vi.fn(),
}));

const createAction = {
  permissionId: "bridge/credential-grant-action/create/YQBiADc",
  principalId: "operator-a",
  operation: "create" as const,
  subjectId: "YQBiADc",
  expectedRevision: "0",
  target: { machineId: "machine-a", nodeId: "node-a", enrollmentGeneration: 7 },
};
const revokeAction = {
  permissionId: "bridge/credential-grant-action/revoke/Z3JhbnQtMQ",
  principalId: "operator-a",
  operation: "revoke" as const,
  subjectId: "grant-1",
  expectedRevision: "3",
  target: { machineId: "machine-a", nodeId: "node-a", enrollmentGeneration: 7 },
};

describe("CredentialGrantActions", () => {
  beforeEach(() => {
    vi.mocked(listCredentialGrantActions).mockResolvedValue([createAction]);
    vi.mocked(writeCredentialGrantAction).mockResolvedValue({
      id: "receipt-1", permissionId: createAction.permissionId, owner: "vrooli-bridge", actorId: "operator-a",
      target: createAction.target, expectedRevision: "0", resultRevision: "1", idempotencyKey: "request-1",
      outcome: "created", ownerState: "pending",
    });
  });
  afterEach(() => { vi.clearAllMocks(); });

  it("keeps action controls separate and requires explicit confirmation; retry reuses request ID", async () => {
    const query = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderWithProviders(<CredentialGrantActions />, { queryClient: query });
    expect(await screen.findByTestId("credential-grant-action-row")).toBeTruthy();
    expect(screen.getByText(/machine-a.*node-a.*7/)).toBeTruthy();

    fireEvent.change(screen.getByLabelText("Logical credential ID"), { target: { value: "service/api" } });
    fireEvent.change(screen.getByLabelText("Field name"), { target: { value: "token" } });
    fireEvent.click(screen.getByRole("button", { name: "Review operation" }));
    expect(await screen.findByTestId("credential-grant-action-review")).toBeTruthy();
    expect(writeCredentialGrantAction).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(writeCredentialGrantAction).toHaveBeenCalledTimes(1));
    expect(await screen.findByText(/Target effect state: pending\. Mutation outcome: created\./)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry exact operation" }));
    await waitFor(() => expect(writeCredentialGrantAction).toHaveBeenCalledTimes(2));
    const calls = vi.mocked(writeCredentialGrantAction).mock.calls;
    expect(calls[0]?.[1]).toBe(calls[1]?.[1]);
    expect(calls[0]?.[2]).toEqual(calls[1]?.[2]);
  });

  it("reuses the same request ID after an ambiguous owner response", async () => {
    vi.mocked(writeCredentialGrantAction).mockRejectedValueOnce(new Error("owner response lost"));
    const query = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderWithProviders(<CredentialGrantActions />, { queryClient: query });
    await screen.findByTestId("credential-grant-action-row");
    fireEvent.change(screen.getByLabelText("Logical credential ID"), { target: { value: "service/api" } });
    fireEvent.change(screen.getByLabelText("Field name"), { target: { value: "token" } });
    fireEvent.click(screen.getByRole("button", { name: "Review operation" }));
    fireEvent.click(await screen.findByRole("button", { name: "Confirm" }));
    expect(await screen.findByText(/You can retry this exact operation safely/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry exact operation" }));
    await waitFor(() => expect(writeCredentialGrantAction).toHaveBeenCalledTimes(2));
    const calls = vi.mocked(writeCredentialGrantAction).mock.calls;
    expect(calls[0]?.[1]).toBe(calls[1]?.[1]);
    expect(calls[0]?.[2]).toEqual(calls[1]?.[2]);
  });

  it("reviews and submits a separate write-only revoke action", async () => {
    vi.mocked(listCredentialGrantActions).mockResolvedValue([revokeAction]);
    vi.mocked(writeCredentialGrantAction).mockResolvedValue({
      id: "receipt-revoke", permissionId: revokeAction.permissionId, owner: "vrooli-bridge", actorId: "operator-a",
      target: revokeAction.target, expectedRevision: "3", resultRevision: "4", idempotencyKey: "revoke-key",
      outcome: "revoked", ownerState: "refused",
    });
    const query = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderWithProviders(<CredentialGrantActions />, { queryClient: query });
    const row = await screen.findByTestId("credential-grant-action-row");
    expect(row.textContent).toContain("Revoke grant grant-1");
    fireEvent.click(screen.getByRole("button", { name: "Review operation" }));
    fireEvent.click(await screen.findByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(writeCredentialGrantAction).toHaveBeenCalledTimes(1));
    expect(vi.mocked(writeCredentialGrantAction).mock.calls[0]?.[0]).toEqual(revokeAction);
    expect(vi.mocked(writeCredentialGrantAction).mock.calls[0]?.[2]).toEqual({ operation: "revoke" });
    expect(await screen.findByText(/Target effect state: refused\. Mutation outcome: revoked\./)).toBeTruthy();
  });
});
