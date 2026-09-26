import { useState } from "react";
import { QualityCheckStatus, QualityReason, QualityEvidenceKind, QualitySeverity, QualityEnforcement, type TestQualityReport } from "@vrooli/proto-types/unit-health/v1/validation/test_quality_pb";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";

export function qualityStatus(status: QualityCheckStatus): string {
  switch (status) {
    case QualityCheckStatus.CHECKED_CLEAN: return "checked_clean";
    case QualityCheckStatus.VIOLATION: return "violation";
    case QualityCheckStatus.NOT_APPLICABLE: return "not_applicable";
    default: return "unknown";
  }
}

/** Assessment counts measure analysis support, never behavioral coverage. */
export function QualityPanel({ report }: { report?: TestQualityReport }) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(50);
  const supported = report?.schemaVersion === "test-quality/v1" && report.catalogVersion !== "";
  const unavailable = !supported || (report!.unavailableReason !== QualityReason.UNSPECIFIED && report!.unavailableReason !== QualityReason.NONE);
  const rows = report?.results ?? [];
  const total = supported && !unavailable && report!.totalResults >= BigInt(rows.length) ? String(report!.totalResults) : "unknown";
  return <section aria-label={t(strings.validation.qualityTitle)} className="rounded-panel border border-app-border bg-app-surface p-4">
    <h3 className="text-sm font-semibold">{t(strings.validation.qualityTitle)}</h3>
    <p className="mt-2 text-sm">{t(strings.validation.qualityLimit)}</p>
    {(unavailable || !report?.coverage.length) && <p className="mt-2 text-sm">{t(strings.validation.qualityUnknown)}</p>}
    {report && <code className="block text-xs">{`reason=${QualityReason[report.unavailableReason] ?? "UNRECOGNIZED_VALUE"}`}</code>}
    {report?.reasonGuidance && <p className="mt-2 text-sm">{report.reasonGuidance}</p>}
    {report?.collectionLimitations.map((limitation, i) => <div className="mt-2 text-sm" key={i}>
      <p>{t(strings.validation.partialCollectionLimit)}</p>
      <code>{QualityReason[limitation.reason] ?? "UNRECOGNIZED_VALUE"}</code>
      <p>{limitation.guidance}</p>
    </div>)}
    {!unavailable && report?.coverage.map((c, i) => {
      const valid = c.assessed <= c.discovered && c.unknown <= c.discovered - c.assessed && c.notApplicable === c.discovered - c.assessed - c.unknown;
      return <code className="mt-2 block text-xs" key={`${c.ruleId}-${c.supportProfile}-${i}`}>
        {`${c.ruleId}/${c.supportProfile} scope=${c.scope || "test"} `}
        {valid ? `assessed=${c.assessed}/discovered=${c.discovered} unknown=${c.unknown} not_applicable=${c.notApplicable}` : "assessment_coverage=unknown (inconsistent denominator)"}
      </code>;
    })}
    {rows.slice(0, visible).map((row, index) => <details key={`${row.ruleId}-${row.target?.file}-${row.target?.testId}-${index}`} className="mt-3 rounded-control border border-app-border p-2">
      <summary className="cursor-pointer text-sm">{`${row.ruleId} [${unavailable ? "unknown" : qualityStatus(row.status)}] ${row.target?.file ?? ""}:${row.location?.line ?? "?"} ${row.target?.scope === "file" ? "[file scope]" : row.target?.testId ?? ""}`}</summary>
      <pre className="mt-2 overflow-auto whitespace-pre-wrap text-xs">{JSON.stringify({
        reason: QualityReason[row.reason] ?? "UNRECOGNIZED_VALUE", evidenceKind: QualityEvidenceKind[row.evidenceKind] ?? "UNKNOWN",
        reasonGuidance: row.reasonGuidance,
        severity: QualitySeverity[row.severity] ?? "UNKNOWN", enforcement: QualityEnforcement[row.enforcement] ?? "UNKNOWN", supportProfile: row.supportProfile,
        location: row.location, nativeDiagnostics: row.diagnostics, limitations: row.limitations, evidenceRefs: row.evidenceRefs,
      }, null, 2)}</pre>
    </details>)}
    <p className="mt-2 text-xs">{t(strings.validation.qualityShowing, { shown: Math.min(visible, rows.length), total })}</p>
    {visible < rows.length && <button type="button" className="mt-2 underline" onClick={() => setVisible((n) => n + 50)}>{t(strings.validation.qualityMore)}</button>}
    {report?.truncated && <p className="mt-2 break-words text-xs">{t(strings.validation.qualityDetails)} <code>{report.detailsRef || "unknown"}</code></p>}
  </section>;
}
