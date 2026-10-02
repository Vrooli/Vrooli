import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { ReplaySpecSchema } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import { serializeExportPayload } from "./replaySpecJson";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

describe("serializeExportPayload", () => {
  it("serializes generated ReplaySpec with stable proto field names", () => {
    const movieSpec = create(ReplaySpecSchema, {
      version: "2025-11-07",
      generatedAt: { seconds: 1_789_475_400n, nanos: 0 },
      execution: {
        executionId: "00000000-0000-4000-8000-000000000001",
        workflowId: "00000000-0000-4000-8000-000000000002",
        status: "completed",
      },
    });

    const payload: unknown = JSON.parse(serializeExportPayload({ movie_spec: movieSpec }));
    expect(isRecord(payload), "serialized payload must be an object").toBe(true);
    if (!isRecord(payload) || !isRecord(payload.movie_spec)) {
      throw new Error("serialized movie_spec missing");
    }
    const serializedSpec = payload.movie_spec;

    expect(serializedSpec.generated_at).toBe("2026-09-15T12:30:00Z");
    expect(serializedSpec).not.toHaveProperty("generatedAt");
    expect(serializedSpec).not.toHaveProperty("$typeName");
  });
});
