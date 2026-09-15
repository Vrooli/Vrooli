import { fromJson, toBinary } from "@bufbuild/protobuf";
import { StructSchema } from "@bufbuild/protobuf/wkt";
import { afterEach, describe, expect, it, vi } from "vitest";

import { makeFixResponse, makeValidationResponse } from "../test-utils";
import { ValidationStatus } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

const mockClient = vi.hoisted(() => ({
  validateScenario: vi.fn(),
  previewFix: vi.fn(),
}));

vi.mock("@connectrpc/connect", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@connectrpc/connect")>();
  return {
    ...actual,
    createClient: () => mockClient,
  };
});

describe("api/validation", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("calls ValidateScenario and unpacks native Struct detail", async () => {
    const nativeStruct = fromJson(StructSchema, {
      scenario: "api-health",
      target: {
        scenario: "api-health",
        resolution: "resolved",
        health_probe: {
          requested: true,
          status_code: 200,
        },
      },
      summary: {
        errors: 0,
        warnings: 0,
        infos: 0,
        passed: true,
      },
    });
    mockClient.validateScenario.mockResolvedValueOnce(
      makeValidationResponse({
        nativeDetail: {
          typeUrl: "type.googleapis.com/google.protobuf.Struct",
          value: toBinary(StructSchema, nativeStruct),
        },
      }),
    );
    const { validateScenario } = await import("./validation");

    const report = await validateScenario({
      scenario: "api-health",
      path: "/tmp/api-health",
      includeExecution: true,
    });

    expect(mockClient.validateScenario).toHaveBeenCalledWith({
      scenario: "api-health",
      path: "/tmp/api-health",
      includeExecution: true,
    });
    expect(report.nativeDetail.target?.resolution).toBe("resolved");
    expect(report.nativeDetail.target?.health_probe?.status_code).toBe(200);
  });

  it("calls PreviewFix with normalized optional fields", async () => {
    mockClient.previewFix.mockResolvedValueOnce(makeFixResponse());
    const { previewFix } = await import("./validation");

    await previewFix({ scenario: "api-health" });

    expect(mockClient.previewFix).toHaveBeenCalledWith({
      scenario: "api-health",
      path: "",
      ruleIds: [],
    });
  });

  it("renders canonical validation status labels", async () => {
    const { statusLabel } = await import("./validation");

    expect(statusLabel(ValidationStatus.PASSED)).toBe("passed");
    expect(statusLabel(ValidationStatus.FAILED)).toBe("failed");
    expect(statusLabel(ValidationStatus.DEGRADED)).toBe("degraded");
    expect(statusLabel(ValidationStatus.ERROR)).toBe("error");
    expect(statusLabel(ValidationStatus.SKIPPED)).toBe("skipped");
    expect(statusLabel(ValidationStatus.UNSPECIFIED)).toBe("unspecified");
  });
});
