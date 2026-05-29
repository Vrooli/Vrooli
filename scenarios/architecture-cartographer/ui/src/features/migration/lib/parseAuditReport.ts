/**
 * Parse a test-genie `--json` SuiteExecutionResult into the shared
 * ArchitectureFinding[] the migration tracker ingests.
 *
 * This is the browser twin of the CLI's `loadAuditFindings`
 * (cli/domains/migration/handlers.go): it flattens every phase's `findings`
 * into one slice. The report is produced by Go `encoding/json` over the
 * generated proto structs, so enum fields arrive as INTEGERS and field names
 * are the proto names (`scenario`, `source`, `code`, `severity`, …). Those
 * integers are exactly the protobuf-es enum values (same .proto), so they
 * drop straight into the message init shape — no name/number translation.
 *
 * The migration domain RECOMPUTES the afid stable id server-side from
 * (scenario, source, code, locations), so we deliberately do not forward any
 * client-supplied stable_id — only the hash inputs and display fields.
 */
import { create } from "@bufbuild/protobuf";
import {
  ArchitectureFindingSchema,
  type ArchitectureFinding,
} from "@vrooli/proto-types/architecture/v1/findings_pb";

/** Raw finding shape as it appears in a Go-serialized report (loose). */
interface RawFinding {
  scenario?: unknown;
  source?: unknown;
  code?: unknown;
  severity?: unknown;
  locations?: unknown;
  domains?: unknown;
  message?: unknown;
  suggestion?: unknown;
}

export class AuditReportParseError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "AuditReportParseError";
  }
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((v): v is string => typeof v === "string");
}

function asInt(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function toFinding(raw: RawFinding): ArchitectureFinding {
  return create(ArchitectureFindingSchema, {
    scenario: asString(raw.scenario),
    source: asInt(raw.source),
    code: asString(raw.code),
    severity: asInt(raw.severity),
    locations: asStringArray(raw.locations),
    domains: asStringArray(raw.domains),
    message: asString(raw.message),
    suggestion: asString(raw.suggestion),
  });
}

/**
 * Parse the report text and return every finding across all phases. Throws
 * AuditReportParseError on malformed JSON or a shape that isn't a test-genie
 * report. An empty (but valid) report returns `[]`.
 *
 * Accepts both the canonical `{ phases: [{ findings: [...] }] }` shape and a
 * bare `{ findings: [...] }` for convenience when pasting a single phase.
 */
export function parseAuditReport(text: string): ArchitectureFinding[] {
  const trimmed = text.trim();
  if (trimmed.length === 0) {
    throw new AuditReportParseError("Paste a test-genie --json report first.");
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch (err) {
    throw new AuditReportParseError(
      `Not valid JSON: ${err instanceof Error ? err.message : String(err)}`,
    );
  }

  if (typeof parsed !== "object" || parsed === null) {
    throw new AuditReportParseError("Expected a test-genie --json report object.");
  }

  const report = parsed as { phases?: unknown; findings?: unknown };

  const out: ArchitectureFinding[] = [];
  if (Array.isArray(report.phases)) {
    for (const phase of report.phases) {
      if (!phase || typeof phase !== "object") continue;
      const findings = (phase as { findings?: unknown }).findings;
      if (Array.isArray(findings)) {
        for (const raw of findings) {
          if (raw && typeof raw === "object") out.push(toFinding(raw as RawFinding));
        }
      }
    }
    return out;
  }

  if (Array.isArray(report.findings)) {
    for (const raw of report.findings) {
      if (raw && typeof raw === "object") out.push(toFinding(raw as RawFinding));
    }
    return out;
  }

  throw new AuditReportParseError(
    "No `phases` or `findings` array found — is this a test-genie --json SuiteExecutionResult?",
  );
}
