import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";

import {
  MaturityAssessmentSchema,
} from "@vrooli/proto-types/common/v1/maturity_pb";
import {
  ValidationStatus,
  ValidateScenarioResponseSchema,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { validateScenario, validationClient } from "./validation";

describe("validateScenario (FromProto)", () => {
  it("converts severity + summary fields", async () => {
    const proto = create(ValidateScenarioResponseSchema, {
      scenario: "ui-health",
      status: ValidationStatus.FAILED,
      assessment: create(MaturityAssessmentSchema, {
        scenario: "ui-health",
        provider: "ui-health",
        phase: "ui-health",
        findings: [
          {
            severity: "SEVERITY_ERROR",
            code: "missing-slot",
            location: "foo.tsx",
            message: "Missing",
            remediation: "Add",
          },
          {
            severity: "SEVERITY_WARNING",
            code: "stale",
            location: "bar",
            message: "Stale",
            remediation: "",
          },
          {
            severity: "SEVERITY_INFO",
            code: "tip",
            location: "",
            message: "Tip",
            remediation: "",
          },
        ],
        findingsBySeverity: {
          SEVERITY_ERROR: 1,
          SEVERITY_WARNING: 1,
          SEVERITY_INFO: 1,
        },
      }),
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
