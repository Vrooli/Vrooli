import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Card, CardBody, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { SeverityBadge } from "../../components/SeverityBadge";
import { useGetConflict } from "./controllers/useConflictsController";
import { severityToLevel } from "./severity";
import {
  FixKind,
  type Fix,
} from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

const SEVERITY_LABEL_KEY = {
  info: strings.shared.severity.info,
  low: strings.shared.severity.low,
  medium: strings.shared.severity.medium,
  high: strings.shared.severity.high,
  critical: strings.shared.severity.critical,
} as const;

const FIX_KIND_LABEL_KEY = {
  [FixKind.UNSPECIFIED]: strings.conflicts.fixKind.unspecified,
  [FixKind.MOVE_FILE]: strings.conflicts.fixKind.move_file,
  [FixKind.REASSIGN_DOMAIN]: strings.conflicts.fixKind.reassign_domain,
  [FixKind.BREAK_CYCLE]: strings.conflicts.fixKind.break_cycle,
  [FixKind.ADD_DEPENDENCY]: strings.conflicts.fixKind.add_dependency,
  [FixKind.ADD_TRANSITIONAL]: strings.conflicts.fixKind.add_transitional,
} as const;

function fixKindLabelKey(kind: Fix["kind"]) {
  return FIX_KIND_LABEL_KEY[kind];
}

export interface ConflictDetailPanelProps {
  scenario: string;
  conflictId: string;
}

/**
 * Read-only conflict detail. The conflicts domain is detection-only — this
 * panel shows what's wrong now (evidence + suggested fixes). Walking a finding
 * through a lifecycle (resolve / validate) lives in the campaign feature.
 */
export function ConflictDetailPanel({ conflictId }: ConflictDetailPanelProps) {
  const { t } = useTranslation();
  const detail = useGetConflict({ id: conflictId });
  const conflict = detail.data?.conflict;

  if (detail.isPending) {
    return (
      <div data-testid={selectors.features.conflicts.detail.loading}>
        <LoadingState label={t(strings.pages.conflicts.loading)} />
      </div>
    );
  }

  if (detail.isError) {
    return (
      <ErrorState
        title={t(strings.pages.conflicts.errorTitle)}
        message={detail.error instanceof Error ? detail.error.message : String(detail.error)}
        retryLabel={t(strings.shared.error.retry)}
        onRetry={() => {
          void detail.refetch();
        }}
      />
    );
  }

  if (!conflict) {
    return (
      <div data-testid={selectors.features.conflicts.detail.notFound}>
        <EmptyState title={t(strings.pages.conflicts.detailNotFound)} />
      </div>
    );
  }

  const severity = severityToLevel(conflict.severity);

  return (
    <article
      data-testid={selectors.features.conflicts.detail.root}
      aria-labelledby={`conflict-detail-${conflict.id}`}
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-baseline gap-2">
          <h3 id={`conflict-detail-${conflict.id}`} className="text-lg font-semibold">
            {conflict.type}
          </h3>
          <SeverityBadge level={severity} label={t(SEVERITY_LABEL_KEY[severity])} />
        </div>
        <p className="text-xs font-mono text-app-muted-foreground">{conflict.id}</p>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.conflicts.snapshotLabel)}{" "}
          <span className="font-mono">{conflict.snapshotId || "—"}</span>
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>{t(strings.pages.conflicts.locationsHeading)}</CardTitle>
        </CardHeader>
        <CardBody data-testid={selectors.features.conflicts.detail.locations}>
          {conflict.locations.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">—</p>
          ) : (
            <ul className="flex flex-col gap-1">
              {conflict.locations.map((loc) => (
                <li key={loc} className="font-mono text-xs">
                  {loc}
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t(strings.pages.conflicts.evidenceHeading)}</CardTitle>
        </CardHeader>
        <CardBody data-testid={selectors.features.conflicts.detail.evidence}>
          {conflict.evidence.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">
              {t(strings.pages.conflicts.noEvidence)}
            </p>
          ) : (
            <ul className="flex flex-col gap-3">
              {conflict.evidence.map((item, index) => (
                <li
                  key={`${item.kind}-${index}`}
                  data-testid={selectors.features.conflicts.detail.evidenceItem({ index })}
                  className="rounded-control border border-app-border bg-app-surface-muted p-2"
                >
                  <p className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
                    {item.kind}
                  </p>
                  {item.summary ? <p className="text-sm">{item.summary}</p> : null}
                  {item.locator ? (
                    <p className="font-mono text-xs text-app-muted-foreground">{item.locator}</p>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t(strings.pages.conflicts.fixesHeading)}</CardTitle>
        </CardHeader>
        <CardBody data-testid={selectors.features.conflicts.detail.fixes}>
          {conflict.suggestedFixes.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.conflicts.noFixes)}</p>
          ) : (
            <ul className="flex flex-col gap-3">
              {conflict.suggestedFixes.map((fix) => (
                <li
                  key={fix.id}
                  data-testid={selectors.features.conflicts.detail.fixItem({ id: fix.id })}
                  className="rounded-control border border-app-border bg-app-surface-muted p-2"
                >
                  <p className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
                    {t(fixKindLabelKey(fix.kind))}
                  </p>
                  <p className="text-sm">{fix.summary}</p>
                  {fix.resolver ? (
                    <p className="font-mono text-xs text-app-muted-foreground">{fix.resolver}</p>
                  ) : null}
                  {fix.confidence > 0 ? (
                    <p className="text-xs text-app-muted-foreground">
                      {t(strings.pages.conflicts.confidenceLabel, {
                        value: fix.confidence.toFixed(2),
                      })}
                    </p>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>
    </article>
  );
}
