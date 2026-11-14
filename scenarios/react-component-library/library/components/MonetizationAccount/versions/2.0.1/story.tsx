import {
  AuthSection as AccountAuthSection,
  EntitlementErrorCard,
  UsageMeter,
} from "./MonetizationAccount";

export function AuthSection() {
  return (
    <AccountAuthSection signedIn={false} onSignIn={() => undefined} onSignOut={() => undefined} />
  );
}

export function MonetizationStates() {
  return (
    <div>
      <UsageMeter used={7} limit={10} />
      <EntitlementErrorCard errorType="unauthorized" />
      <EntitlementErrorCard errorType="subscription_required" />
      <EntitlementErrorCard errorType="credits_required" />
      <EntitlementErrorCard errorType="authority_unavailable" />
      <EntitlementErrorCard errorType="rate_limited" />
      <EntitlementErrorCard errorType="rank_required" />
      <EntitlementErrorCard errorType="unknown" />
    </div>
  );
}

export function UsageState() {
  return <UsageMeter used={7} limit={10} />;
}

const errorState = (errorType: string) => () => <EntitlementErrorCard errorType={errorType} />;

export const UnauthorizedState = errorState("unauthorized");
export const SubscriptionRequiredState = errorState("subscription_required");
export const CreditsRequiredState = errorState("credits_required");
export const AuthorityUnavailableState = errorState("authority_unavailable");
export const RateLimitedState = errorState("rate_limited");
export const RankRequiredState = errorState("rank_required");
export const UnknownErrorState = errorState("unknown");
