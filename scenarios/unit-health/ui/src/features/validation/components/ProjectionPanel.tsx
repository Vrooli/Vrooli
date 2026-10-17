import type { ProjectionCheck } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Panel, Pill } from "./shared";
import { normalize, shortPath } from "./tone";

const statusToneClass = (status: string) => {
  switch (normalize(status)) {
    case "pass":
      return "border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";
    case "drift":
    case "missing":
      return "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300";
    default:
      return "border-app-border bg-app-surface-muted text-app-muted-foreground";
  }
};

/**
 * ProjectionPanel renders policy-vs-native unit infrastructure checks. Unlike
 * findings, passing rows are intentionally visible so operators can inspect
 * which unit policy requirements are actually projected into native files.
 */
export function ProjectionPanel({ checks }: { checks: ProjectionCheck[] }) {
  const { t } = useTranslation();

  return (
    <Panel
      title={t(strings.validation.projectionTitle)}
      testId={selectors.validationWorkbench.projections}
    >
      {checks.length === 0 ? (
        <p
          data-testid={selectors.validationWorkbench.projectionsEmpty}
          className="text-sm text-app-muted-foreground"
        >
          {t(strings.validation.projectionEmpty)}
        </p>
      ) : (
        <div className="grid gap-2 lg:grid-cols-2">
          {checks.map((check) => (
            <article
              key={check.id}
              data-testid={selectors.validationWorkbench.projectionRow({ id: check.id })}
              className="rounded-control border border-app-border bg-app-surface p-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                <Pill tone={statusToneClass(check.status)}>{check.status || "unknown"}</Pill>
                <span className="font-mono text-xs font-semibold text-app-muted-foreground">
                  {check.workspaceId}
                </span>
                <span className="text-xs font-medium text-app-muted-foreground">
                  {check.key}
                </span>
              </div>
              <p className="mt-2 text-xs text-app-muted-foreground">{check.owner}</p>
              <p className="mt-1 font-mono text-xs text-app-muted-foreground">
                {shortPath(check.filePath)}
              </p>
              <dl className="mt-2 grid gap-1 text-xs sm:grid-cols-2">
                <div>
                  <dt className="font-semibold">{t(strings.validation.projectionPolicy)}</dt>
                  <dd className="text-app-muted-foreground">{check.policyValue || "<empty>"}</dd>
                </div>
                <div>
                  <dt className="font-semibold">{t(strings.validation.projectionNative)}</dt>
                  <dd className="text-app-muted-foreground">{check.nativeValue || "<empty>"}</dd>
                </div>
              </dl>
              {check.remediation && (
                <p className="mt-2 text-xs text-app-muted-foreground">{check.remediation}</p>
              )}
            </article>
          ))}
        </div>
      )}
    </Panel>
  );
}
