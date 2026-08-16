/**
 * @vrooliComponentSource react-component-library:MonetizationAccount
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption d3e8da31-4e4d-4ac4-a555-4fe89b6c91a5
 * @vrooliComponentAppliedAt 2026-08-16T08:48:57Z
 * @vrooliComponentSourceSha256 5c6775a0860af0416d74190540056828a899655a26779e49cc9de1e54dd5aafa
 * @vrooliComponentDriftHash 5c6775a0860af0416d74190540056828a899655a26779e49cc9de1e54dd5aafa
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";

export type EntitlementStatus = "active" | "trialing" | "past_due" | "canceled" | "inactive";

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

export function EntitlementErrorCard({ errorType, children }: { errorType: string; children?: ReactNode }) {
  return <section role="alert" data-error-type={errorType}>{children ?? "Subscription access is temporarily unavailable."}</section>;
}

export function useEntitlement<T extends { features?: string[]; planRank?: number; stale?: boolean }>(lease: T | null) {
  return { lease, stale: Boolean(lease?.stale), hasFeature: (key: string) => Boolean(lease?.features?.includes(key)), atLeastRank: (rank: number) => (lease?.planRank ?? 0) >= rank };
}
