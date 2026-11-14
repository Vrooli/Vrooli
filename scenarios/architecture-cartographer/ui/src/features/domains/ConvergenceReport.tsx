import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { SeverityBadge, type SeverityLevel } from "../../components/SeverityBadge";
import type { ConvergenceFinding } from "@vrooli/proto-types/architecture-cartographer/v1/domains/domains_pb";
import { ConvergenceSeverity } from "@vrooli/proto-types/architecture-cartographer/v1/domains/domains_pb";
import { useConvergenceReport } from "./controllers/useDomainsController";

export interface ConvergenceReportProps {
  scenario: string;
}

/**
 * Map a convergence severity to the shared SeverityBadge level. WARN maps to
 * the "medium" warning style and INFO to the "info" style; both render with a
 * text label so severity is never conveyed by color alone.
 */
function severityLevel(severity: ConvergenceSeverity): SeverityLevel {
  return severity === ConvergenceSeverity.WARN ? "medium" : "info";
}

/**
 * i18n key path for a finding's severity label, keyed by proto severity.
 * Declared `as const` so the value stays a literal key the typed `t()`
 * accepts (mirrors the conflicts feature's SEVERITY_LABEL_KEY pattern).
 */
const SEVERITY_LABEL_KEY = {
  warn: strings.pages.targetDomains.convergence.severityWarn,
  info: strings.pages.targetDomains.convergence.severityInfo,
} as const;

function severityLabelKey(severity: ConvergenceSeverity) {
  return severity === ConvergenceSeverity.WARN
    ? SEVERITY_LABEL_KEY.warn
    : SEVERITY_LABEL_KEY.info;
}

export function ConvergenceReport({ scenario }: ConvergenceReportProps) {
  const { t } = useTranslation();
  const report = useConvergenceReport(scenario);

  const heading = (
    <h4 id="domains-convergence-heading" className="text-lg font-semibold">
      {t(strings.pages.targetDomains.convergence.heading)}
    </h4>
  );

  let body: React.ReactNode;
  if (report.isPending) {
    body = (
      <div data-testid={selectors.features.domains.convergence.loading}>
        <LoadingState label={t(strings.pages.targetDomains.convergence.loading)} />
      </div>
    );
  } else if (report.isError) {
    body = (
      <div data-testid={selectors.features.domains.convergence.error}>
        <ErrorState
          title={t(strings.pages.targetDomains.convergence.errorTitle)}
          message={report.error instanceof Error ? report.error.message : String(report.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void report.refetch();
          }}
        />
      </div>
    );
  } else {
    const findings: readonly ConvergenceFinding[] = report.data.findings;
    body =
      findings.length === 0 ? (
        <div data-testid={selectors.features.domains.convergence.converged}>
          <EmptyState title={t(strings.pages.targetDomains.convergence.converged)} />
        </div>
      ) : (
        <ul className="flex flex-col gap-2">
          {findings.map((finding, index) => (
            <li
              key={`${finding.kind}-${finding.domain}-${index}`}
              data-testid={selectors.features.domains.convergence.finding({ index })}
              className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface p-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                <SeverityBadge
                  level={severityLevel(finding.severity)}
                  label={t(severityLabelKey(finding.severity))}
                />
                <span className="font-semibold">{finding.domain}</span>
                <span className="font-mono text-xs text-app-muted-foreground">{finding.kind}</span>
              </div>
              <p className="text-sm text-app-foreground">{finding.message}</p>
            </li>
          ))}
        </ul>
      );
  }

  return (
    <section
      aria-labelledby="domains-convergence-heading"
      data-testid={selectors.features.domains.convergence.root}
      className="flex flex-col gap-2"
    >
      {heading}
      {body}
    </section>
  );
}
