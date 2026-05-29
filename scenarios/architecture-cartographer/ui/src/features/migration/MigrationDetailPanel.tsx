import * as React from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { Textarea } from "../../components/ui/textarea";
import { Card, CardBody, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { SeverityBadge, type SeverityLevel } from "../../components/SeverityBadge";
import {
  useApplyFinding,
  useCloseMigration,
  useMigrationStatus,
  useNextStep,
  useReauditMigration,
  useResolveFinding,
} from "./controllers/useMigrationController";
import { severityTokenToLevel } from "./severity";
import { statusToState, type MigrationFindingState } from "./flow/transition";
import { AuditReportParseError, parseAuditReport } from "./lib/parseAuditReport";
import {
  MigrationLifecycle,
  type TrackedFinding,
} from "@vrooli/proto-types/architecture-cartographer/v1/migration/migration_pb";

const SEVERITY_LABEL_KEY = {
  info: strings.shared.severity.info,
  low: strings.shared.severity.low,
  medium: strings.shared.severity.medium,
  high: strings.shared.severity.high,
  critical: strings.shared.severity.critical,
} as const satisfies Record<SeverityLevel, string>;

const STATUS_LABEL_KEY = {
  detected: strings.migration.status.detected,
  assigned: strings.migration.status.assigned,
  split: strings.migration.status.split,
  resolved: strings.migration.status.resolved,
  validated: strings.migration.status.validated,
  committed: strings.migration.status.committed,
  force_resolved: strings.migration.status.force_resolved,
} as const satisfies Record<MigrationFindingState, string>;

function RollupStat({ label, value, danger = false }: { label: string; value: number; danger?: boolean }) {
  return (
    <div className="rounded-control border border-app-border bg-app-surface-muted p-2">
      <dt className="text-xs text-app-muted-foreground">{label}</dt>
      <dd className={`text-lg font-semibold ${danger ? "text-app-danger" : ""}`}>{value}</dd>
    </div>
  );
}

export interface MigrationDetailPanelProps {
  scenario: string;
  migrationId: string;
}

export function MigrationDetailPanel({ scenario, migrationId }: MigrationDetailPanelProps) {
  const { t } = useTranslation();
  const status = useMigrationStatus({ id: migrationId });
  const worklist = useNextStep({ id: migrationId });
  const resolve = useResolveFinding(migrationId);
  const apply = useApplyFinding(migrationId);
  const reaudit = useReauditMigration(migrationId);
  const close = useCloseMigration(migrationId, scenario);

  const [report, setReport] = React.useState("");
  const [reauditError, setReauditError] = React.useState<string | null>(null);

  if (status.isPending) {
    return (
      <div data-testid={selectors.features.migration.detail.loading}>
        <LoadingState label={t(strings.pages.migration.loading)} />
      </div>
    );
  }

  if (status.isError) {
    return (
      <div data-testid={selectors.features.migration.detail.error}>
        <ErrorState
          title={t(strings.pages.migration.errorTitle)}
          message={status.error instanceof Error ? status.error.message : String(status.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void status.refetch();
          }}
        />
      </div>
    );
  }

  const projection = status.data.status;
  const migration = projection?.migration;
  if (!projection || !migration) {
    return (
      <div data-testid={selectors.features.migration.detail.notFound}>
        <EmptyState title={t(strings.pages.migration.detail.notFound)} />
      </div>
    );
  }

  const isOpen = migration.status !== MigrationLifecycle.CLOSED;
  const pending = resolve.isPending || apply.isPending || close.isPending;

  const onResolve = (finding: TrackedFinding) => {
    const note = window.prompt(t(strings.migration.actions.resolveNotePrompt)) ?? "";
    resolve.mutate({ stableId: finding.stableId, note: note.trim() });
  };

  const onReaudit = () => {
    setReauditError(null);
    let findings;
    try {
      findings = parseAuditReport(report);
    } catch (err) {
      setReauditError(err instanceof AuditReportParseError ? err.message : String(err));
      return;
    }
    reaudit.mutate(findings, { onSuccess: () => setReport("") });
  };

  const findings = worklist.data?.findings ?? [];

  return (
    <article
      data-testid={selectors.features.migration.detail.root}
      aria-labelledby={`migration-detail-${migration.id}`}
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-baseline gap-2">
          <h3 id={`migration-detail-${migration.id}`} className="text-lg font-semibold">
            {migration.name || t(strings.pages.migration.unnamed)}
          </h3>
          <Badge variant={isOpen ? "info" : "default"}>
            {t(isOpen ? strings.migration.lifecycle.open : strings.migration.lifecycle.closed)}
          </Badge>
        </div>
        <p className="text-xs font-mono text-app-muted-foreground">{migration.id}</p>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.migration.detail.scenarioLabel)}{" "}
          <span className="font-mono text-app-foreground">{migration.scenario}</span>
        </p>
      </header>

      <dl
        data-testid={selectors.features.migration.detail.rollup}
        className="grid grid-cols-2 gap-2 sm:grid-cols-5"
      >
        <RollupStat label={t(strings.pages.migration.detail.rollup.total)} value={projection.total} />
        <RollupStat label={t(strings.pages.migration.detail.rollup.open)} value={projection.open} />
        <RollupStat label={t(strings.pages.migration.detail.rollup.resolved)} value={projection.resolved} />
        <RollupStat label={t(strings.pages.migration.detail.rollup.validated)} value={projection.validated} />
        <RollupStat
          label={t(strings.pages.migration.detail.rollup.regressions)}
          value={projection.regressions}
          danger={projection.regressions > 0}
        />
      </dl>

      <Card>
        <CardHeader>
          <CardTitle>{t(strings.pages.migration.detail.worklistHeading)}</CardTitle>
        </CardHeader>
        <CardBody data-testid={selectors.features.migration.detail.worklist}>
          {findings.length === 0 ? (
            <div data-testid={selectors.features.migration.detail.worklistEmpty}>
              <EmptyState title={t(strings.pages.migration.detail.worklistEmpty)} />
            </div>
          ) : (
            <ul className="flex flex-col gap-3">
              {findings.map((finding) => {
                const level = severityTokenToLevel(finding.severity);
                return (
                  <li
                    key={finding.stableId}
                    data-testid={selectors.features.migration.detail.findingCard({ stableId: finding.stableId })}
                    className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-2"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs text-app-muted-foreground">{finding.source}</span>
                      <span className="text-sm font-medium">{finding.code}</span>
                      <SeverityBadge level={level} label={t(SEVERITY_LABEL_KEY[level])} />
                      <Badge variant="outline">{t(STATUS_LABEL_KEY[statusToState(finding.status)])}</Badge>
                      {finding.regressed ? (
                        <Badge variant="danger">{t(strings.pages.migration.detail.regressedBadge)}</Badge>
                      ) : null}
                    </div>
                    {finding.message ? <p className="text-sm">{finding.message}</p> : null}
                    {finding.locations.length > 0 ? (
                      <p className="font-mono text-xs text-app-muted-foreground">
                        <span className="not-italic">{t(strings.pages.migration.detail.locationsLabel)}:</span>{" "}
                        {finding.locations.join(", ")}
                      </p>
                    ) : null}
                    {finding.suggestion ? (
                      <p className="text-xs text-app-muted-foreground">
                        {t(strings.pages.migration.detail.suggestionLabel)} {finding.suggestion}
                      </p>
                    ) : null}
                    {isOpen ? (
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          variant="default"
                          size="sm"
                          disabled={pending}
                          data-testid={selectors.features.migration.detail.actionButton({
                            stableId: finding.stableId,
                            action: "resolve",
                          })}
                          onClick={() => onResolve(finding)}
                        >
                          {pending ? t(strings.migration.actions.pending) : t(strings.migration.actions.resolve)}
                        </Button>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          disabled={pending}
                          data-testid={selectors.features.migration.detail.actionButton({
                            stableId: finding.stableId,
                            action: "apply",
                          })}
                          onClick={() => apply.mutate(finding.stableId)}
                        >
                          {pending ? t(strings.migration.actions.pending) : t(strings.migration.actions.apply)}
                        </Button>
                      </div>
                    ) : null}
                  </li>
                );
              })}
            </ul>
          )}
        </CardBody>
      </Card>

      {isOpen ? (
        <Card>
          <CardHeader>
            <CardTitle>{t(strings.pages.migration.detail.reauditHeading)}</CardTitle>
          </CardHeader>
          <CardBody data-testid={selectors.features.migration.detail.reaudit} className="flex flex-col gap-2">
            <p className="text-xs text-app-muted-foreground">
              {t(strings.pages.migration.detail.reauditHint, { scenario })}
            </p>
            <Textarea
              data-testid={selectors.features.migration.detail.reauditInput}
              value={report}
              onChange={(e) => setReport(e.target.value)}
              placeholder={t(strings.pages.migration.create.reportPlaceholder)}
              rows={4}
              className="font-mono text-xs"
            />
            {reauditError ? (
              <ErrorState title={t(strings.pages.migration.create.parseErrorTitle)} message={reauditError} />
            ) : null}
            {reaudit.isSuccess ? (
              <p
                data-testid={selectors.features.migration.detail.reauditResult}
                className="text-sm text-app-foreground"
              >
                {t(strings.pages.migration.detail.reauditResult, {
                  validated: reaudit.data.validated.length,
                  open: reaudit.data.stillOpen.length,
                  regressions: reaudit.data.regressions.length,
                })}
              </p>
            ) : null}
            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="default"
                size="sm"
                data-testid={selectors.features.migration.detail.reauditSubmit}
                onClick={onReaudit}
                disabled={reaudit.isPending || report.trim().length === 0}
              >
                {reaudit.isPending
                  ? t(strings.pages.migration.detail.reauditting)
                  : t(strings.pages.migration.detail.reauditSubmit)}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                data-testid={selectors.features.migration.detail.closeButton}
                onClick={() => close.mutate()}
                disabled={close.isPending}
              >
                {close.isPending
                  ? t(strings.pages.migration.detail.closing)
                  : t(strings.pages.migration.detail.closeButton)}
              </Button>
            </div>
          </CardBody>
        </Card>
      ) : (
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.migration.detail.closedNote)}</p>
      )}
    </article>
  );
}
