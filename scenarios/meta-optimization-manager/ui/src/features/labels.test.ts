import { describe, it, expect } from "vitest";

import { Projection, CellStatus, DenominatorConfidence, GapAxis } from "@vrooli/proto-types/meta-optimization-manager/v1/shared/model_pb";
import {
  FitnessTier,
  ReferenceEligibility,
} from "@vrooli/proto-types/meta-optimization-manager/v1/convergence/convergence_pb";
import { TrialVerdict } from "@vrooli/proto-types/meta-optimization-manager/v1/trials/trials_pb";

import {
  projectionLabel,
  cellStatusLabel,
  gapAxisLabel,
  confidenceLabel,
  tierLabel,
  eligibilityLabel,
  verdictLabel,
  pct,
} from "./labels";

describe("labels", () => {
  it("maps projections (incl. unspecified → cross-cutting)", () => {
    expect(projectionLabel(Projection.ANSWER)).toBe("answer");
    expect(projectionLabel(Projection.VALIDATE)).toBe("validate");
    expect(projectionLabel(Projection.GUIDE)).toBe("guide");
    expect(projectionLabel(Projection.UNSPECIFIED)).toBe("cross-cutting");
  });

  it("maps cell statuses", () => {
    expect(cellStatusLabel(CellStatus.NOW)).toBe("now");
    expect(cellStatusLabel(CellStatus.IN_REACH)).toBe("in_reach");
    expect(cellStatusLabel(CellStatus.MISSING)).toBe("missing");
    expect(cellStatusLabel(CellStatus.UNSPECIFIED)).toBe("?");
  });

  it("maps gap axes", () => {
    expect(gapAxisLabel(GapAxis.COVERAGE)).toBe("coverage");
    expect(gapAxisLabel(GapAxis.EMPIRICAL)).toBe("empirical");
    expect(gapAxisLabel(GapAxis.UNSPECIFIED)).toBe("?");
  });

  it("maps denominator confidence", () => {
    expect(confidenceLabel(DenominatorConfidence.AUTHORITATIVE)).toBe("authoritative");
    expect(confidenceLabel(DenominatorConfidence.PARTIAL)).toBe("partial");
    expect(confidenceLabel(DenominatorConfidence.SKETCH)).toBe("sketch");
    expect(confidenceLabel(DenominatorConfidence.UNSPECIFIED)).toBe("unspecified");
  });

  it("maps fitness tiers", () => {
    expect(tierLabel(FitnessTier.STRONG)).toBe("strong");
    expect(tierLabel(FitnessTier.FAIR)).toBe("fair");
    expect(tierLabel(FitnessTier.WEAK)).toBe("weak");
    expect(tierLabel(FitnessTier.UNSPECIFIED)).toBe("?");
  });

  it("maps reference eligibility", () => {
    expect(eligibilityLabel(ReferenceEligibility.ELIGIBLE)).toBe("eligible");
    expect(eligibilityLabel(ReferenceEligibility.CANDIDATE)).toBe("candidate");
    expect(eligibilityLabel(ReferenceEligibility.INELIGIBLE)).toBe("ineligible");
    expect(eligibilityLabel(ReferenceEligibility.UNSPECIFIED)).toBe("?");
  });

  it("maps trial verdicts", () => {
    expect(verdictLabel(TrialVerdict.PASS)).toBe("pass");
    expect(verdictLabel(TrialVerdict.FAIL)).toBe("fail");
    expect(verdictLabel(TrialVerdict.ERROR)).toBe("error");
    expect(verdictLabel(TrialVerdict.UNSPECIFIED)).toBe("?");
  });

  it("formats ratios as whole-number percentages", () => {
    expect(pct(0)).toBe("0");
    expect(pct(0.5)).toBe("50");
    expect(pct(0.083)).toBe("8");
    expect(pct(1)).toBe("100");
  });
});
