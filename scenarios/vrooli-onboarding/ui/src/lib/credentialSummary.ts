export type CredentialSummaryItem = {
  logical_id: string;
  field: string;
  required: boolean;
  status: string;
  evidence_status?: string;
};

const uncertainEvidence = new Set(["unverified", "unknown", "pending", "verification_unknown"]);

export function credentialNeedsAttention(
  credential: CredentialSummaryItem,
  lastSavedCredential?: string | null,
): boolean {
  if (!credential.required) return false;
  const key = `${credential.logical_id}/${credential.field}`;
  if (credential.status !== "configured" && key !== lastSavedCredential) return true;
  return uncertainEvidence.has(credential.evidence_status ?? "");
}

export function summarizeCredentials(
  credentials: CredentialSummaryItem[],
  lastSavedCredential?: string | null,
) {
  const required = credentials.filter((credential) => credential.required);
  const optional = credentials.filter((credential) => !credential.required);
  return {
    total: credentials.length,
    required: required.length,
    requiredNeedsAttention: required.filter((credential) => credentialNeedsAttention(credential, lastSavedCredential)).length,
    optional: optional.length,
    optionalDeferred: optional.filter((credential) => credential.status === "deferred").length,
  };
}
