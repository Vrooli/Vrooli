/**
 * @libraryId react-component-library:MonetizationAccount
 * @displayName MonetizationAccount
 * @description Shared account surface primitives for paid scenarios.
 * @version 1.0.2
 * @tags ["monetization","account"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

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
    color: "text-app-muted-foreground",
    bgColor: "bg-app-surface-muted",
    borderColor: "border-app-border",
  },
  solo: {
    label: "Solo",
    color: "text-app-foreground",
    bgColor: "bg-app-surface",
    borderColor: "border-app-border",
  },
  pro: {
    label: "Pro",
    color: "text-app-primary",
    bgColor: "bg-app-surface",
    borderColor: "border-app-primary",
  },
  studio: {
    label: "Studio",
    color: "text-app-primary",
    bgColor: "bg-app-surface-muted",
    borderColor: "border-app-primary",
  },
  business: {
    label: "Business",
    color: "text-app-foreground",
    bgColor: "bg-app-surface-muted",
    borderColor: "border-app-border",
  },
};

export function PlanBadge({
  plan,
  size = "md",
}: {
  plan: PlanTier;
  size?: "sm" | "md";
}) {
  const config = PLAN_CONFIG[plan];
  return (
    <span
      className={`${config.color} ${config.bgColor} ${config.borderColor} inline-flex items-center rounded border px-space-2xs py-space-4xs text-caption font-medium ${size === "sm" ? "text-caption" : ""}`}
    >
      {config.label}
    </span>
  );
}

export function SubscriptionStatusCard({
  plan,
  status,
  credits,
  multiplier = 1,
  label = translate("monetization.account-surface.label.1", "credits"),
}: {
  plan: string;
  status: EntitlementStatus;
  credits: number;
  multiplier?: number;
  label?: string;
}) {
  return (
    <section aria-label={translate("monetization.account-surface.aria-label.2", "Subscription status")}>
      <strong>{plan}</strong>
      <span>{status}</span>
      <span>
        {credits * multiplier} {label}
      </span>
    </section>
  );
}

export function AuthSection({
  signedIn,
  onSignIn,
  onSignOut,
}: {
  signedIn: boolean;
  onSignIn: () => void;
  onSignOut: () => void;
}) {
  return (
    <section aria-label={translate("monetization.account-surface.aria-label.3", "Account")}>
      <button type="button" onClick={signedIn ? onSignOut : onSignIn}>
        {signedIn ? "Sign out" : "Sign in"}
      </button>
    </section>
  );
}

export function UpgradePrompt({
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
      <a href={href}>{translate("monetization.account-surface.text.7", "Manage subscription")}</a>
    </section>
  );
}

export function PendingSyncBadge({ pending }: { pending: number }) {
  return pending > 0 ? <span role="status">{pending} pending sync</span> : null;
}

export function SubscriptionBadge({
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
      aria-label={translate("monetization.account-surface.aria-label.4", "Manage subscription")}
      className="relative"
    >
      <span>{plan}</span>
      <span>{status}</span>
      <span>{credits} credits</span>
      {offline ? <span aria-label={translate("monetization.account-surface.aria-label.5", "offline")}>{translate("monetization.account-surface.text.8", "cached")}</span> : null}
    </button>
  );
}

export function EntitlementErrorCard({
  errorType,
  title = translate("monetization.account-surface.title.6", "Subscription access unavailable"),
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
}

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
