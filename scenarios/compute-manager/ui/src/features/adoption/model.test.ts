import { describe, expect, it } from "vitest";

import { buildAdoptionDraft, toAdoptionViewModel, type AdoptionRecordSnapshot } from "./model";

const validDraft = {
  displayName: "  build host  ",
  host: "  build.example.test ",
  user: " runner ",
  monthlyCostMinor: "1250",
  currency: "usd",
  taxBasis: "operator-declared, pre-tax",
};

const record = (overrides: Partial<AdoptionRecordSnapshot> = {}): AdoptionRecordSnapshot => ({
  id: "adoption-1",
  displayName: "Build host",
  host: "build.example.test",
  user: "runner",
  monthlyCostMinor: 1250,
  currency: "USD",
  taxBasis: "operator-declared, pre-tax",
  state: "ready",
  adoptionEvidenceVerified: true,
  ...overrides,
});

describe("adopted VPS view model", () => {
  it("normalizes explicit non-secret facts without inventing cost or tax defaults", () => {
    expect(buildAdoptionDraft(validDraft)).toEqual({
      ok: true,
      value: {
        displayName: "build host",
        host: "build.example.test",
        user: "runner",
        monthlyCostMinor: 1250,
        currency: "USD",
        taxBasis: "operator-declared, pre-tax",
      },
    });
    expect(buildAdoptionDraft({ ...validDraft, monthlyCostMinor: "0" }).ok).toBe(true);
    expect(buildAdoptionDraft({ ...validDraft, monthlyCostMinor: "" })).toEqual({ ok: false, error: "monthly_cost_invalid" });
    expect(buildAdoptionDraft({ ...validDraft, monthlyCostMinor: "-1" })).toEqual({ ok: false, error: "monthly_cost_invalid" });
    expect(buildAdoptionDraft({ ...validDraft, taxBasis: "" })).toEqual({ ok: false, error: "tax_basis_required" });
  });

  it("drops unexpected secret-shaped values from client-visible draft state", () => {
    const withUnexpectedSecret = { ...validDraft, password: "do-not-retain", privateKey: "do-not-retain" } as typeof validDraft;
    const result = buildAdoptionDraft(withUnexpectedSecret);
    expect(result.ok).toBe(true);
    expect(JSON.stringify(result)).not.toContain("do-not-retain");
    expect(Object.keys(result.ok ? result.value : {})).not.toContain("password");
  });

  it("does not report ready without persisted verified readiness evidence", () => {
    const view = toAdoptionViewModel(record({ adoptionEvidenceVerified: false }));
    expect(view.state).toBe("bridge_enrolling");
    expect(view.statusReason).toBe("verification_pending_without_evidence");
  });

  it("accepts ready only with persisted verified evidence", () => {
    expect(toAdoptionViewModel(record()).state).toBe("ready");
    expect(toAdoptionViewModel(record({ adoptionEvidenceVerified: false })).state).toBe("bridge_enrolling");
  });

  it("makes release semantics explicit without implying VPS destruction", () => {
    const view = toAdoptionViewModel(record({ state: "released" }));
    expect(view.state).toBe("released");
    expect(view.releaseEffect).toBe("record_and_bridge_attachment_only");
  });
});
