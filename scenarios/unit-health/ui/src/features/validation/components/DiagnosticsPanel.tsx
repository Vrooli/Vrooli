import type { Diagnostic } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Panel, Pill } from "./shared";
import { severityToneClass } from "./tone";

/**
 * DiagnosticsPanel lists workspace-level diagnostics (e.g. flake detection)
 * that are not normalized findings, each tagged with its severity and kind.
 */
export function DiagnosticsPanel({ diagnostics }: { diagnostics: Diagnostic[] }) {
  const { t } = useTranslation();

  return (
    <Panel
      title={t(strings.validation.diagnosticsTitle)}
      testId={selectors.validationWorkbench.diagnostics}
    >
      {diagnostics.length === 0 ? (
        <p
          data-testid={selectors.validationWorkbench.diagnosticsEmpty}
          className="text-sm text-app-muted-foreground"
        >
          {t(strings.validation.diagnosticsEmpty)}
        </p>
      ) : (
        <div className="flex flex-col gap-2">
          {diagnostics.map((diagnostic, index) => (
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
              {diagnostic.evidence && (
                <p className="mt-1 text-xs text-app-muted-foreground">{diagnostic.evidence}</p>
              )}
            </article>
          ))}
        </div>
      )}
    </Panel>
  );
}
