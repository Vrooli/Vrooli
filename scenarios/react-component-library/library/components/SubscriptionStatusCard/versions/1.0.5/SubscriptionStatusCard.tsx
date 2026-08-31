/**
 * @libraryId react-component-library:SubscriptionStatusCard
 * @displayName SubscriptionStatusCard
 * @description Hosted account status and lease-backed balance.
 * @version 1.0.5
 * @tags ["monetization","account"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:SubscriptionStatusCard */
import { SubscriptionStatusCard as BaseSubscriptionStatusCard } from "@vrooli/react-component-library/MonetizationAccount/1";
export type SubscriptionStatusCardProps = {
  plan: string;
  status: "active" | "trialing" | "past_due" | "canceled" | "inactive";
  credits: number;
  multiplier?: number;
  label?: string;
  className?: string;
};
export const SubscriptionStatusCard = withClassName(function SubscriptionStatusCard(
  props: SubscriptionStatusCardProps,
) {
  return (
    <BaseSubscriptionStatusCard data-testid="monetization.subscription-status-card" {...props} />
  );
});
