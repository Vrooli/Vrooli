/**
 * @libraryId react-component-library:IntegrationCard
 * @displayName IntegrationCard
 * @description
 * @version 0.1.4
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode } from "react";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";

export type IntegrationCardStatus =
  | "connected"
  | "checking"
  | "needs_attention"
  | "disconnected"
  | "expired"
  | "insufficient_scope"
  | "provider_unavailable"
  | "revoked"
  | "offline"
  | "unknown";

export interface IntegrationCardProps {
  providerName: string;
  connectionName: string;
  accountLabel?: string;
  status: IntegrationCardStatus;
  scopes?: string[];
  usageSummary?: string;
  lastVerifiedAt?: string;
  freshness?: string;
  bindings?: string[];
  nextAction?: string;
  actions?: ReactNode;
}

const statusLabel: Record<IntegrationCardStatus, string> = {
  connected: "Connected",
  checking: "Checking",
  needs_attention: "Needs attention",
  disconnected: "Disconnected",
  expired: "Expired",
  insufficient_scope: "Insufficient permissions",
  provider_unavailable: "Provider unavailable",
  revoked: "Revoked",
  offline: "Offline",
  unknown: "Unknown",
};

const statusTone: Record<IntegrationCardStatus, StatusTone> = {
  connected: "success",
  checking: "info",
  needs_attention: "warning",
  disconnected: "neutral",
  expired: "warning",
  insufficient_scope: "warning",
  provider_unavailable: "danger",
  revoked: "danger",
  offline: "warning",
  unknown: "neutral",
};

/** Provider-neutral connection presentation; lifecycle ownership stays outside the card. */
export function IntegrationCard({
  providerName = "Provider",
  connectionName = "Connection",
  accountLabel,
  status = "unknown",
  scopes = [],
  usageSummary,
  lastVerifiedAt,
  freshness,
  bindings = [],
  nextAction,
  actions,
}: IntegrationCardProps) {
  return (
    <article
      data-testid="integration-card"
      aria-label={`${providerName}: ${connectionName}`}
      style={{
        display: "grid",
        gap: "var(--space-sm, 16px)",
        padding: "var(--space-md, 24px)",
        border: "1px solid var(--color-border, #cbd5e1)",
        borderRadius: "var(--radius-card, 0.75rem)",
      }}
    >
      <header
        style={{ display: "flex", justifyContent: "space-between", gap: "var(--space-sm, 16px)" }}
      >
        <div>
          <strong>{connectionName}</strong>
          <div>
            {providerName}
            {accountLabel ? ` · ${accountLabel}` : ""}
          </div>
        </div>
        <StatusBadge
          role="status"
          aria-label={`Connection status: ${statusLabel[status]}`}
          tone={statusTone[status]}
        >
          {statusLabel[status]}
        </StatusBadge>
      </header>
      {scopes.length > 0 && (
        <dl>
          <dt>Scopes</dt>
          <dd>{scopes.join(", ")}</dd>
        </dl>
      )}
      {bindings.length > 0 && (
        <dl>
          <dt>Used by</dt>
          <dd>{bindings.join(", ")}</dd>
        </dl>
      )}
      {usageSummary && <p>{usageSummary}</p>}
      {(lastVerifiedAt || freshness) && (
        <p>
          Last verified: {lastVerifiedAt ?? "Not verified"}
          {freshness ? ` · ${freshness}` : ""}
        </p>
      )}
      {nextAction && <p>{nextAction}</p>}
      {actions && <footer>{actions}</footer>}
    </article>
  );
}

export default IntegrationCard;
