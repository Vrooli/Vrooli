/**
 * findingsModel — pure grouping/sorting/code-stripping logic. No React or
 * connect client; asserts the grouping contract the page relies on.
 */
import { describe, expect, it } from "vitest";

import {
  findingCapabilityId,
  findingDocPath,
  groupFindings,
  severityRank,
  strippedCode,
} from "./findingsModel";
import {
  makeBusinessContractReport,
  makeCapabilityRollup,
  makeContractFinding,
} from "./mocks/factories";

describe("strippedCode / findingDocPath", () => {
  it("strips a :CLAIM-ID suffix at the first colon", () => {
    expect(strippedCode("intent.ot_orphan:CLAIM-1")).toBe("intent.ot_orphan");
    expect(strippedCode("intent.ot_orphan")).toBe("intent.ot_orphan");
  });

  it("builds the doc path from the stripped code", () => {
    expect(findingDocPath("intent.ot_orphan:CLAIM-9")).toBe(
      "docs/findings/intent.ot_orphan.md",
    );
  });
});

describe("severityRank", () => {
  it("ranks errors before warnings before anything else", () => {
    expect(severityRank("error")).toBeLessThan(severityRank("warning"));
    expect(severityRank("warning")).toBeLessThan(severityRank("info"));
  });
});

describe("findingCapabilityId", () => {
  const caps = [makeCapabilityRollup({ capabilityId: "intent_linkage" })];

  it("maps a code prefix onto a capability id", () => {
    expect(findingCapabilityId("intent.ot_orphan", caps)).toBe("intent_linkage");
  });

  it("returns null when nothing matches", () => {
    expect(findingCapabilityId("evidence.stale", caps)).toBeNull();
  });
});

describe("groupFindings", () => {
  it("groups by capability and sorts errors before warnings within a group", () => {
    const report = makeBusinessContractReport({
      capabilities: [makeCapabilityRollup({ capabilityId: "intent_linkage" })],
      findings: [
        makeContractFinding({ code: "intent.warn", severity: "warning" }),
        makeContractFinding({ code: "intent.err", severity: "error" }),
      ],
    });

    const groups = groupFindings(report);
    expect(groups).toHaveLength(1);
    expect(groups[0]?.capability?.capabilityId).toBe("intent_linkage");
    expect(groups[0]?.findings.map((f) => f.severity)).toEqual(["error", "warning"]);
  });

  it("falls back to severity groups (errors first) for unmapped findings", () => {
    const report = makeBusinessContractReport({
      capabilities: [],
      findings: [
        makeContractFinding({ code: "evidence.stale", severity: "warning" }),
        makeContractFinding({ code: "evidence.missing", severity: "error" }),
      ],
    });

    const groups = groupFindings(report);
    expect(groups.map((g) => g.key)).toEqual(["severity:error", "severity:warning"]);
  });

  it("returns no groups for a missing report", () => {
    expect(groupFindings(undefined)).toEqual([]);
  });
});
