import { useState } from "react";
import { QualityReason, RequirementRegistration, RequirementExecutionState, RequirementApplicability, type RequirementTraceabilityReport } from "@vrooli/proto-types/unit-health/v1/validation/test_quality_pb";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";

const unavailable = (reason: QualityReason) => reason !== QualityReason.NONE && reason !== QualityReason.UNSPECIFIED;
const executionLabels: Readonly<Record<number, string>> = {
  [RequirementExecutionState.PASSED]: "passed", [RequirementExecutionState.FAILED]: "failed",
  [RequirementExecutionState.SKIPPED]: "skipped", [RequirementExecutionState.NOT_RUN]: "not_run",
};

export function TraceabilityPanel({ report }: { report?: RequirementTraceabilityReport }) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(50);
  const supported = report?.schemaVersion === "requirement-traceability/v1";
  const registryKnown = supported && !unavailable(report!.unavailableReason);
  const links = report?.links ?? [];
  return <section aria-label={t(strings.validation.traceabilityTitle)} className="rounded-panel border border-app-border bg-app-surface p-4">
    <h3 className="text-sm font-semibold">{t(strings.validation.traceabilityTitle)}</h3>
    <p className="mt-2 text-sm">{t(strings.validation.traceabilityLimit)}</p>
    {!supported && <p className="mt-2">{t(strings.validation.traceabilityUnknown)}</p>}
    {report && <code className="mt-2 block text-xs">{`registry=${registryKnown ? "available" : "unknown"} reason=${QualityReason[report.unavailableReason] ?? "UNRECOGNIZED_VALUE"}; evidence=${supported && !unavailable(report.evidenceUnavailableReason) ? "supplied" : "unknown"} reason=${QualityReason[report.evidenceUnavailableReason] ?? "UNRECOGNIZED_VALUE"}`}</code>}
    {registryKnown && <details className="mt-2"><summary className="cursor-pointer">{t(strings.validation.traceabilityResponsibilities)}</summary>
      {report!.requirements.map((scope, i) => <code key={`${scope.requirementId}-${i}`} className="block text-xs">{`${scope.requirementId} unit_responsibility=${scope.applicability === RequirementApplicability.APPLICABLE ? "applicable" : scope.applicability === RequirementApplicability.NOT_APPLICABLE ? "not_applicable" : "unknown"}`}</code>)}
    </details>}
    {links.slice(0, visible).map((link, index) => {
      const registration = registryKnown && link.registration === RequirementRegistration.REGISTERED ? "registered" : registryKnown && link.registration === RequirementRegistration.STALE ? "stale" : "unknown";
      const execution = supported ? (executionLabels[link.execution] ?? "unknown") : "unknown";
      return <details key={`${link.requirementId}-${index}`} className="mt-2 rounded-control border border-app-border p-2">
        <summary className="cursor-pointer text-sm">{`${link.requirementId} [${registration}] execution=${execution}`}</summary>
        <code className="block break-words text-xs">{`${link.target?.workspace ?? ""}/${link.target?.file ?? ""}:${link.target?.testId ?? ""} run=${link.runId} reason=${QualityReason[link.reason] ?? "UNRECOGNIZED_VALUE"}`}</code>
      </details>;
    })}
    <p className="mt-2 text-xs">{t(strings.validation.qualityShowing, { shown: Math.min(visible, links.length), total: supported ? links.length : "unknown" })}</p>
    {visible < links.length && <button type="button" className="mt-2 underline" onClick={() => setVisible((n) => n + 50)}>{t(strings.validation.qualityMore)}</button>}
    {report?.limitations.map((limitation, i) => <p key={i} className="mt-2 text-xs">{limitation}</p>)}
  </section>;
}
