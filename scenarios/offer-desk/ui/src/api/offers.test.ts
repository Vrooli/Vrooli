import { beforeEach, describe, expect, it, vi } from "vitest";
import { NodeKind, Status, TriggerComposition } from "@vrooli/proto-types/offer-desk/v1/offers/offers_pb";

const client = vi.hoisted(() => ({
  getBoard: vi.fn(),
  listNodes: vi.fn(),
  listEdges: vi.fn(),
  listProposals: vi.fn(),
  evaluate: vi.fn(),
  createNode: vi.fn(),
  transition: vi.fn(),
  createEdge: vi.fn(),
  declareTrigger: vi.fn(),
  addFact: vi.fn(),
  promote: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => client),
}));

vi.mock("./client", () => ({ transport: {} }));

import {
  addFact,
  createEdge,
  createNode,
  declareTrigger,
  evaluateTriggers,
  fetchBoard,
  fetchEdges,
  fetchNodes,
  fetchProposals,
  promoteNode,
  transition,
} from "./offers";

describe("offer Connect wrappers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    for (const method of Object.values(client)) method.mockResolvedValue({ ok: true });
  });

  it("builds typed requests for reads and evaluation", async () => {
    expect(await fetchBoard()).toEqual({ ok: true });
    expect(await fetchNodes()).toEqual({ ok: true });
    expect(await fetchEdges()).toEqual({ ok: true });
    expect(await fetchProposals()).toEqual({ ok: true });
    expect(await evaluateTriggers()).toEqual({ ok: true });
    expect(await evaluateTriggers(false)).toEqual({ ok: true });
    expect(client.getBoard).toHaveBeenCalledOnce();
    expect(client.listNodes).toHaveBeenCalledOnce();
    expect(client.listEdges).toHaveBeenCalledOnce();
    expect(client.listProposals).toHaveBeenCalledOnce();
    expect(client.evaluate).toHaveBeenCalledTimes(2);
  });

  it("builds typed requests for catalog, gate, and operator actions", async () => {
    expect(await createNode({ kind: NodeKind.OFFER, name: "Consulting", status: Status.IDEA })).toEqual({ ok: true });
    expect(await createNode({ kind: NodeKind.CHANNEL, name: "Referral", status: Status.CANDIDATE, triggerId: "trigger-1", actualAccountId: "account-1" })).toEqual({ ok: true });
    expect(await transition({ nodeId: "node-1", status: Status.ACTIVE, actor: "operator" })).toEqual({ ok: true });
    expect(await createEdge({ fromId: "node-1", toId: "node-2", kind: "feeds" })).toEqual({ ok: true });
    expect(await createEdge({ fromId: "node-1", toId: "node-2", kind: "sold-at", intendedPriceMinor: 2500n, currency: "USD" })).toEqual({ ok: true });
    expect(await declareTrigger({
      id: "trigger-1",
      nodeId: "node-1",
      factName: "revenue",
      operator: ">=",
      threshold: 100,
      clauses: [{ factName: "margin", operator: ">=", threshold: 20 }],
      composition: TriggerComposition.ANY,
    })).toEqual({ ok: true });
    expect(await declareTrigger({ nodeId: "node-2", factName: "users", operator: ">=", threshold: 10 })).toEqual({ ok: true });
    expect(await addFact({ name: "revenue", value: 250, observedAt: new Date("2026-08-16T12:00:00Z"), staleAfterDays: 30, dimension: "monthly" })).toEqual({ ok: true });
    expect(await addFact({ name: "users", value: 10, observedAt: new Date("2026-08-16T12:00:00Z"), staleAfterDays: 30 })).toEqual({ ok: true });
    expect(await promoteNode({ nodeId: "node-1", actor: "operator", role: "operator" })).toEqual({ ok: true });

    expect(client.createNode).toHaveBeenCalledTimes(2);
    expect(client.createEdge).toHaveBeenCalledTimes(2);
    expect(client.declareTrigger).toHaveBeenCalledWith(expect.objectContaining({ trigger: expect.objectContaining({ id: "trigger-1", clauses: [expect.objectContaining({ factName: "margin" })] }) }));
    expect(client.addFact).toHaveBeenCalledWith(expect.objectContaining({ fact: expect.objectContaining({ name: "revenue", dimension: "monthly" }) }));
    expect(client.promote).toHaveBeenCalledWith(expect.objectContaining({ nodeId: "node-1", role: "operator" }));
  });
});
