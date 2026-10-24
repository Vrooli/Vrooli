import { describe, expect, it } from "vitest";

import { TupleKind, Verdict } from "../api/validationRecord";
import {
  segmentToTupleKind,
  summarizeVerdicts,
  summaryToVariant,
  tupleKindToSegment,
  verdictToKind,
} from "./verdict";

describe("verdict utilities", () => {
  it("maps verdicts and tuple-kind route segments", () => {
    expect(verdictToKind(Verdict.PASS, false)).toBe("pass");
    expect(verdictToKind(Verdict.PASS, true)).toBe("stale");
    expect(verdictToKind(Verdict.UNEXPECTED_MUTATION, false)).toBe("unexpected");
    expect(verdictToKind(Verdict.RUN_FAILURE, false)).toBe("failure");
    expect(verdictToKind(Verdict.TOOL_FAILURE, false)).toBe("failure");
    expect(verdictToKind(Verdict.UNSPECIFIED, false)).toBe("neutral");
    expect(tupleKindToSegment(TupleKind.TOOL)).toBe("tool");
    expect(tupleKindToSegment(TupleKind.SKILL)).toBe("skill");
    expect(segmentToTupleKind("tool")).toBe(TupleKind.TOOL);
    expect(segmentToTupleKind("anything-else")).toBe(TupleKind.SKILL);
  });

  it("summarizes verdicts with severity precedence", () => {
    const counts = summarizeVerdicts([
      {
        $typeName: "vrooli.development_toolchain_validator.v1.report.TupleVerdict",
        tupleKind: TupleKind.SKILL,
        subjectId: "pass",
        latestVerdict: Verdict.PASS,
        latestRecordId: "1",
        stale: false,
      },
      {
        $typeName: "vrooli.development_toolchain_validator.v1.report.TupleVerdict",
        tupleKind: TupleKind.SKILL,
        subjectId: "stale",
        latestVerdict: Verdict.PASS,
        latestRecordId: "2",
        stale: true,
      },
      {
        $typeName: "vrooli.development_toolchain_validator.v1.report.TupleVerdict",
        tupleKind: TupleKind.TOOL,
        subjectId: "fail",
        latestVerdict: Verdict.TOOL_FAILURE,
        latestRecordId: "3",
        stale: false,
      },
    ]);

    expect(counts).toMatchObject({ pass: 1, stale: 1, failure: 1, total: 3 });
    expect(summaryToVariant(counts)).toBe("failure");
    expect(summaryToVariant({ ...counts, failure: 0 })).toBe("stale");
    expect(summaryToVariant({ pass: 1, stale: 0, unexpected: 1, failure: 0, total: 2 })).toBe("unexpected");
    expect(summaryToVariant({ pass: 1, stale: 0, unexpected: 0, failure: 0, total: 1 })).toBe("pass");
    expect(summaryToVariant({ pass: 0, stale: 0, unexpected: 0, failure: 0, total: 0 })).toBe("neutral");
  });
});
