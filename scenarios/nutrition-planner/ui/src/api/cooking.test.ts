import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { CookingSessionSchema } from "@vrooli/proto-types/nutrition-planner/v1/cooking/cooking_pb";

const start = vi.hoisted(() => vi.fn());
const get = vi.hoisted(() => vi.fn());
const list = vi.hoisted(() => vi.fn());
const save = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ startSession: start, getSession: get, listSessions: list, saveSession: save }) }));
vi.mock("./client", () => ({ transport: {} }));

import { getCookingSession, listCookingSessions, saveCookingSession, startCookingSession } from "./cooking";

const session = create(CookingSessionSchema, { id: "cook-1", workspaceId: "w1", recipeId: "r1", recipeRevision: 2n, methodId: "stove", scale: "4", status: "active", version: 1n });

describe("cooking API", () => {
  beforeEach(() => vi.clearAllMocks());
  it("starts a pinned session and returns its durable record", async () => { start.mockResolvedValue({ session }); await expect(startCookingSession({ workspaceId: "w1", sessionId: "cook-1", recipeId: "r1", recipeRevision: 2n, methodId: "stove", scale: "4" })).resolves.toBe(session); expect(start).toHaveBeenCalledWith(expect.objectContaining({ recipeRevision: 2n, methodId: "stove", scale: "4" })); });
  it("rejects a start response without a session", async () => { start.mockResolvedValue({}); await expect(startCookingSession({ workspaceId: "w1", sessionId: "cook-1", recipeId: "r1", recipeRevision: 2n, methodId: "stove", scale: "4" })).rejects.toThrow("no cooking session"); });
  it("loads and lists persisted cooking sessions", async () => { get.mockResolvedValue({ session }); list.mockResolvedValue({ sessions: [session] }); await expect(getCookingSession("w1", "cook-1")).resolves.toBe(session); await expect(listCookingSessions("w1")).resolves.toEqual([session]); });
  it("rejects empty get and save responses", async () => { get.mockResolvedValue({}); save.mockResolvedValue({}); await expect(getCookingSession("w1", "cook-1")).rejects.toThrow("no cooking session"); await expect(saveCookingSession({ workspaceId: "w1", sessionId: "cook-1", eventId: "e1", expectedVersion: 1n, currentStepIndex: 0, completedSteps: [], timers: [] })).rejects.toThrow("no updated cooking session"); });
  it("sends explicit finish and optional yield state", async () => { save.mockResolvedValue({ session }); await saveCookingSession({ workspaceId: "w1", sessionId: "cook-1", eventId: "finish-1", expectedVersion: 1n, currentStepIndex: 1, completedSteps: ["step-1"], timers: [], finish: true, actualYield: "3.5", yieldUnit: "bowls" }); expect(save).toHaveBeenCalledWith(expect.objectContaining({ finish: true, actualYield: "3.5", yieldUnit: "bowls", expectedVersion: 1n })); });
});
