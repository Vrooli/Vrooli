import { SubscriptionStatusCard } from "./SubscriptionStatusCard";

export function StatusStory() {
  return <SubscriptionStatusCard plan="pro" status="active" credits={42} />;
}
