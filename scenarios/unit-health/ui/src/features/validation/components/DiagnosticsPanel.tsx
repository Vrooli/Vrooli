import type { Diagnostic } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";
import type { QualityCheckResult } from "@vrooli/proto-types/unit-health/v1/validation/test_quality_pb";
import { useState } from "react";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Panel, Pill } from "./shared";
import { severityToneClass } from "./tone";

/**
 * DiagnosticsPanel lists workspace-level diagnostics (e.g. flake detection)
 * that are not normalized findings, each tagged with its severity and kind.
 */
export function DiagnosticsPanel({ diagnostics, nativeResults = [] }: { diagnostics: Diagnostic[]; nativeResults?: QualityCheckResult[] }) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(50);
  const observations = nativeResults.filter((row) => row.runtimeObservation !== undefined);

  return (
    <Panel
      title={t(strings.validation.diagnosticsTitle)}
      testId={selectors.validationWorkbench.diagnostics}
    >
      {diagnostics.length === 0 && observations.length === 0 ? (
        <p
          data-testid={selectors.validationWorkbench.diagnosticsEmpty}
          className="text-sm text-app-muted-foreground"
        >
          {t(strings.validation.diagnosticsEmpty)}
        </p>
      ) : (
        <div className="flex flex-col gap-2">
          {diagnostics.slice(0, visible).map((diagnostic, index) => (
            <article
              key={`${diagnostic.kind}-${diagnostic.workspaceId}-${index}`}
              className="rounded-control border border-app-border bg-app-surface p-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                <Pill tone={severityToneClass(diagnostic.severity)}>{diagnostic.severity}</Pill>
                <span className="text-xs font-medium text-app-muted-foreground">
                  {diagnostic.kind}
                </span>
                {diagnostic.workspaceId && (
                  <span className="text-xs text-app-muted-foreground">{diagnostic.workspaceId}</span>
                )}
              </div>
              <p className="mt-2 text-sm">{diagnostic.message}</p>
              {diagnostic.reliability && (
                <code className="mt-1 block text-xs text-app-muted-foreground">
                  {`reliability=${["unknown", "insufficient_samples", "comparable_observations", "suspected_instability"].includes(diagnostic.reliability.state) ? diagnostic.reliability.state : "unknown"} scope=${diagnostic.reliability.scope} samples=${diagnostic.reliability.sampleCount} passed=${diagnostic.reliability.passed} failed=${diagnostic.reliability.failed} excluded_infrastructure_or_unclassified=${diagnostic.reliability.excludedInfrastructure} excluded_incompatible=${diagnostic.reliability.excludedIncompatible}`}
                  {diagnostic.reliability.seed !== undefined && ` seed=${JSON.stringify(diagnostic.reliability.seed)}`}
                  {diagnostic.reliability.retryOrdinal !== undefined && ` retry_ordinal=${diagnostic.reliability.retryOrdinal}`}
                </code>
              )}
              {diagnostic.evidence && (
                <p className="mt-1 text-xs text-app-muted-foreground">{diagnostic.evidence}</p>
              )}
            </article>
          ))}
        </div>
      )}
      {observations.length > 0 && (
        <div className="mt-3 flex flex-col gap-2">
          <p className="text-sm">{t(strings.validation.nativeFinalWarning)}</p>
          {observations.slice(0, visible).map((row, index) => {
            const native = row.runtimeObservation!;
            const state = ["pass", "fail", "skip", "todo", "run", "only"].includes(native.state) ? native.state : "unknown";
            return <code key={`${row.target?.file}-${row.target?.testId}-${index}`} className="block break-words text-xs text-app-muted-foreground">
              {`${row.target?.workspace ?? ""}/${row.target?.file ?? ""}:${row.target?.testId ?? ""} final_state=${state} run=${native.runId}`}
              {native.seed !== undefined && ` seed=${JSON.stringify(native.seed)}`}
              {native.retryCount !== undefined && ` retry_count=${native.retryCount}`}
              {native.retryOrdinal !== undefined && ` retry_ordinal=${native.retryOrdinal}`}
            </code>;
          })}
        </div>
      )}
      {(diagnostics.length > 0 || observations.length > 0) && <p className="mt-2 text-xs">{t(strings.validation.qualityShowing, { shown: Math.min(visible, diagnostics.length) + Math.min(visible, observations.length), total: String(diagnostics.length + observations.length) })}</p>}
      {(visible < diagnostics.length || visible < observations.length) && <button type="button" className="mt-2 underline" onClick={() => setVisible((n) => n + 50)}>{t(strings.validation.qualityMore)}</button>}
    </Panel>
  );
}
