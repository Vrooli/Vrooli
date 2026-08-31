/**
 * @libraryId react-component-library:MonetizationAccount
 * @displayName MonetizationAccount
 * @description Shared account surface primitives for paid scenarios.
 * @version 1.0.4
 * @tags ["monetization","account"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

export type EntitlementStatus =
  | "active"
  | "trialing"
  | "past_due"
  | "canceled"
  | "inactive";
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

export const SubscriptionStatusCard = withClassName(
  function SubscriptionStatusCard({
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
  },
);

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

export const EntitlementErrorCard = withClassName(
  function EntitlementErrorCard({
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
    return (
      <section role="alert" data-error-type={errorType}>
        <strong>{title}</strong>
        <p>{message}</p>
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
  },
);

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
