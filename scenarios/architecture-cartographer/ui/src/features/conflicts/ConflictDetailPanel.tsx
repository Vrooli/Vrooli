import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Button } from "../../components/ui/button";
import { Card, CardBody, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { SeverityBadge } from "../../components/SeverityBadge";
import { useConflictActions } from "./hooks/useConflictActions";
import {
  useAssignConflict,
  useGetConflict,
  useReopenConflict,
  useResolveConflict,
  useValidateConflicts,
} from "./controllers/useConflictsController";
import { severityToLevel } from "./severity";
import type { ConflictEvent } from "./flow/transition";
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

const ACTION_LABEL_KEY = {
  assign: strings.conflicts.actions.assign,
  split: strings.conflicts.actions.split,
  resolve: strings.conflicts.actions.resolve,
  force_resolve: strings.conflicts.actions.force_resolve,
  validate: strings.conflicts.actions.validate,
  commit: strings.conflicts.actions.commit,
  reopen: strings.conflicts.actions.reopen,
} as const satisfies Record<ConflictEvent, string>;

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

export function ConflictDetailPanel({ scenario, conflictId }: ConflictDetailPanelProps) {
  const { t } = useTranslation();
  const detail = useGetConflict({ id: conflictId });
  const conflict = detail.data?.conflict;
  const { legalEvents } = useConflictActions(conflict);

  const assign = useAssignConflict(scenario);
  const resolve = useResolveConflict(scenario);
  const reopen = useReopenConflict(scenario);
  const validate = useValidateConflicts(scenario);

  const pending =
    assign.isPending || resolve.isPending || reopen.isPending || validate.isPending;

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

  const runAction = (event: ConflictEvent) => {
    switch (event) {
      case "assign": {
        const domain = window.prompt(t(strings.conflicts.actions.assignDomainPrompt)) ?? "";
        if (domain.trim().length === 0) return;
        assign.mutate({ id: conflict.id, domain: domain.trim() });
        return;
      }
      case "resolve":
        resolve.mutate({ id: conflict.id, force: false });
        return;
      case "force_resolve": {
        const note = window.prompt(t(strings.conflicts.actions.notePlaceholder)) ?? "";
        if (note.trim().length === 0) {
          window.alert(t(strings.conflicts.actions.forceNoteRequired));
          return;
        }
        resolve.mutate({ id: conflict.id, note: note.trim(), force: true });
        return;
      }
      case "reopen":
        reopen.mutate({ id: conflict.id });
        return;
      case "validate":
        validate.mutate();
        return;
      case "split":
      case "commit":
        // v0.1: split & commit are routed through dedicated flows (apply for
        // commit, deferred for split). Buttons render to communicate state
        // but the action is a no-op until those phases land.
        return;
    }
  };

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
        <p
          data-testid={selectors.features.conflicts.detail.assignedDomain}
          className="text-sm text-app-muted-foreground"
        >
          {t(strings.pages.conflicts.assignedDomainLabel)}{" "}
          <span className="font-mono">
            {conflict.assignedDomain || t(strings.pages.conflicts.noAssignedDomain)}
          </span>
        </p>
      </header>

      <div
        data-testid={selectors.features.conflicts.detail.actions}
        className="flex flex-wrap gap-2"
        role="group"
        aria-label={t(strings.pages.conflicts.title)}
      >
        {legalEvents.map((event) => (
          <Button
            key={event}
            type="button"
            variant={event === "force_resolve" ? "outline" : "default"}
            size="sm"
            disabled={pending}
            data-testid={selectors.features.conflicts.detail.actionButton({ event })}
            onClick={() => runAction(event)}
          >
            {pending
              ? t(strings.conflicts.actions.pending)
              : t(ACTION_LABEL_KEY[event])}
          </Button>
        ))}
      </div>

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
