/**
 * @libraryId react-component-library:EntitlementErrorCard
 * @displayName EntitlementErrorCard
 * @description Stable paid-surface error presentation.
 * @version 1.0.4
 * @tags ["monetization","errors"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:EntitlementErrorCard */
import { EntitlementErrorCard as BaseEntitlementErrorCard } from "@vrooli/react-component-library/MonetizationAccount/1";
import type { ReactNode } from "react";
export type EntitlementErrorCardProps = {
  errorType: string;
  children?: ReactNode;
  className?: string;
};
export const EntitlementErrorCard = withClassName(function EntitlementErrorCard(
  props: EntitlementErrorCardProps,
) {
  return (
    <BaseEntitlementErrorCard
      data-testid="monetization.entitlement-error-card"
      {...props}
    />
  );
});
