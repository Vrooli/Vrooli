/**
 * @libraryId react-component-library:MonetizationAccount
 * @displayName Monetization Account
 * @description Reusable monetization account component implementation for the React component library.
 * @version 2.0.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode } from "react";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

export type EntitlementStatus = "active" | "trialing" | "past_due" | "canceled" | "inactive";
export type PlanTier = "free" | "solo" | "pro" | "studio" | "business";

export const PLAN_CONFIG: Record<
  PlanTier,
  { label: string; color: string; bgColor: string; borderColor: string }
> = {
  free: {
    label: "Free",
    color: "var(--color-muted-foreground)",
    bgColor: "var(--color-surface-muted)",
    borderColor: "var(--color-border)",
  },
  solo: {
    label: "Solo",
    color: "var(--color-foreground)",
    bgColor: "var(--color-surface)",
    borderColor: "var(--color-border)",
  },
  pro: {
    label: "Pro",
    color: "var(--color-primary)",
    bgColor: "var(--color-surface)",
    borderColor: "var(--color-primary)",
  },
  studio: {
    label: "Studio",
    color: "var(--color-primary)",
    bgColor: "var(--color-surface-muted)",
    borderColor: "var(--color-primary)",
  },
  business: {
    label: "Business",
    color: "var(--color-foreground)",
    bgColor: "var(--color-surface-muted)",
    borderColor: "var(--color-border)",
  },
};

export const PlanBadge = withClassName(function PlanBadge({
  plan,
  size = "md",
}: {
  plan: PlanTier;
  size?: "sm" | "md";
}) {
  const config = PLAN_CONFIG[plan];
  return (
    <span
      data-testid="monetization.account-surface"
      data-size={size}
      style={{
        display: "inline-flex",
        alignItems: "center",
        border: `var(--border-hairline) solid ${config.borderColor}`,
        borderRadius: "var(--radius-control)",
        background: config.bgColor,
        color: config.color,
        padding: "var(--space-4xs) var(--space-2xs)",
        fontSize: "var(--text-caption-size)",
        lineHeight: "var(--text-caption-line)",
        fontWeight: 600,
      }}
    >
      {config.label}
    </span>
  );
});

export const SubscriptionStatusCard = withClassName(function SubscriptionStatusCard({
  plan,
  status,
  credits,
  multiplier = 1,
  label = "credits",
}: {
  plan: string;
  status: EntitlementStatus;
  credits: number;
  multiplier?: number;
  label?: string;
}) {
  return (
    <section aria-label="Subscription status">
      <strong>{plan}</strong>
      <span>{status}</span>
      <span>
        {credits * multiplier} {label}
      </span>
    </section>
  );
});

export const AuthSection = withClassName(function AuthSection({
  signedIn,
  onSignIn,
  onSignOut,
}: {
  signedIn: boolean;
  onSignIn: () => void;
  onSignOut: () => void;
}) {
  return (
    <section aria-label="Account">
      <button type="button" onClick={signedIn ? onSignOut : onSignIn}>
        {signedIn ? "Sign out" : "Sign in"}
      </button>
    </section>
  );
});

export const UpgradePrompt = withClassName(function UpgradePrompt({
  feature,
  requiredPlan,
  href = "/account",
}: {
  feature: string;
  requiredPlan: string;
  href?: string;
}) {
  return (
    <section role="alert">
      <p>
        {feature} requires {requiredPlan}.
      </p>
      <a href={href}>Manage subscription</a>
    </section>
  );
});

export const PendingSyncBadge = withClassName(function PendingSyncBadge({
  pending,
}: {
  pending: number;
}) {
  return pending > 0 ? <span role="status">{pending} pending sync</span> : null;
});

export const SubscriptionBadge = withClassName(function SubscriptionBadge({
  plan,
  status,
  credits,
  offline = false,
  onClick,
}: {
  plan: string;
  status: EntitlementStatus;
  credits: number;
  offline?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Manage subscription"
      style={{ position: "relative" }}
    >
      <span>{plan}</span>
      <span>{status}</span>
      <span>{credits} credits</span>
      {offline ? <span aria-label="offline">cached</span> : null}
    </button>
  );
});

export const UsageMeter = withClassName(function UsageMeter({
  used,
  limit,
  label = "credits",
}: {
  used: number;
  limit: number;
  label?: string;
}) {
  const safeLimit = Math.max(0, limit);
  const percent = safeLimit > 0 ? Math.min(100, Math.max(0, (used / safeLimit) * 100)) : 0;
  return (
    <section aria-label={`${label} usage`}>
      <span>
        {used} / {safeLimit} {label}
      </span>
      <progress value={percent} max={100} aria-label={`${label} usage percent`} />
    </section>
  );
});

export const EntitlementErrorCard = withClassName(function EntitlementErrorCard({
  errorType,
  title = "Subscription access unavailable",
  message = "Subscription access is temporarily unavailable.",
  plan,
  creditsUsed,
  creditsLimit,
  resetDate,
  onManage,
}: {
  errorType: string;
  title?: string;
  message?: string;
  plan?: string;
  creditsUsed?: number;
  creditsLimit?: number;
  resetDate?: string;
  onManage?: () => void;
}) {
  const copy: Record<string, string> = {
    unauthorized: "Sign in to use this feature.",
    subscription_required: "An active subscription is required.",
    credits_required: "More credits are required for this operation.",
    authority_unavailable: "Subscription service is temporarily unavailable.",
    rate_limited: "This operation is temporarily rate limited.",
    rank_required: "Your current plan does not include this feature.",
  };
  return (
    <section role="alert" data-error-type={errorType}>
      <strong>{title}</strong>
      <p>{copy[errorType] || message}</p>
      {plan ? <span>Current plan: {plan}</span> : null}
      {creditsLimit && creditsLimit > 0 ? (
        <span>
          {creditsUsed ?? creditsLimit} / {creditsLimit} credits used
        </span>
      ) : null}
      {resetDate ? <span>{resetDate}</span> : null}
      {onManage ? (
        <button type="button" onClick={onManage}>
          Manage subscription
        </button>
      ) : null}
    </section>
  );
});

export interface MonetizationAccountPanelProps {
  configured: boolean;
  plan: PlanTier;
  status: string;
  statusLabel: string;
  creditsLabel: string;
  renewalLabel?: string;
  pendingSync?: number;
  busyAction?: string | null;
  error?: string;
  children?: ReactNode;
  signInDetails?: ReactNode;
  onSignIn: () => void;
  onSignOut: () => void;
  onRefresh: () => void;
  onManageAccount: () => void;
  onComparePlans: () => void;
}

const CONTROL_STYLE = {
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-control)",
  background: "var(--color-surface)",
  color: "var(--color-foreground)",
  padding: "var(--space-3xs, 4px) var(--space-xs, 12px)",
  font: "inherit",
  cursor: "pointer",
};

const PANEL_STYLE = {
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-card)",
  background: "var(--color-surface)",
  color: "var(--color-foreground)",
};

/** Shared account and subscription presentation. Entitlement decisions remain server-owned. */
export const MonetizationAccountPanel = withClassName(function MonetizationAccountPanel({
  configured,
  plan,
  status,
  statusLabel,
  creditsLabel,
  renewalLabel,
  pendingSync = 0,
  busyAction,
  error,
  children,
  signInDetails,
  onSignIn,
  onSignOut,
  onRefresh,
  onManageAccount,
  onComparePlans,
}: MonetizationAccountPanelProps) {
  const statusTone =
    status === "active" || status === "trialing" || status === "connected"
      ? "var(--color-success, #16a34a)"
      : status === "past_due" || status === "canceled"
        ? "var(--color-warning, #d97706)"
        : "var(--color-muted-foreground)";

  return (
    <section
      aria-label="Subscription and access"
      data-testid="monetization-account-panel"
      style={{ display: "grid", gap: "var(--space-md, 24px)" }}
    >
      <header
        style={{
          display: "flex",
          alignItems: "flex-start",
          justifyContent: "space-between",
          gap: "var(--space-sm, 16px)",
        }}
      >
        <div>
          <p
            style={{
              margin: 0,
              color: "var(--color-muted-foreground)",
              fontSize: "var(--text-caption-size)",
            }}
          >
            Account
          </p>
          <h2
            style={{ margin: "var(--space-4xs, 4px) 0", fontSize: "var(--text-title-size, 24px)" }}
          >
            Subscription and access
          </h2>
          <p style={{ margin: 0, color: "var(--color-muted-foreground)" }}>
            Manage the account that powers Vrooli services on this machine. Local workspace features
            stay available without an account.
          </p>
        </div>
        <button
          type="button"
          onClick={onRefresh}
          disabled={busyAction === "refresh"}
          aria-label="Refresh account status"
          style={CONTROL_STYLE}
        >
          {busyAction === "refresh" ? "Refreshing…" : "Refresh"}
        </button>
      </header>

      <section
        aria-label="Account identity"
        data-testid="account-identity-card"
        style={{
          ...PANEL_STYLE,
          padding: "var(--space-md, 24px)",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: "var(--space-sm, 16px)",
          flexWrap: "wrap",
        }}
      >
        <div>
          <strong>{configured ? "Vrooli account connected" : "Not signed in"}</strong>
          <p
            style={{ margin: "var(--space-4xs, 4px) 0 0", color: "var(--color-muted-foreground)" }}
          >
            {configured
              ? "Subscription access is shared with Vrooli apps on this machine."
              : "Sign in only when you want Vrooli-provided services."}
          </p>
        </div>
        <button
          type="button"
          onClick={configured ? onSignOut : onSignIn}
          disabled={busyAction === "sign-in" || busyAction === "sign-out"}
          style={{
            ...CONTROL_STYLE,
            background: configured ? "var(--color-surface)" : "var(--color-primary)",
            color: configured
              ? "var(--color-foreground)"
              : "var(--color-primary-foreground, #ffffff)",
          }}
        >
          {busyAction === "sign-in"
            ? "Connecting…"
            : busyAction === "sign-out"
              ? "Signing out…"
              : configured
                ? "Sign out"
                : "Sign in"}
        </button>
        {!configured && signInDetails ? (
          <div
            style={{
              flexBasis: "100%",
              borderTop: "var(--border-hairline) solid var(--color-border)",
              paddingTop: "var(--space-sm, 16px)",
            }}
          >
            {signInDetails}
          </div>
        ) : null}
      </section>

      <section
        aria-label="Subscription overview"
        data-testid="account-overview-card"
        style={{ ...PANEL_STYLE, padding: "var(--space-md, 24px)" }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            gap: "var(--space-sm, 16px)",
            flexWrap: "wrap",
          }}
        >
          <h3 style={{ margin: 0 }}>Subscription overview</h3>
          <div
            style={{ display: "inline-flex", alignItems: "center", gap: "var(--space-2xs, 8px)" }}
          >
            <PlanBadge plan={plan} size="sm" />
            <span
              data-testid="account-status"
              style={{ color: statusTone }}
              aria-label={`Subscription status: ${statusLabel}`}
            >
              {statusLabel}
            </span>
          </div>
        </div>
        <dl
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(10rem, 1fr))",
            gap: "var(--space-sm, 16px)",
            margin: "var(--space-md, 24px) 0",
          }}
        >
          <div>
            <dt style={{ color: "var(--color-muted-foreground)" }}>Plan</dt>
            <dd style={{ margin: 0 }}>{PLAN_CONFIG[plan].label}</dd>
          </div>
          <div>
            <dt style={{ color: "var(--color-muted-foreground)" }}>AI credits</dt>
            <dd style={{ margin: 0 }}>{creditsLabel}</dd>
          </div>
          <div>
            <dt style={{ color: "var(--color-muted-foreground)" }}>Renewal</dt>
            <dd style={{ margin: 0 }}>{renewalLabel ?? "Managed by your account"}</dd>
          </div>
        </dl>
        <div
          role="group"
          aria-label="Billing actions"
          style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-xs, 12px)" }}
        >
          <button type="button" onClick={onManageAccount} style={CONTROL_STYLE}>
            Manage billing
          </button>
          <button type="button" onClick={onComparePlans} style={CONTROL_STYLE}>
            Compare plans
          </button>
        </div>
      </section>

      {pendingSync > 0 ? (
        <div
          role="status"
          data-testid="account-pending-sync"
          style={{ ...PANEL_STYLE, padding: "var(--space-sm, 16px)" }}
        >
          {pendingSync} usage {pendingSync === 1 ? "item" : "items"} waiting to sync. Local activity
          will be reconciled when this machine is online.
        </div>
      ) : null}
      {error ? (
        <div
          role="alert"
          data-testid="account-error"
          style={{
            ...PANEL_STYLE,
            padding: "var(--space-sm, 16px)",
            borderColor: "var(--color-danger, #dc2626)",
          }}
        >
          <strong>Account status unavailable</strong>
          <p style={{ margin: "var(--space-4xs, 4px) 0 0" }}>{error}</p>
        </div>
      ) : null}
      {children}
    </section>
  );
});

export function useEntitlement<
  T extends { features?: string[]; planRank?: number; stale?: boolean },
>(lease: T | null) {
  return {
    lease,
    stale: Boolean(lease?.stale),
    hasFeature: (key: string) => Boolean(lease?.features?.includes(key)),
    atLeastRank: (rank: number) => (lease?.planRank ?? 0) >= rank,
  };
}
