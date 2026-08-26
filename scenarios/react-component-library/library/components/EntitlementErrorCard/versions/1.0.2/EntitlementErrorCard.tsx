/**
 * @libraryId react-component-library:EntitlementErrorCard
 * @displayName EntitlementErrorCard
 * @description Stable paid-surface error presentation.
 * @version 1.0.2
 * @tags ["monetization","errors"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:EntitlementErrorCard */
import { EntitlementErrorCard as BaseEntitlementErrorCard } from "../../../MonetizationAccount/versions/1.0.0/MonetizationAccount";
import type { ReactNode } from "react";
export type EntitlementErrorCardProps = {
  errorType: string;
  children?: ReactNode;
  className?: string;
};
export function EntitlementErrorCard(props: EntitlementErrorCardProps) {
  return <BaseEntitlementErrorCard {...props} />;
}
