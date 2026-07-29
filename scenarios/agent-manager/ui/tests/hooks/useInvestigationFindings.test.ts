import assert from "node:assert/strict";
import { test } from "vitest";
import { getInvestigationFindings } from "../../src/hooks/useApi.js";
import { StructuredResultStatus, type Run } from "../../src/types.js";

function runWith(value: unknown, status = StructuredResultStatus.SUCCESS): Run {
  return {
    result: { structured: { status, value } },
  } as unknown as Run;
}

test("getInvestigationFindings accepts successful structured findings from object, text, and bytes", () => {
  const payload = {
    summary: "Tool failure rate is elevated",
    primaryCategory: "Reliability",
    confidence: 0.92,
    categories: [{ name: "Reliability", findings: ["retry shell invocation"] }],
  };
  assert.deepEqual(getInvestigationFindings(runWith(payload)), payload);
  assert.deepEqual(getInvestigationFindings(runWith(JSON.stringify(payload))), payload);
  assert.deepEqual(getInvestigationFindings(runWith(new TextEncoder().encode(JSON.stringify(payload)))), payload);
});

test("getInvestigationFindings is conservative for non-success, malformed, and incomplete outputs", () => {
  assert.equal(getInvestigationFindings(null), null);
  assert.equal(getInvestigationFindings(runWith("{}")), null);
  assert.equal(getInvestigationFindings(runWith("not json")), null);
  assert.equal(getInvestigationFindings(runWith(new Uint8Array())), null);
  assert.equal(getInvestigationFindings(runWith({ categories: "not an array" })), null);
  assert.equal(getInvestigationFindings(runWith({ categories: [] }, StructuredResultStatus.INVALID)), null);
  assert.deepEqual(getInvestigationFindings(runWith({ categories: [] })), {
    summary: "",
    primaryCategory: "Both",
    confidence: undefined,
    categories: [],
  });
});
