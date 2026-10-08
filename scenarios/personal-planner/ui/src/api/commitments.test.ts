import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ listCommitments: vi.fn(), createCommitment: vi.fn(), updateCommitmentState: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { createCommitment, fetchCommitments, updateCommitmentState } from "./commitments";
afterEach(() => vi.clearAllMocks());

describe("commitments API transport", () => {
  it("maps list, create, and revision-checked state changes", async () => {
    const commitment = { id: "c-1", revision: 1n } as never;
    client.listCommitments.mockResolvedValue({ commitments: [commitment] }); client.createCommitment.mockResolvedValue({ commitment }); client.updateCommitmentState.mockResolvedValue({ commitment });
    expect(await fetchCommitments()).toEqual([commitment]);
    expect(await createCommitment({ result: "Send the brief", promisedBoundary: "2026-10-01" })).toBe(commitment);
    expect(client.createCommitment).toHaveBeenCalledWith(expect.objectContaining({ result: "Send the brief", promisedBoundary: "2026-10-01", state: "proposed" }));
    expect(await updateCommitmentState(commitment, "active")).toBe(commitment);
    expect(client.updateCommitmentState).toHaveBeenCalledWith({ id: "c-1", state: "active", expectedRevision: 1n });
  });
});
