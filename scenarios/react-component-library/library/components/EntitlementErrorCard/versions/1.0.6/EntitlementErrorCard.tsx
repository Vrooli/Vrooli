/**
 * @libraryId react-component-library:EntitlementErrorCard
 * @displayName EntitlementErrorCard
 * @description An explicit recovery-oriented entitlement error surface.
 * @version 1.0.6
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:EntitlementErrorCard */
import { EntitlementErrorCard as BaseEntitlementErrorCard } from "@vrooli/react-component-library/MonetizationAccount/2";
import type { ReactNode } from "react";
export type EntitlementErrorCardProps = {
  errorType: string;
  children?: ReactNode;
  className?: string;
};
export const EntitlementErrorCard = withClassName(function EntitlementErrorCard(
  props: EntitlementErrorCardProps,
) {
  return <BaseEntitlementErrorCard data-testid="monetization.entitlement-error-card" {...props} />;
});
