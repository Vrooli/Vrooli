import { beforeEach, describe, expect, it, vi } from "vitest";

const listIntakes = vi.hoisted(() => vi.fn());
const recordIntake = vi.hoisted(() => vi.fn());
const evaluateScope = vi.hoisted(() => vi.fn());
const listTargets = vi.hoisted(() => vi.fn());
const createTarget = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ listTargets, createTarget, listIntakes, recordIntake, evaluateScope }) }));
vi.mock("./client", () => ({ transport: {} }));

import { createTarget as create, evaluateScope as evaluate, listIntakes as list, listTargets as targets, recordIntake as record } from "./nutrition";

describe("nutrition intake API", () => {
  beforeEach(() => vi.clearAllMocks());
  it("lists persisted intake events", async () => {
    listIntakes.mockResolvedValue({ events: [{ id: "e1", date: "2026-09-18", recipeId: "r1", recipeRevision: 2n, nutrientId: "protein", amount: "41.5", unit: "g", reason: "portion", correctionOf: "", recordedAt: "" }] });
    await expect(list("w1")).resolves.toMatchObject([{ id: "e1", amount: "41.5" }]);
  });
  it("maps persisted targets and creates a target", async () => {
    listTargets.mockResolvedValue({ targets: [{ id: "t1", revision: 1n, nutrientId: "protein", lower: "80", upper: "unknown", period: "local_day", scope: "planned_day", enforcement: "preferred", provenance: "user_assertion", effectiveFrom: "2026-09-18T00:00:00Z", effectiveTo: "", active: true }] });
    createTarget.mockResolvedValue({ target: { id: "t2", revision: 1n, nutrientId: "protein", lower: "90", upper: "unknown", period: "local_day", scope: "planned_day", enforcement: "preferred", provenance: "user_assertion", effectiveFrom: "2026-09-18T00:00:00Z", effectiveTo: "", active: true } });
    await expect(targets("w1")).resolves.toMatchObject([{ id: "t1", lower: "80" }]);
    await expect(create({ workspaceId: "w1", nutrientId: "protein", lower: "90", upper: "unknown", period: "local_day", scope: "planned_day", enforcement: "preferred", provenance: "user_assertion", effectiveFrom: "2026-09-18T00:00:00Z" })).resolves.toMatchObject({ id: "t2", lower: "90" });
  });
  it("rejects an empty target response", async () => {
    createTarget.mockResolvedValue({});
    await expect(create({ workspaceId: "w1", nutrientId: "protein", lower: "1", upper: "unknown", period: "local_day", scope: "planned_day", enforcement: "preferred", provenance: "user_assertion", effectiveFrom: "2026-09-18T00:00:00Z" })).rejects.toThrow("no nutrition target");
  });
  it("records an intake event and keeps correction metadata", async () => {
    recordIntake.mockResolvedValue({ event: { id: "c1", date: "2026-09-18", recipeId: "r1", recipeRevision: 2n, nutrientId: "protein", amount: "40", unit: "g", reason: "correction", correctionOf: "e1", recordedAt: "" } });
    await expect(record({ workspaceId: "w1", id: "c1", date: "2026-09-18", recipeId: "r1", recipeRevision: 2n, nutrientId: "protein", amount: "40", unit: "g", reason: "correction", correctionOf: "e1" })).resolves.toMatchObject({ correctionOf: "e1" });
  });
  it("rejects an empty record response", async () => {
    recordIntake.mockResolvedValue({});
    await expect(record({ workspaceId: "w1", id: "e1", date: "2026-09-18", nutrientId: "protein", amount: "1", unit: "g" })).rejects.toThrow("no intake event");
  });
  it("evaluates a scope without losing completeness metadata", async () => {
    evaluateScope.mockResolvedValue({ nutrientId: "protein", known: "41.5", complete: false, unresolved: ["2026-09-17"], status: "unknown", reason: "unresolved contributions remain" });
    await expect(evaluate({ workspaceId: "w1", targetId: "t1", targetRevision: 1n, nutrientId: "protein", scope: "recorded_so_far", intakes: [{ date: "2026-09-17", nutrientId: "protein", actual: "41.5", recorded: true, past: true }] })).resolves.toMatchObject({ known: "41.5", complete: false, status: "unknown" });
  });
});
