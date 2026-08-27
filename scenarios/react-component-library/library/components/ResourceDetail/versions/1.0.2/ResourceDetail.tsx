/**
 * @libraryId react-component-library:ResourceDetail
 * @displayName ResourceDetail
 * @description A route-safe resource identity, metadata, history, freshness, permission, and recovery composition.
 * @version 1.0.2
 * @tags ["pattern","data-source","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource patterns.resource-detail */
import { resolveStrings, useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { CSSProperties, ReactNode } from "react";
import {
  AsyncBoundary,
  type AsyncBoundaryStatus,
} from "@vrooli/react-component-library/AsyncBoundary/1.0.0";
import { AuditTrail } from "@vrooli/react-component-library/AuditTrail/1.0.0";
import { DescriptionList } from "@vrooli/react-component-library/DescriptionList/1.0.0";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/1.0.0";

export interface ResourceDetailEntry {
  term: string;
  description: string;
}

export interface ResourceDetailHistoryEntry {
  actor: string;
  action: string;
}

export type ResourceDetailStatus =
  | "default"
  | "loading"
  | "refreshing"
  | "empty"
  | "partial"
  | "stale"
  | "submitting"
  | "success"
  | "request-error"
  | "permission-denied"
  | "offline"
  | "retry";

export interface ResourceDetailProps {
  title?: string;
  description?: string;
  entries?: ResourceDetailEntry[];
  history?: ResourceDetailHistoryEntry[];
  status?: ResourceDetailStatus;
  freshness?: string;
  actions?: ReactNode;
  children?: ReactNode;
  onRetry?: () => void | Promise<void>;
  permissionMessage?: ReactNode;
  emptyMessage?: ReactNode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-resource-detail] { display: grid; gap: var(--space-lg, 1.5rem); min-inline-size: 0; color: var(--color-foreground, #0f172a); }
[data-rcl-resource-detail-header] { display: grid; gap: var(--space-xs, .625rem); min-inline-size: 0; }
[data-rcl-resource-detail-header] > header { padding-block-end: 0; }
[data-rcl-resource-detail-freshness] { display: inline-flex; align-items: center; gap: var(--space-2xs, .35rem); inline-size: fit-content; max-inline-size: 100%; padding: var(--space-3xs, .2rem) var(--space-xs, .625rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, .625rem); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 .75rem/1.35 system-ui, sans-serif); }
[data-rcl-resource-detail-freshness]::before { content: ""; inline-size: .45rem; block-size: .45rem; border-radius: 50%; background: var(--color-success, #16803c); }
[data-rcl-resource-detail-grid] { display: grid; grid-template-columns: minmax(0, 1.35fr) minmax(16rem, .65fr); gap: var(--space-md, 1rem); align-items: start; min-inline-size: 0; }
[data-rcl-resource-detail-section] { display: grid; gap: var(--space-sm, .75rem); min-inline-size: 0; padding: var(--space-md, 1rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 1rem); background: var(--color-surface-raised, #fff); box-shadow: var(--elev-raised, 0 3px 12px rgb(15 23 42 / .06)); }
[data-rcl-resource-detail-section-title] { margin: 0; font: var(--text-subtitle, 700 1rem/1.35 system-ui, sans-serif); }
[data-rcl-resource-detail-section-copy] { margin: 0; color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 .9rem/1.45 system-ui, sans-serif); }
[data-rcl-resource-detail-message] { display: grid; place-items: center; min-block-size: 12rem; gap: var(--space-xs, .625rem); padding: var(--space-xl, 2rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 1rem); background: var(--color-surface-raised, #fff); color: var(--color-muted-foreground, #64748b); text-align: center; font: var(--text-body, 400 .9rem/1.45 system-ui, sans-serif); }
[data-rcl-resource-detail-message="permission"] { color: var(--color-warning, #b45309); }
[data-rcl-resource-detail-partial] { padding: var(--space-xs, .625rem) var(--space-sm, .75rem); border-inline-start: 3px solid var(--color-warning, #d97706); border-radius: var(--radius-control, .625rem); background: color-mix(in srgb, var(--color-warning, #d97706) 8%, var(--color-surface-raised, #fff)); color: var(--color-foreground, #0f172a); font: var(--text-caption, 600 .75rem/1.35 system-ui, sans-serif); }
@media (max-width: 52rem) { [data-rcl-resource-detail-grid] { grid-template-columns: 1fr; } }
@media (max-width: 34rem) { [data-rcl-resource-detail] { gap: var(--space-md, 1rem); } [data-rcl-resource-detail-section] { padding: var(--space-sm, .75rem); } }

`;

const boundaryStatus = (status: ResourceDetailStatus): AsyncBoundaryStatus => {
  if (status === "loading" || status === "retry") return "pending";
  if (status === "refreshing") return "refreshing";
  if (status === "stale") return "stale";
  if (status === "partial") return "partial-error";
  if (status === "request-error") return "error";
  if (status === "offline") return "offline";
  return "idle";
};

export const ResourceDetail = withClassName(function ResourceDetail({
  title = resolveStrings("patterns.resource-detail.resource", "Resource"),
  description = resolveStrings(
    "patterns.resource-detail.review-identity-metadata-and-recent-changes-in-o",
    "Review identity, metadata, and recent changes in one place.",
  ),
  entries = [],
  history = [],
  status = "default",
  freshness,
  actions,
  children,
  onRetry,
  permissionMessage = "You do not have permission to inspect this resource.",
  emptyMessage = "This resource has no detail to show yet.",
  className,
  style,
}: ResourceDetailProps) {
  const strings = useStrings();
  const isPermissionDenied = status === "permission-denied";
  const isEmpty = status === "empty";
  const content = isPermissionDenied ? (
    <div
      data-testid="patterns.resource-detail"
      data-rcl-resource-detail-message="permission"
      role="status"
    >
      {permissionMessage}
    </div>
  ) : isEmpty ? (
    <div data-rcl-resource-detail-message role="status">
      {emptyMessage}
    </div>
  ) : (
    <>
      <div data-rcl-resource-detail-grid>
        <section data-rcl-resource-detail-section aria-labelledby="rcl-resource-metadata-title">
          <h2 id="rcl-resource-metadata-title" data-rcl-resource-detail-section-title>
            {strings("patterns.resource-detail.metadata", "Metadata")}
          </h2>
          {entries.length ? (
            <DescriptionList entries={entries} />
          ) : (
            <p data-rcl-resource-detail-section-copy>
              {strings(
                "patterns.resource-detail.no-metadata-is-available-yet",
                "No metadata is available yet.",
              )}
            </p>
          )}
        </section>
        <section data-rcl-resource-detail-section aria-labelledby="rcl-resource-history-title">
          <h2 id="rcl-resource-history-title" data-rcl-resource-detail-section-title>
            {strings("patterns.resource-detail.history", "History")}
          </h2>
          {history.length ? (
            <AuditTrail entries={history} />
          ) : (
            <p data-rcl-resource-detail-section-copy>
              {strings(
                "patterns.resource-detail.no-recorded-changes-yet",
                "No recorded changes yet.",
              )}
            </p>
          )}
        </section>
      </div>
      {children}
    </>
  );
  return (
    <article data-rcl-resource-detail className={className} style={style}>
      <StyleSheet name="resourcedetail-1-0-2-1" css={styles} />
      <div data-rcl-resource-detail-header>
        <PageHeader title={title} description={description} actions={actions} />
        {freshness ? <span data-rcl-resource-detail-freshness>{freshness}</span> : null}
      </div>
      {status === "partial" ? (
        <div data-rcl-resource-detail-partial role="status">
          {strings(
            "patterns.resource-detail.some-fields-are-still-arriving-the-information-s",
            "Some fields are still arriving. The information shown is usable but not complete.",
          )}
        </div>
      ) : null}
      <AsyncBoundary
        status={isPermissionDenied || isEmpty ? "idle" : boundaryStatus(status)}
        retry={onRetry}
        preserveContent={status === "refreshing" || status === "stale" || status === "partial"}
        errorTitle="We couldn’t load this resource"
        error="The resource could not be refreshed. Your last useful view remains available when possible."
        aria-label={`${title} state`}
      >
        {content}
      </AsyncBoundary>
    </article>
  );
});
