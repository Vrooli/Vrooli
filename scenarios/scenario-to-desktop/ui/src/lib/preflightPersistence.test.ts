import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  PreflightResponseSchema,
  PreflightStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import {
  deserializePreflight,
  serializePreflight,
} from "./preflightPersistence";

describe("preflight persistence", () => {
  it("round-trips generated preflight evidence without a shadow DTO", () => {
    const result = create(PreflightResponseSchema, {
      status: PreflightStatus.PASSED,
      sessionId: "session-1",
      ports: [{ serviceId: "api", name: "http", port: 8080 }],
      validation: { valid: true },
    });

    expect(deserializePreflight(serializePreflight(result))).toEqual(result);
  });

  it("rejects malformed persisted values", () => {
    expect(deserializePreflight("not-a-message")).toBeNull();
  });
});
