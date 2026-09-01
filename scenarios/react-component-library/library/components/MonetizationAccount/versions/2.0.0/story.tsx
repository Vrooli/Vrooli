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
