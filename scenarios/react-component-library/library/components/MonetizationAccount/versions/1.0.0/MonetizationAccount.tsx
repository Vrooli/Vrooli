import type { ReactNode } from "react";

export type EntitlementStatus = "active" | "trialing" | "past_due" | "canceled" | "inactive";

export interface SubscriptionStatusCardProps { plan: string; status: EntitlementStatus; credits: number; multiplier?: number; label?: string; className?: string }
export interface AuthSectionProps { signedIn: boolean; onSignIn: () => void; onSignOut: () => void; className?: string }
export interface UpgradePromptProps { feature: string; requiredPlan: string; href?: string; className?: string }
export interface PendingSyncBadgeProps { pending: number; className?: string }
export interface EntitlementErrorCardProps { errorType: string; children?: ReactNode; className?: string }

export function SubscriptionStatusCard({ plan, status, credits, multiplier = 1, label = "credits" }: SubscriptionStatusCardProps) {
  return <section aria-label="Subscription status"><strong>{plan}</strong><span>{status}</span><span>{credits * multiplier} {label}</span></section>;
}

export function AuthSection({ signedIn, onSignIn, onSignOut }: AuthSectionProps) {
  return <section aria-label="Account"><button type="button" onClick={signedIn ? onSignOut : onSignIn}>{signedIn ? "Sign out" : "Sign in"}</button></section>;
}

export function UpgradePrompt({ feature, requiredPlan, href = "/account" }: UpgradePromptProps) {
  return <section role="alert"><p>{feature} requires {requiredPlan}.</p><a href={href}>Manage subscription</a></section>;
}

export function PendingSyncBadge({ pending }: PendingSyncBadgeProps) {
  return pending > 0 ? <span role="status">{pending} pending sync</span> : null;
}

export function EntitlementErrorCard({ errorType, children }: EntitlementErrorCardProps) {
  return <section role="alert" data-error-type={errorType}>{children ?? "Subscription access is temporarily unavailable."}</section>;
}

export function useEntitlement<T extends { features?: string[]; planRank?: number; stale?: boolean }>(lease: T | null) {
  return { lease, stale: Boolean(lease?.stale), hasFeature: (key: string) => Boolean(lease?.features?.includes(key)), atLeastRank: (rank: number) => (lease?.planRank ?? 0) >= rank };
}
