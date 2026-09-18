import { beforeEach, describe, expect, it, vi } from "vitest";

const listIntakes = vi.hoisted(() => vi.fn());
const recordIntake = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ listIntakes, recordIntake }) }));
vi.mock("./client", () => ({ transport: {} }));

import { listIntakes as list, recordIntake as record } from "./nutrition";

describe("nutrition intake API", () => {
  beforeEach(() => vi.clearAllMocks());
  it("lists persisted intake events", async () => {
    listIntakes.mockResolvedValue({ events: [{ id: "e1", date: "2026-09-18", recipeId: "r1", recipeRevision: 2n, nutrientId: "protein", amount: "41.5", unit: "g", reason: "portion", correctionOf: "", recordedAt: "" }] });
    await expect(list("w1")).resolves.toMatchObject([{ id: "e1", amount: "41.5" }]);
  });
  it("records an intake event and keeps correction metadata", async () => {
    recordIntake.mockResolvedValue({ event: { id: "c1", date: "2026-09-18", recipeId: "r1", recipeRevision: 2n, nutrientId: "protein", amount: "40", unit: "g", reason: "correction", correctionOf: "e1", recordedAt: "" } });
    await expect(record({ workspaceId: "w1", id: "c1", date: "2026-09-18", recipeId: "r1", recipeRevision: 2n, nutrientId: "protein", amount: "40", unit: "g", reason: "correction", correctionOf: "e1" })).resolves.toMatchObject({ correctionOf: "e1" });
  });
  it("rejects an empty record response", async () => {
    recordIntake.mockResolvedValue({});
    await expect(record({ workspaceId: "w1", id: "e1", date: "2026-09-18", nutrientId: "protein", amount: "1", unit: "g" })).rejects.toThrow("no intake event");
  });
});
