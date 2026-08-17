export type EntitlementStatus = "active" | "trialing" | "past_due" | "canceled" | "inactive";
export type PlanTier = "free" | "solo" | "pro" | "studio" | "business";

export const PLAN_CONFIG: Record<PlanTier, { label: string; color: string; bgColor: string; borderColor: string }> = {
  free: { label: "Free", color: "text-gray-600", bgColor: "bg-gray-100", borderColor: "border-gray-300" },
  solo: { label: "Solo", color: "text-blue-600", bgColor: "bg-blue-100", borderColor: "border-blue-300" },
  pro: { label: "Pro", color: "text-purple-600", bgColor: "bg-purple-100", borderColor: "border-purple-300" },
  studio: { label: "Studio", color: "text-pink-600", bgColor: "bg-pink-100", borderColor: "border-pink-300" },
  business: { label: "Business", color: "text-amber-600", bgColor: "bg-amber-100", borderColor: "border-amber-300" },
};

export function PlanBadge({ plan, size = "md" }: { plan: PlanTier; size?: "sm" | "md" }) {
  const config = PLAN_CONFIG[plan];
  return <span className={`${config.color} ${config.bgColor} ${config.borderColor} inline-flex items-center rounded border px-2 py-0.5 text-xs font-medium ${size === "sm" ? "text-[10px]" : ""}`}>{config.label}</span>;
}

export function SubscriptionStatusCard({ plan, status, credits, multiplier = 1, label = "credits" }: { plan: string; status: EntitlementStatus; credits: number; multiplier?: number; label?: string }) {
  return <section aria-label="Subscription status"><strong>{plan}</strong><span>{status}</span><span>{credits * multiplier} {label}</span></section>;
}

export function AuthSection({ signedIn, onSignIn, onSignOut }: { signedIn: boolean; onSignIn: () => void; onSignOut: () => void }) {
  return <section aria-label="Account"><button type="button" onClick={signedIn ? onSignOut : onSignIn}>{signedIn ? "Sign out" : "Sign in"}</button></section>;
}

export function UpgradePrompt({ feature, requiredPlan, href = "/account" }: { feature: string; requiredPlan: string; href?: string }) {
  return <section role="alert"><p>{feature} requires {requiredPlan}.</p><a href={href}>Manage subscription</a></section>;
}

export function PendingSyncBadge({ pending }: { pending: number }) {
  return pending > 0 ? <span role="status">{pending} pending sync</span> : null;
}

export function SubscriptionBadge({ plan, status, credits, offline = false, onClick }: { plan: string; status: EntitlementStatus; credits: number; offline?: boolean; onClick?: () => void }) {
  return (
    <button type="button" onClick={onClick} aria-label="Manage subscription" className="relative">
      <span>{plan}</span>
      <span>{status}</span>
      <span>{credits} credits</span>
      {offline ? <span aria-label="offline">cached</span> : null}
    </button>
  );
}

export function EntitlementErrorCard({
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
      {creditsLimit && creditsLimit > 0 ? <span>{creditsUsed ?? creditsLimit} / {creditsLimit} credits used</span> : null}
      {resetDate ? <span>{resetDate}</span> : null}
      {onManage ? <button type="button" onClick={onManage}>Manage subscription</button> : null}
    </section>
  );
}

export function useEntitlement<T extends { features?: string[]; planRank?: number; stale?: boolean }>(lease: T | null) {
  return { lease, stale: Boolean(lease?.stale), hasFeature: (key: string) => Boolean(lease?.features?.includes(key)), atLeastRank: (rank: number) => (lease?.planRank ?? 0) >= rank };
}
