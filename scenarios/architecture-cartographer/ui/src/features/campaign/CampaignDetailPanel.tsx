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
  RankProfile,
  useApplyItem,
  useCloseCampaign,
  useCampaignStatus,
  useNextStep,
  useReauditCampaign,
  useResolveItem,
} from "./controllers/useCampaignController";
import { severityTokenToLevel } from "./severity";
import { statusToState, type CampaignItemState } from "./flow/transition";
import { AuditReportParseError, parseAuditReport } from "./lib/parseAuditReport";
import {
  CampaignLifecycle,
  type CampaignItem,
} from "@vrooli/proto-types/architecture-cartographer/v1/campaign/campaign_pb";

const SEVERITY_LABEL_KEY = {
  info: strings.shared.severity.info,
  low: strings.shared.severity.low,
  medium: strings.shared.severity.medium,
  high: strings.shared.severity.high,
  critical: strings.shared.severity.critical,
} as const satisfies Record<SeverityLevel, string>;

const STATUS_LABEL_KEY = {
  detected: strings.campaign.status.detected,
  assigned: strings.campaign.status.assigned,
  split: strings.campaign.status.split,
  resolved: strings.campaign.status.resolved,
  validated: strings.campaign.status.validated,
  committed: strings.campaign.status.committed,
  force_resolved: strings.campaign.status.force_resolved,
} as const satisfies Record<CampaignItemState, string>;

const PROFILE_OPTIONS = [
  { value: RankProfile.BALANCED, labelKey: strings.campaign.profile.balanced },
  { value: RankProfile.FAST, labelKey: strings.campaign.profile.fast },
  { value: RankProfile.LONG_TERM, labelKey: strings.campaign.profile.longTerm },
] as const;

function RollupStat({ label, value, danger = false }: { label: string; value: number; danger?: boolean }) {
  return (
    <div className="rounded-control border border-app-border bg-app-surface-muted p-2">
      <dt className="text-xs text-app-muted-foreground">{label}</dt>
      <dd className={`text-lg font-semibold ${danger ? "text-app-danger" : ""}`}>{value}</dd>
    </div>
  );
}

export interface CampaignDetailPanelProps {
  scenario: string;
  campaignId: string;
}

export function CampaignDetailPanel({ scenario, campaignId }: CampaignDetailPanelProps) {
  const { t } = useTranslation();
  const [profile, setProfile] = React.useState<RankProfile>(RankProfile.BALANCED);
  const status = useCampaignStatus({ id: campaignId });
  const worklist = useNextStep({ id: campaignId, profile });
  const resolve = useResolveItem(campaignId);
  const apply = useApplyItem(campaignId);
  const reaudit = useReauditCampaign(campaignId);
  const close = useCloseCampaign(campaignId, scenario);

  const [report, setReport] = React.useState("");
  const [reauditError, setReauditError] = React.useState<string | null>(null);

  if (status.isPending) {
    return (
      <div data-testid={selectors.features.campaign.detail.loading}>
        <LoadingState label={t(strings.pages.campaign.loading)} />
      </div>
    );
  }

  if (status.isError) {
    return (
      <div data-testid={selectors.features.campaign.detail.error}>
        <ErrorState
          title={t(strings.pages.campaign.errorTitle)}
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
  const campaign = projection?.campaign;
  if (!projection || !campaign) {
    return (
      <div data-testid={selectors.features.campaign.detail.notFound}>
        <EmptyState title={t(strings.pages.campaign.detail.notFound)} />
      </div>
    );
  }

  const isOpen = campaign.status !== CampaignLifecycle.CLOSED;
  const pending = resolve.isPending || apply.isPending || close.isPending;

  const onResolve = (item: CampaignItem) => {
    const note = window.prompt(t(strings.campaign.actions.resolveNotePrompt)) ?? "";
    resolve.mutate({ stableId: item.stableId, note: note.trim() });
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

  const items = worklist.data?.items ?? [];

  return (
    <article
      data-testid={selectors.features.campaign.detail.root}
      aria-labelledby={`campaign-detail-${campaign.id}`}
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-baseline gap-2">
          <h3 id={`campaign-detail-${campaign.id}`} className="text-lg font-semibold">
            {campaign.name || t(strings.pages.campaign.unnamed)}
          </h3>
          <Badge variant={isOpen ? "info" : "default"}>
            {t(isOpen ? strings.campaign.lifecycle.open : strings.campaign.lifecycle.closed)}
          </Badge>
        </div>
        <p className="text-xs font-mono text-app-muted-foreground">{campaign.id}</p>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.campaign.detail.scenarioLabel)}{" "}
          <span className="font-mono text-app-foreground">{campaign.scenario}</span>
        </p>
      </header>

      <dl
        data-testid={selectors.features.campaign.detail.rollup}
        className="grid grid-cols-2 gap-2 sm:grid-cols-5"
      >
        <RollupStat label={t(strings.pages.campaign.detail.rollup.total)} value={projection.total} />
        <RollupStat label={t(strings.pages.campaign.detail.rollup.open)} value={projection.open} />
        <RollupStat label={t(strings.pages.campaign.detail.rollup.resolved)} value={projection.resolved} />
        <RollupStat label={t(strings.pages.campaign.detail.rollup.validated)} value={projection.validated} />
        <RollupStat
          label={t(strings.pages.campaign.detail.rollup.regressions)}
          value={projection.regressions}
          danger={projection.regressions > 0}
        />
      </dl>

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <CardTitle>{t(strings.pages.campaign.detail.worklistHeading)}</CardTitle>
            <select
              data-testid={selectors.features.campaign.detail.profileSelect}
              value={profile}
              onChange={(e) => setProfile(Number(e.target.value))}
              className="rounded-control border border-app-border bg-app-surface px-2 py-1 text-xs"
            >
              {PROFILE_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {t(opt.labelKey)}
                </option>
              ))}
            </select>
          </div>
        </CardHeader>
        <CardBody data-testid={selectors.features.campaign.detail.worklist}>
          {items.length === 0 ? (
            <div data-testid={selectors.features.campaign.detail.worklistEmpty}>
              <EmptyState title={t(strings.pages.campaign.detail.worklistEmpty)} />
            </div>
          ) : (
            <ul className="flex flex-col gap-3">
              {items.map((item) => {
                const level = severityTokenToLevel(item.severity);
                return (
                  <li
                    key={item.stableId}
                    data-testid={selectors.features.campaign.detail.itemCard({ stableId: item.stableId })}
                    className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-2"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs text-app-muted-foreground">{item.source}</span>
                      <span className="text-sm font-medium">{item.code}</span>
                      <SeverityBadge level={level} label={t(SEVERITY_LABEL_KEY[level])} />
                      <Badge variant="outline">{t(STATUS_LABEL_KEY[statusToState(item.status)])}</Badge>
                      {item.effort && item.effort !== "unspecified" ? (
                        <Badge variant="outline" className="font-mono text-xs">{item.effort}</Badge>
                      ) : null}
                      {item.regressed ? (
                        <Badge variant="danger">{t(strings.pages.campaign.detail.regressedBadge)}</Badge>
                      ) : null}
                    </div>
                    {item.message ? <p className="text-sm">{item.message}</p> : null}
                    {item.locations.length > 0 ? (
                      <p className="font-mono text-xs text-app-muted-foreground">
                        <span className="not-italic">{t(strings.pages.campaign.detail.locationsLabel)}:</span>{" "}
                        {item.locations.join(", ")}
                      </p>
                    ) : null}
                    {item.suggestion ? (
                      <p className="text-xs text-app-muted-foreground">
                        {t(strings.pages.campaign.detail.suggestionLabel)} {item.suggestion}
                      </p>
                    ) : null}
                    {isOpen ? (
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          variant="default"
                          size="sm"
                          disabled={pending}
                          data-testid={selectors.features.campaign.detail.actionButton({
                            stableId: item.stableId,
                            action: "resolve",
                          })}
                          onClick={() => onResolve(item)}
                        >
                          {pending ? t(strings.campaign.actions.pending) : t(strings.campaign.actions.resolve)}
                        </Button>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          disabled={pending}
                          data-testid={selectors.features.campaign.detail.actionButton({
                            stableId: item.stableId,
                            action: "apply",
                          })}
                          onClick={() => apply.mutate(item.stableId)}
                        >
                          {pending ? t(strings.campaign.actions.pending) : t(strings.campaign.actions.apply)}
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
            <CardTitle>{t(strings.pages.campaign.detail.reauditHeading)}</CardTitle>
          </CardHeader>
          <CardBody data-testid={selectors.features.campaign.detail.reaudit} className="flex flex-col gap-2">
            <p className="text-xs text-app-muted-foreground">
              {t(strings.pages.campaign.detail.reauditHint, { scenario })}
            </p>
            <Textarea
              data-testid={selectors.features.campaign.detail.reauditInput}
              value={report}
              onChange={(e) => setReport(e.target.value)}
              placeholder={t(strings.pages.campaign.create.reportPlaceholder)}
              rows={4}
              className="font-mono text-xs"
            />
            {reauditError ? (
              <ErrorState title={t(strings.pages.campaign.create.parseErrorTitle)} message={reauditError} />
            ) : null}
            {reaudit.isSuccess ? (
              <p
                data-testid={selectors.features.campaign.detail.reauditResult}
                className="text-sm text-app-foreground"
              >
                {t(strings.pages.campaign.detail.reauditResult, {
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
                data-testid={selectors.features.campaign.detail.reauditSubmit}
                onClick={onReaudit}
                disabled={reaudit.isPending || report.trim().length === 0}
              >
                {reaudit.isPending
                  ? t(strings.pages.campaign.detail.reauditting)
                  : t(strings.pages.campaign.detail.reauditSubmit)}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                data-testid={selectors.features.campaign.detail.closeButton}
                onClick={() => close.mutate()}
                disabled={close.isPending}
              >
                {close.isPending
                  ? t(strings.pages.campaign.detail.closing)
                  : t(strings.pages.campaign.detail.closeButton)}
              </Button>
            </div>
          </CardBody>
        </Card>
      ) : (
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.campaign.detail.closedNote)}</p>
      )}
    </article>
  );
}
