import { describe, expect, it } from "vitest";
import { credentialNeedsAttention, summarizeCredentials } from "./credentialSummary";

const row = (overrides: Partial<Parameters<typeof credentialNeedsAttention>[0]> = {}) => ({
  logical_id: "mail",
  field: "api_key",
  required: true,
  status: "pending",
  ...overrides,
});

describe("credential summaries", () => {
  it("separates declared, required, actionable, optional and deferred counts", () => {
    expect(summarizeCredentials([
      row(),
      row({ logical_id: "storage", status: "configured" }),
      row({ logical_id: "mail", field: "region", required: false, status: "deferred" }),
      row({ logical_id: "cache", required: false, status: "unconfigured" }),
    ])).toEqual({
      total: 4,
      required: 2,
      requiredNeedsAttention: 1,
      optional: 2,
      optionalDeferred: 1,
    });
  });

  it("does not count a just-saved required value as outstanding while readiness refreshes", () => {
    const saved = row();
    expect(credentialNeedsAttention(saved)).toBe(true);
    expect(credentialNeedsAttention(saved, "mail/api_key")).toBe(false);
    expect(summarizeCredentials([saved], "mail/api_key").requiredNeedsAttention).toBe(0);
  });

  it("keeps required configured values needing verification in the attention count", () => {
    expect(credentialNeedsAttention(row({ status: "configured", evidence_status: "unverified" }))).toBe(true);
  });
});
