import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Badge } from "../../components/ui/badge";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { SeverityBadge, type SeverityLevel } from "../../components/SeverityBadge";
import type { DomainCoupling } from "@vrooli/proto-types/architecture-cartographer/v1/signals/signals_pb";
import { CouplingSeverity } from "@vrooli/proto-types/architecture-cartographer/v1/signals/signals_pb";
import { useBoundaryHealth } from "./controllers/useDomainsController";

export interface BoundaryHealthProps {
  scenario: string;
}

/**
 * i18n key path for a smell's severity label, keyed by proto severity.
 * Declared `as const` so the value stays a literal key the typed `t()`
 * accepts (mirrors the conflicts feature's SEVERITY_LABEL_KEY pattern).
 */
const SMELL_LABEL_KEY = {
  warn: strings.pages.targetDomains.boundaries.severityWarn,
  info: strings.pages.targetDomains.boundaries.severityInfo,
} as const;

function smellLabelKey(severity: CouplingSeverity) {
  return severity === CouplingSeverity.WARN ? SMELL_LABEL_KEY.warn : SMELL_LABEL_KEY.info;
}

function smellLevel(severity: CouplingSeverity): SeverityLevel {
  return severity === CouplingSeverity.WARN ? "medium" : "info";
}

/** Health-score badge tone: >=0.8 good, >=0.5 medium, else warning. */
function healthVariant(score: number): "success" | "default" | "warning" {
  if (score >= 0.8) return "success";
  if (score >= 0.5) return "default";
  return "warning";
}

export function BoundaryHealth({ scenario }: BoundaryHealthProps) {
  const { t } = useTranslation();
  const report = useBoundaryHealth(scenario);

  const heading = (
    <h4 id="domains-boundaries-heading" className="text-lg font-semibold">
      {t(strings.pages.targetDomains.boundaries.heading)}
    </h4>
  );

  let body: React.ReactNode;
  if (report.isPending) {
    body = (
      <div data-testid={selectors.features.domains.boundaries.loading}>
        <LoadingState label={t(strings.pages.targetDomains.boundaries.loading)} />
      </div>
    );
  } else if (report.isError) {
    body = (
      <div data-testid={selectors.features.domains.boundaries.error}>
        <ErrorState
          title={t(strings.pages.targetDomains.boundaries.errorTitle)}
          message={report.error instanceof Error ? report.error.message : String(report.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void report.refetch();
          }}
        />
      </div>
    );
  } else {
    const domains: readonly DomainCoupling[] = report.data.domains;
    body =
      domains.length === 0 ? (
        <div data-testid={selectors.features.domains.boundaries.empty}>
          <EmptyState title={t(strings.pages.targetDomains.boundaries.empty)} />
        </div>
      ) : (
        <ul className="flex flex-col gap-2">
          {domains.map((coupling) => (
            <li
              key={coupling.domain}
              data-testid={selectors.features.domains.boundaries.row({ domain: coupling.domain })}
              className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-semibold">{coupling.domain}</span>
                {coupling.archetype && (
                  <span className="font-mono text-xs text-app-muted-foreground">
                    {coupling.archetype}
                  </span>
                )}
                <Badge variant={healthVariant(coupling.healthScore)}>
                  {`${t(strings.pages.targetDomains.boundaries.healthScore)} ${coupling.healthScore.toFixed(2)}`}
                </Badge>
                {coupling.stableKernel && (
                  <Badge variant="outline">
                    {t(strings.pages.targetDomains.boundaries.stableKernel)}
                  </Badge>
                )}
              </div>

              <dl className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-app-muted-foreground">
                <div className="flex gap-1">
                  <dt className="font-medium">{t(strings.pages.targetDomains.boundaries.efferent)}</dt>
                  <dd>{coupling.efferent}</dd>
                </div>
                <div className="flex gap-1">
                  <dt className="font-medium">{t(strings.pages.targetDomains.boundaries.afferent)}</dt>
                  <dd>{coupling.afferent}</dd>
                </div>
                <div className="flex gap-1">
                  <dt className="font-medium">{t(strings.pages.targetDomains.boundaries.instability)}</dt>
                  <dd>{coupling.instability.toFixed(2)}</dd>
                </div>
                <div className="flex gap-1">
                  <dt className="font-medium">{t(strings.pages.targetDomains.boundaries.fanOut)}</dt>
                  <dd>{coupling.fanOut.toFixed(2)}</dd>
                </div>
              </dl>

              {coupling.smells.length > 0 && (
                <ul className="flex flex-col gap-1">
                  {coupling.smells.map((smell, index) => (
                    <li
                      key={`${smell.kind}-${index}`}
                      className="flex flex-wrap items-center gap-2 text-sm"
                    >
                      <SeverityBadge
                        level={smellLevel(smell.severity)}
                        label={t(smellLabelKey(smell.severity))}
                      />
                      <span className="font-mono text-xs text-app-muted-foreground">{smell.kind}</span>
                      <span className="text-app-foreground">{smell.message}</span>
                    </li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ul>
      );
  }

  return (
    <section
      aria-labelledby="domains-boundaries-heading"
      data-testid={selectors.features.domains.boundaries.root}
      className="flex flex-col gap-2"
    >
      {heading}
      {body}
    </section>
  );
}
