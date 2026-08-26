/**
 * @libraryId react-component-library:SubscriptionStatusCard
 * @displayName SubscriptionStatusCard
 * @description Hosted account status and lease-backed balance.
 * @version 1.0.2
 * @tags ["monetization","account"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:SubscriptionStatusCard */
import { SubscriptionStatusCard as BaseSubscriptionStatusCard } from "../../../MonetizationAccount/versions/1.0.0/MonetizationAccount";
export type SubscriptionStatusCardProps = {
  plan: string;
  status: "active" | "trialing" | "past_due" | "canceled" | "inactive";
  credits: number;
  multiplier?: number;
  label?: string;
  className?: string;
};
export function SubscriptionStatusCard(props: SubscriptionStatusCardProps) {
  return <BaseSubscriptionStatusCard {...props} />;
}
