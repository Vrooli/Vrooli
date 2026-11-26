import { describe, expect, it } from "vitest";

import type { ComponentExperience } from "../api/components";
import { evidenceStatusFor, summarizeExperience } from "./experienceSummary";

function experience(overrides: Partial<ComponentExperience> = {}): ComponentExperience {
  return {
    componentId: "cmp",
    libraryId: "react-component-library:Card",
    version: "1.0.0",
    contractId: "card",
    title: "Card",
    purpose: "",
    states: [{ id: "default", exampleName: "default", description: "" }],
    claims: [
      {
        id: "present",
        type: "element-present",
        statement: "Card is present.",
        tier: "machine",
        states: ["default"],
      },
    ],
    evidence: [],
    evidenceStatus: "covered",
    evidenceMessage: "",
    ...overrides,
  };
}

describe("summarizeExperience", () => {
  it("does not count a skipped capture as state coverage", () => {
    const summary = summarizeExperience(
      experience({
        evidence: [
          {
            claimId: "present",
            verdict: "skipped",
            stateId: "default",
            exampleName: "default",
            captureRef: "",
            checkedAt: "2026-11-22T10:00:00Z",
            message: "capture unavailable",
            viewport: "phone",
            viewportWidth: 390,
            viewportHeight: 844,
          },
        ],
      }),
    );
    expect(summary.stateCoverage).toMatchObject({
      total: 1,
      covered: 0,
      status: "unmeasured",
      captures: 1,
      message: "capture unavailable",
    });
    expect(summary.status).toBe("unmeasured");
  });

  it("treats a passed past-tense capture as real coverage", () => {
    const summary = summarizeExperience(
      experience({
        evidence: [
          {
            claimId: "present",
            verdict: "passed",
            stateId: "default",
            exampleName: "default",
            captureRef: "",
            checkedAt: "2026-11-22T10:00:00Z",
            message: "",
            viewport: "phone",
            viewportWidth: 390,
            viewportHeight: 844,
          },
        ],
      }),
    );
    expect(summary.stateCoverage?.covered).toBe(1);
    expect(summary.stateCoverage?.status).toBe("covered");
  });

  it("distinguishes partial coverage from complete coverage", () => {
    const partial = summarizeExperience(
      experience({
        states: [
          { id: "default", exampleName: "default", description: "" },
          { id: "empty", exampleName: "empty", description: "" },
        ],
        evidence: [
          {
            claimId: "present",
            verdict: "passed",
            stateId: "default",
            exampleName: "default",
            captureRef: "",
            checkedAt: "2026-11-22T10:00:00Z",
            message: "",
            viewport: "phone",
            viewportWidth: 390,
            viewportHeight: 844,
          },
        ],
      }),
    );
    expect(partial.stateCoverage).toMatchObject({
      total: 2,
      covered: 1,
      status: "partial",
      captures: 1,
    });
  });

  it("derives coverage instead of trusting a legacy transport status", () => {
    const summary = summarizeExperience(
      experience({
        evidenceStatus: "available",
        evidence: [
          {
            claimId: "present",
            verdict: "passed",
            stateId: "default",
            exampleName: "default",
            captureRef: "",
            checkedAt: "2026-11-22T10:00:00Z",
            message: "",
            viewport: "phone",
            viewportWidth: 390,
            viewportHeight: 844,
          },
        ],
      }),
    );
    expect(summary.status).toBe("covered");
  });

  it("stays truthful when the manager is unavailable", () => {
    const summary = summarizeExperience(
      experience({ evidenceStatus: "unavailable", evidence: [] }),
    );
    expect(summary.status).toBe("unavailable");
    expect(evidenceStatusFor(undefined)).toBe("not-configured");
  });

  it("does not invent coverage for a contract without states", () => {
    const summary = summarizeExperience(experience({ states: [], claims: [] }));
    expect(summary.stateCoverage).toBeNull();
    expect(summary.status).toBe("unmeasured");
  });
});
