import type { ValidationFinding } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";
import { useState } from "react";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Panel, Pill } from "./shared";
import { severityToneClass, shortPath } from "./tone";

/**
 * FindingsPanel groups architecture/quality findings by category, then renders
 * each finding's severity, code, file, message, evidence, and the
 * expected/observed/why-it-matters/remediation context when present.
 */
export function FindingsPanel({ findings, suppressed = false }: { findings: ValidationFinding[]; suppressed?: boolean }) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(50);

  const byCategory = new Map<string, ValidationFinding[]>();
  for (const finding of findings.slice(0, visible)) {
    const category = finding.category || "other";
    const list = byCategory.get(category) ?? [];
    list.push(finding);
    byCategory.set(category, list);
  }

  return (
    <Panel title={t(suppressed ? strings.validation.suppressedTitle : strings.validation.findingsTitle)} testId={suppressed ? "suppressed-findings" : selectors.validationWorkbench.findings}>
      {suppressed && <p className="mb-2 text-sm">{t(strings.validation.suppressedLimit)}</p>}
      <p className="mb-2 text-xs">{t(strings.validation.qualityShowing, { shown: Math.min(visible, findings.length), total: String(findings.length) })}</p>
      {findings.length === 0 ? (!suppressed && (
        <p
          data-testid={suppressed ? "suppressed-findings-empty" : selectors.validationWorkbench.empty}
          className="text-sm text-app-muted-foreground"
        >
          {t(strings.validation.noFindings)}
        </p>
      )) : (
        <div className="flex flex-col gap-4">
          {[...byCategory.entries()].map(([category, group]) => (
            <div
              key={category}
              data-testid={selectors.validationWorkbench.findingCategory({ category })}
            >
              <p className="text-xs font-semibold uppercase text-app-muted-foreground">{category}</p>
              <div className="mt-2 flex flex-col gap-2">
                {group.map((finding) => (
                  <article
                    key={finding.id}
                    data-testid={selectors.validationWorkbench.findingRow({ id: finding.id })}
                    className="rounded-control border border-app-border bg-app-surface p-3"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <Pill tone={severityToneClass(finding.severity)}>{finding.severity}</Pill>
                      <span className="text-xs font-medium text-app-muted-foreground">
                        {finding.code}
                      </span>
                      <span className="text-xs text-app-muted-foreground">
                        {shortPath(finding.filePath)}
                      </span>
                    </div>
                    <p className="mt-2 text-sm font-medium">{finding.message}</p>
                    {finding.evidence && (
                      <p className="mt-1 break-words text-xs text-app-muted-foreground">
                        {finding.evidence}
                      </p>
                    )}
                    <dl className="mt-2 grid gap-1 text-xs sm:grid-cols-2">
                      {finding.expected && (
                        <div>
                          <dt className="font-semibold">{t(strings.validation.findingExpected)}</dt>
                          <dd className="text-app-muted-foreground">{finding.expected}</dd>
                        </div>
                      )}
                      {finding.observed && (
                        <div>
                          <dt className="font-semibold">{t(strings.validation.findingObserved)}</dt>
                          <dd className="text-app-muted-foreground">{finding.observed}</dd>
                        </div>
                      )}
                      {finding.whyItMatters && (
                        <div>
                          <dt className="font-semibold">{t(strings.validation.findingWhy)}</dt>
                          <dd className="text-app-muted-foreground">{finding.whyItMatters}</dd>
                        </div>
                      )}
                      {finding.remediation && (
                        <div>
                          <dt className="font-semibold">{t(strings.validation.findingRemediation)}</dt>
                          <dd className="text-app-muted-foreground">{finding.remediation}</dd>
                        </div>
                      )}
                    </dl>
                    {suppressed && <details className="mt-2 text-xs">
                      <summary className="cursor-pointer">{t(strings.validation.suppressedDetails)}</summary>
                      <pre className="whitespace-pre-wrap break-words">{finding.suppressionReasons?.length ? JSON.stringify(finding.suppressionReasons, null, 2) : t(strings.validation.suppressedUnknown)}</pre>
                    </details>}
                  </article>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
      {visible < findings.length && <button type="button" className="mt-2 underline" onClick={() => setVisible((n) => n + 50)}>{t(strings.validation.qualityMore)}</button>}
    </Panel>
  );
}
