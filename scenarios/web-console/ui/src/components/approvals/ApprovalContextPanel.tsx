import { useTranslation } from "react-i18next";
import { ExternalLink } from "lucide-react";
import type { ApprovalDecision } from "../../api/approvals";
import { strings } from "../../consts/strings";
import { safeContextUrl, sourceLabel } from "./approvalsFormat";

/**
 * Pluggable per-source context panel.
 *
 * The decision detail reserves ONE slot; this component chooses its body by the
 * decision's `source`. Every panel is fed from the decision itself (the hub's
 * domain-neutral Ask) plus a read-only `context_url` deep link to the producer's
 * own review surface — it NEVER calls a producer directly, so the inbox's
 * decoupling from agent-manager (and every other source) holds. When a source
 * has no bespoke panel the generic panel renders the event type, any ask-level
 * explanation, and the context link.
 *
 * The hub's Ask carries two optional, source-neutral explanations. They are
 * shown independently so either can be omitted without hiding the other.
 */
export default function ApprovalContextPanel({
  decision,
  contextUrl,
}: {
  decision: ApprovalDecision;
  contextUrl: string;
}) {
  const { t } = useTranslation();

  const label =
    decision.source === "deployment-manager"
      ? t(strings.approvals.contextReleaseLabel)
      : decision.source === "agent-manager"
        ? t(strings.approvals.contextRunLabel)
        : t(strings.approvals.contextGenericLabel);

  const rows: Array<{ term: string; detail: string }> = [];
  if (decision.kind) rows.push({ term: "Kind", detail: decision.kind });
  if (decision.sourceEventType) rows.push({ term: "Event", detail: decision.sourceEventType });
  if (decision.createdAt) rows.push({ term: "Raised", detail: decision.createdAt });
  if (decision.deadline) rows.push({ term: "Deadline", detail: decision.deadline });

  // Defense-in-depth: only link an http(s)/root-relative context_url. A
  // javascript:/data:/protocol-relative payload is rendered as inert text, never
  // an href. (The substrate validates Facts.context_url too; this guard stands
  // on its own so the UI never trusts the string.)
  const linkHref = safeContextUrl(contextUrl);

  return (
    <section
      data-testid="approval-context-panel"
      data-source={decision.source}
      className="rounded-lg border border-wc-default bg-wc-surface-raised p-4"
    >
      <h3 className="mb-3 text-[11px] font-bold uppercase tracking-wide text-wc-text-faint">{label}</h3>
      <dl className="grid grid-cols-[auto,1fr] gap-x-4 gap-y-1.5 text-[13px]">
        {rows.map((row) => (
          <div key={row.term} className="contents">
            <dt className="text-wc-text-faint">{row.term}</dt>
            <dd className="break-words text-wc-text-secondary">{row.detail}</dd>
          </div>
        ))}
      </dl>
      {decision.bigPicture && (
        <details className="mt-3 rounded-md border border-wc-default p-3">
          <summary className="cursor-pointer text-[13px] font-semibold text-wc-text-primary">
            {t(strings.approvals.bigPicture)}
          </summary>
          <p className="mt-2 whitespace-pre-wrap break-words text-[13px] text-wc-text-secondary">
            {decision.bigPicture}
          </p>
        </details>
      )}
      {decision.technicalDetails && (
        <details className="mt-3 rounded-md border border-wc-default p-3">
          <summary className="cursor-pointer text-[13px] font-semibold text-wc-text-primary">
            {t(strings.approvals.technicalDetails)}
          </summary>
          <p className="mt-2 whitespace-pre-wrap break-words text-[13px] text-wc-text-secondary">
            {decision.technicalDetails}
          </p>
        </details>
      )}
      {linkHref ? (
        <a
          data-testid="approval-context-link"
          href={linkHref}
          target="_blank"
          rel="noreferrer"
          className="mt-3 inline-flex items-center gap-1.5 text-[13px] text-wc-accent hover:underline"
        >
          {t(strings.approvals.openContext, { source: sourceLabel(decision.source) })}
          <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
        </a>
      ) : contextUrl ? (
        // A context_url that is not a safe http(s)/root-relative link is shown as
        // inert text so the operator still sees it, but it is never clickable.
        <p data-testid="approval-context-link-inert" className="mt-3 break-all text-[12px] text-wc-text-faint">
          {contextUrl}
        </p>
      ) : null}
    </section>
  );
}
