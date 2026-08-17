import { AuthSection as AccountAuthSection } from "./MonetizationAccount";
import { SubscriptionBadge } from "./MonetizationAccount";

export function AuthSection() {
  return (
    <AccountAuthSection
      signedIn={false}
      onSignIn={() => undefined}
      onSignOut={() => undefined}
    />
  );
}

export function BadgeStory() {
  return <SubscriptionBadge plan="pro" status="active" credits={42} />;
}
