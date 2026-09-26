import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";

import {
  ReindexCancelResponseSchema,
  ReindexResponseSchema,
  ReindexStatusResponseSchema,
} from "@vrooli/proto-types/ui-health/v1/reindex/reindex_pb";

import {
  isTerminal,
  reindex,
  reindexCancel,
  reindexClient,
  reindexStateFromString,
  reindexStatus,
} from "./reindex";

describe("reindexStateFromString", () => {
  it.each([
    ["queued", "queued"],
    ["running", "running"],
    ["succeeded", "succeeded"],
    ["failed", "failed"],
    ["cancelled", "cancelled"],
    ["", "unknown"],
    ["garbage", "unknown"],
  ] as const)("maps %s → %s", (input, expected) => {
    expect(reindexStateFromString(input)).toBe(expected);
  });
});

describe("isTerminal", () => {
  it("returns true for succeeded/failed/cancelled", () => {
    expect(isTerminal("succeeded")).toBe(true);
    expect(isTerminal("failed")).toBe(true);
    expect(isTerminal("cancelled")).toBe(true);
  });
  it("returns false for in-flight states", () => {
    expect(isTerminal("queued")).toBe(false);
    expect(isTerminal("running")).toBe(false);
    expect(isTerminal("unknown")).toBe(false);
  });
});

describe("reindex client wrappers", () => {
  it("reindex() converts the trigger response", async () => {
    const proto = create(ReindexResponseSchema, {
      jobId: "j-1",
      plannedUpserts: 3,
      plannedDeletes: 1,
      dryRun: true,
    });
    vi.spyOn(reindexClient, "reindex").mockResolvedValueOnce(proto);
    expect(await reindex("ui-health", true)).toEqual({
      jobId: "j-1",
      plannedUpserts: 3,
      plannedDeletes: 1,
      dryRun: true,
    });
  });

  it("reindexStatus() converts state strings", async () => {
    const proto = create(ReindexStatusResponseSchema, {
      jobId: "j-1",
      state: "running",
      processed: 2,
      total: 4,
      error: "",
    });
    vi.spyOn(reindexClient, "reindexStatus").mockResolvedValueOnce(proto);
    const out = await reindexStatus("j-1");
    expect(out.state).toBe("running");
    expect(out.processed).toBe(2);
  });

  it("reindexCancel() forwards the cancelled flag", async () => {
    const proto = create(ReindexCancelResponseSchema, {
      jobId: "j-1",
      cancelled: true,
    });
    vi.spyOn(reindexClient, "reindexCancel").mockResolvedValueOnce(proto);
    expect(await reindexCancel("j-1")).toEqual({ jobId: "j-1", cancelled: true });
  });
});
