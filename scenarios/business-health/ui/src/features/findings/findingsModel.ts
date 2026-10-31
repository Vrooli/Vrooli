/**
 * Pure helpers for the findings surface: code-suffix stripping, doc-path
 * derivation, and grouping findings by capability (falling back to severity)
 * with errors sorted ahead of warnings. Kept dependency-free so the grouping
 * logic is unit-testable without React or the connect client.
 */
import type {
  BusinessContractReport,
  CapabilityRollup,
  ContractFinding,
} from "@vrooli/proto-types/business-health/v1/contract/contract_pb";

/**
 * Strip an optional ":CLAIM-ID" suffix from a finding code. Finding codes may
 * carry a per-claim suffix at runtime (e.g. "intent.ot_orphan:CLAIM-1"); the
 * canonical rule id — used for doc lookup and fixer selection — is everything
 * before the first colon.
 */
export const strippedCode = (code: string): string => code.split(":")[0] ?? code;

/** Repo-relative remediation doc path for a finding code (suffix stripped). */
export const findingDocPath = (code: string): string =>
  `docs/findings/${strippedCode(code)}.md`;

/** Sort rank so errors sort ahead of warnings ahead of anything else. */
export const severityRank = (severity: string): number => {
  const normalized = severity.toLowerCase();
  if (normalized === "error") return 0;
  if (normalized === "warning") return 1;
  return 2;
};

export interface FindingGroup {
  /** Stable id for the React key and group testid namespace. */
  readonly key: string;
  /** Heading label — a capability descriptor or a severity name. */
  readonly label: string;
  /** Present when the group is anchored to a capability rollup. */
  readonly capability?: CapabilityRollup;
  readonly findings: readonly ContractFinding[];
}

/**
 * Resolve the capability a finding belongs to by matching the leading dotted
 * segment of its (suffix-stripped) code against the known capability ids
 * (e.g. code "intent.ot_orphan" → capability "intent_linkage"). Returns null
 * when nothing matches so the caller falls back to grouping by severity.
 */
export const findingCapabilityId = (
  code: string,
  capabilities: readonly CapabilityRollup[],
): string | null => {
  const prefix = strippedCode(code).split(".")[0] ?? "";
  if (!prefix) return null;
  for (const cap of capabilities) {
    const id = cap.capabilityId;
    if (id === prefix || id.startsWith(`${prefix}_`) || prefix.startsWith(id)) {
      return id;
    }
  }
  return null;
};

/**
 * Group a report's findings by capability, falling back to severity for
 * findings that don't map to a capability. Capability groups come first (in
 * the report's capability declaration order), then severity groups (errors
 * before warnings). Within every group, errors sort ahead of warnings.
 */
export const groupFindings = (
  report: BusinessContractReport | undefined,
): FindingGroup[] => {
  if (!report) return [];

  const byCapability = new Map<string, ContractFinding[]>();
  const bySeverity = new Map<string, ContractFinding[]>();

  for (const finding of report.findings) {
    const capId = findingCapabilityId(finding.code, report.capabilities);
    if (capId) {
      const bucket = byCapability.get(capId) ?? [];
      bucket.push(finding);
      byCapability.set(capId, bucket);
    } else {
      const severity = finding.severity.toLowerCase() || "other";
      const bucket = bySeverity.get(severity) ?? [];
      bucket.push(finding);
      bySeverity.set(severity, bucket);
    }
  }

  const sortFindings = (list: ContractFinding[]): ContractFinding[] =>
    [...list].sort((a, b) => severityRank(a.severity) - severityRank(b.severity));

  const groups: FindingGroup[] = [];

  for (const cap of report.capabilities) {
    const bucket = byCapability.get(cap.capabilityId);
    if (bucket && bucket.length > 0) {
      groups.push({
        key: `capability:${cap.capabilityId}`,
        label: cap.levelName ? `${cap.capabilityId} · ${cap.levelName}` : cap.capabilityId,
        capability: cap,
        findings: sortFindings(bucket),
      });
    }
  }

  const severityKeys = [...bySeverity.keys()].sort(
    (a, b) => severityRank(a) - severityRank(b) || a.localeCompare(b),
  );
  for (const severity of severityKeys) {
    const bucket = bySeverity.get(severity);
    if (bucket && bucket.length > 0) {
      groups.push({
        key: `severity:${severity}`,
        label: severity,
        findings: sortFindings(bucket),
      });
    }
  }

  return groups;
};
