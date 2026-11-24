export type AdoptionState =
  | "draft"
  | "awaiting_secret"
  | "credential_stored"
  | "bridge_enrolling"
  | "ready"
  | "failed"
  | "released";

export interface AdoptionDraftInput {
  displayName?: string;
  host: string;
  user: string;
  monthlyCostMinor: string | number;
  currency: string;
  taxBasis: string;
}

export interface AdoptionDraft {
  displayName?: string;
  host: string;
  user: string;
  monthlyCostMinor: number;
  currency: string;
  taxBasis: string;
}

export type AdoptionDraftError =
  | "address_required"
  | "ssh_user_required"
  | "monthly_cost_invalid"
  | "currency_invalid"
  | "tax_basis_required";

export type AdoptionDraftResult =
  | { ok: true; value: AdoptionDraft }
  | { ok: false; error: AdoptionDraftError };

/** Validate only non-secret facts. There is intentionally no credential input. */
export function buildAdoptionDraft(input: AdoptionDraftInput): AdoptionDraftResult {
  const host = input.host.trim();
  const user = input.user.trim();
  const currency = input.currency.trim().toUpperCase();
  const taxBasis = input.taxBasis.trim();
  const rawCost = typeof input.monthlyCostMinor === "number" ? String(input.monthlyCostMinor) : input.monthlyCostMinor.trim();
  const monthlyCostMinor = rawCost === "" ? Number.NaN : Number(rawCost);

  if (!host) return { ok: false, error: "address_required" };
  if (!user) return { ok: false, error: "ssh_user_required" };
  if (!Number.isSafeInteger(monthlyCostMinor) || monthlyCostMinor < 0) {
    return { ok: false, error: "monthly_cost_invalid" };
  }
  if (!/^[A-Z]{3}$/.test(currency)) return { ok: false, error: "currency_invalid" };
  if (!taxBasis) return { ok: false, error: "tax_basis_required" };

  const displayName = input.displayName?.trim();
  return {
    ok: true,
    value: {
      ...(displayName ? { displayName } : {}),
      host,
      user,
      monthlyCostMinor,
      currency,
      taxBasis,
    },
  };
}

export interface AdoptionRecordSnapshot {
  id: string;
  displayName?: string;
  credentialStepRef?: string;
  host: string;
  user: string;
  monthlyCostMinor: number;
  currency: string;
  taxBasis: string;
  state: AdoptionState;
  adoptionEvidenceVerified: boolean;
  failureMessage?: string;
}

/** Safe, client-displayable allowlist derived from persisted adoption state. */
export interface AdoptionViewModel extends AdoptionRecordSnapshot {
  state: AdoptionState;
  statusReason?: "verification_pending_without_evidence" | "adoption_failed";
  credentialStepRef?: string;
  releaseEffect: "record_and_bridge_attachment_only";
}

/**
 * Convert persisted API state to the UI's safe view. Inconsistent `ready`
 * records fail closed: without persisted verified evidence, the UI shows the
 * in-progress state and cannot announce completion.
 */
export function toAdoptionViewModel(record: AdoptionRecordSnapshot): AdoptionViewModel {
  const state: AdoptionState = record.state === "ready" && !record.adoptionEvidenceVerified
    ? "bridge_enrolling"
    : record.state;
  const statusReason = record.state === "ready" && !record.adoptionEvidenceVerified
    ? "verification_pending_without_evidence"
    : record.state === "failed" ? "adoption_failed" : undefined;

  return {
    id: record.id,
    ...(record.displayName ? { displayName: record.displayName } : {}),
    ...(record.credentialStepRef ? { credentialStepRef: record.credentialStepRef } : {}),
    host: record.host,
    user: record.user,
    monthlyCostMinor: record.monthlyCostMinor,
    currency: record.currency,
    taxBasis: record.taxBasis,
    state,
    adoptionEvidenceVerified: record.adoptionEvidenceVerified,
    ...(statusReason ? { statusReason } : {}),
    releaseEffect: "record_and_bridge_attachment_only",
  };
}
