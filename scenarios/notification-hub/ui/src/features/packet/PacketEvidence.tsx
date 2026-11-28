import { Link, useLocation } from "react-router-dom";
import type { Ask, RelatedAsk } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Clock, ExternalLink } from "lucide-react";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Time } from "../../components/Time";
import { isAwaiting, safeContextUrl } from "../asks/askView";
import { preserveRoutedTestMode } from "../routedTestMode";
import {
  boundParts,
  FOLLOWTHROUGH_STATE_LABEL,
  FOLLOWTHROUGH_TONE,
  followthroughState,
  parseQualityReport,
  READINESS_LABEL,
  READINESS_TONE,
  readinessGrade,
  relatedLabelKey,
  scoreText,
  SEVERITY_LABEL,
  severityOf,
} from "./readiness";

/**
 * The packet's quieter evidence: how ready it is, what else might belong with
 * it, and what the answer started. Every part is optional; a missing report,
 * suggestion or report leaves nothing behind rather than a placeholder.
 */
export function PacketEvidence({ ask }: { ask: Ask }) {
  return (
    <>
      <ReadinessPanel ask={ask} />
      <RelatedDecisions related={ask.related} />
      <ImpactFacts ask={ask} />
    </>
  );
}

/** Grade plus the Hub's findings, collapsed to one line until the owner asks. */
export function ReadinessPanel({ ask }: { ask: Ask }) {
  const { t } = useTranslation();
  const report = parseQualityReport(ask.qualityReport);
  const grade = report?.grade || readinessGrade(ask.qualityGrade);
  const findings = report?.findings ?? [];
  if (!grade && !report) return null;
  return (
    <section aria-labelledby="packet-readiness-heading" data-testid="packet-readiness" className="rounded-panel border border-app-border bg-app-surface">
      <div className="flex flex-wrap items-center justify-between gap-2 px-3 py-2.5">
        <h2 id="packet-readiness-heading" className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-app-muted-foreground">
          {t(strings.packet.readiness.title)}
        </h2>
        {grade && <StatusBadge tone={READINESS_TONE[grade]}>{t(READINESS_LABEL[grade])}</StatusBadge>}
      </div>
      {findings.length > 0 ? (
        <details className="border-t border-app-border px-3 open:pb-3">
          <summary className="min-h-11 cursor-pointer py-2.5 text-sm font-medium">
            {t(strings.packet.readiness.findings)}
            <span className="ms-2 font-mono text-xs text-app-muted-foreground">{findings.length}</span>
          </summary>
          <ul className="flex flex-col gap-3">
            {findings.map((finding, index) => {
              const severity = severityOf(finding.severity);
              return (
                <li key={`${finding.code || "finding"}-${index}`} className="flex flex-col gap-1 text-sm">
                  <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                    <span className={`inline-flex items-center gap-1.5 font-medium ${severity === "blocker" ? "text-app-danger" : severity === "warning" ? "text-app-warning" : "text-app-muted-foreground"}`}>
                      <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-current" />
                      {t(SEVERITY_LABEL[severity])}
                    </span>
                    {finding.field && <span className="text-xs text-app-muted-foreground">{t(strings.packet.readiness.field, { field: finding.field })}</span>}
                  </div>
                  {finding.message && <p className="break-words text-app-muted-foreground">{finding.message}</p>}
                  {finding.owner && <p className="text-xs text-app-muted-foreground">{t(strings.packet.readiness.owner, { owner: finding.owner })}</p>}
                </li>
              );
            })}
          </ul>
        </details>
      ) : (
        <p className="border-t border-app-border px-3 py-2.5 text-sm text-app-muted-foreground">
          {report ? t(strings.packet.readiness.none) : t(strings.packet.readiness.unavailable)}
        </p>
      )}
    </section>
  );
}

/** Up to three suggestions, labelled by the evidence that connects them. */
export function RelatedDecisions({ related }: { related: RelatedAsk[] }) {
  const { t } = useTranslation();
  const { pathname, search } = useLocation();
  if (related.length === 0) return null;
  return (
    <section aria-labelledby="packet-related-heading" data-testid="packet-related" className="flex flex-col gap-2">
      <h2 id="packet-related-heading" className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-app-muted-foreground">
        {t(strings.packet.related.title)}
      </h2>
      <ul className="flex flex-col">
        {related.slice(0, 3).map((item) => {
          const labelKey = relatedLabelKey(item.basis);
          return (
            <li key={item.askId} className="border-t border-app-border first:border-t-0">
              <Link
                to={preserveRoutedTestMode(`/asks/${encodeURIComponent(item.askId)}`, search)}
                state={{ from: pathname + search }}
                className="flex min-h-11 flex-col justify-center gap-0.5 rounded-control px-2 py-2 hover:bg-app-surface-muted"
              >
                <span className="break-words text-sm font-medium leading-snug">{item.title || item.question || item.askId}</span>
                <span className="flex flex-wrap items-center gap-x-2 text-xs text-app-muted-foreground">
                  <span className="text-app-primary">{t(labelKey, { score: scoreText(item.score) })}</span>
                  {item.sender && <><span aria-hidden="true">·</span><span className="[overflow-wrap:anywhere]">{item.sender}</span></>}
                </span>
              </Link>
            </li>
          );
        })}
      </ul>
      <p className="text-xs leading-relaxed text-app-muted-foreground">{t(strings.packet.related.note)}</p>
    </section>
  );
}

/**
 * What the owner reported after the answer. It shows only the fields the
 * owner sent: an unanswered owner is not read as progress, and an answered
 * ask with no report says exactly that.
 */
export function FollowthroughBanner({ ask }: { ask: Ask }) {
  const { t } = useTranslation();
  const report = ask.followthrough;
  if (!report) {
    if (ask.state !== "answered") return null;
    return (
      <section aria-labelledby="packet-followthrough-heading" data-testid="packet-followthrough" className="rounded-panel border border-app-border bg-app-surface px-3 py-2.5">
        <h2 id="packet-followthrough-heading" className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-app-muted-foreground">
          {t(strings.packet.followthrough.title)}
        </h2>
        <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.packet.followthrough.absent)}</p>
      </section>
    );
  }
  const state = followthroughState(report.state);
  const bound = boundParts(report.boundSeconds);
  const boundLabel = bound
    ? bound.key === strings.packet.followthrough.boundMinutesSeconds
      ? t(bound.key, { minutes: bound.minutes ?? 0, seconds: bound.seconds ?? 0 })
      : t(bound.key, { count: bound.count ?? 0 })
    : null;
  const resultHref = report.resultUrl.trim() ? safeContextUrl(report.resultUrl) : null;
  return (
    <section aria-labelledby="packet-followthrough-heading" data-testid="packet-followthrough" className="rounded-panel border border-app-border bg-app-surface px-3 py-2.5">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
        <h2 id="packet-followthrough-heading" className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-app-muted-foreground">
          {t(strings.packet.followthrough.title)}
        </h2>
        {report.state && (
          <StatusBadge tone={state ? FOLLOWTHROUGH_TONE[state] : "neutral"}>
            {state ? t(FOLLOWTHROUGH_STATE_LABEL[state]) : report.state}
          </StatusBadge>
        )}
        {report.owner && <span className="text-xs text-app-muted-foreground">{t(strings.packet.followthrough.owner, { owner: report.owner })}</span>}
      </div>
      {(bound || report.startedAt || report.finishedAt) && (
        <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-app-muted-foreground">
          {bound && <span className="inline-flex items-center gap-1"><Clock size={13} aria-hidden="true" />{boundLabel}</span>}
          {report.startedAt && (report.state === "running" || report.state === "queued") && (
            <span>{t(strings.packet.followthrough.since)} <Time value={report.startedAt} /></span>
          )}
          {report.finishedAt && <span>{t(strings.packet.followthrough.finished)} <Time value={report.finishedAt} /></span>}
        </div>
      )}
      {report.summary && <p className="mt-2 break-words text-sm leading-relaxed text-app-muted-foreground">{report.summary}</p>}
      {resultHref && (
        <a href={resultHref} target="_blank" rel="noopener noreferrer" className="mt-2 inline-flex min-h-11 items-center gap-1.5 text-sm font-medium text-app-primary hover:underline">
          {t(strings.packet.followthrough.result)}
          <ExternalLink size={14} aria-hidden="true" />
        </a>
      )}
    </section>
  );
}

/** What the decision holds up, and what waiting costs: one quiet line each. */
export function ImpactFacts({ ask }: { ask: Ask }) {
  const { t } = useTranslation();
  const waiting = isAwaiting(ask);
  if (!ask.unblocks && !(waiting && ask.costOfWaiting)) return null;
  return (
    <ul data-testid="packet-impact" aria-label={t(strings.packet.impact.title)} className="flex flex-col gap-1 text-xs leading-relaxed text-app-muted-foreground">
      {ask.unblocks && <li className="break-words">{t(strings.packet.impact.unblocks, { value: ask.unblocks })}</li>}
      {waiting && ask.costOfWaiting && <li className="break-words">{t(strings.packet.impact.cost, { value: ask.costOfWaiting })}</li>}
    </ul>
  );
}
