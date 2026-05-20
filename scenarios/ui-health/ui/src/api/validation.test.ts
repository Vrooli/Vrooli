import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";

import {
  Severity,
  ValidateScenarioResponseSchema,
} from "@vrooli/proto-types/ui-health/v1/validation/validation_pb";

import { validateScenario, validationClient } from "./validation";

describe("validateScenario (FromProto)", () => {
  it("converts severity + summary fields", async () => {
    const proto = create(ValidateScenarioResponseSchema, {
      scenario: "ui-health",
      passed: false,
      findings: [
        {
          severity: Severity.ERROR,
          code: "missing-slot",
          location: "foo.tsx",
          message: "Missing",
          suggestion: "Add",
        },
        {
          severity: Severity.WARNING,
          code: "stale",
          location: "bar",
          message: "Stale",
          suggestion: "",
        },
        {
          severity: Severity.INFO,
          code: "tip",
          location: "",
          message: "Tip",
          suggestion: "",
        },
      ],
      summary: { errors: 1, warnings: 1, infos: 1 },
    });
    vi.spyOn(validationClient, "validateScenario").mockResolvedValueOnce(proto);
    const out = await validateScenario("ui-health");
    expect(out.passed).toBe(false);
    expect(out.findings.map((f) => f.severity)).toEqual([
      "error",
      "warning",
      "info",
    ]);
    expect(out.summary).toEqual({ errors: 1, warnings: 1, infos: 1 });
  });
});
