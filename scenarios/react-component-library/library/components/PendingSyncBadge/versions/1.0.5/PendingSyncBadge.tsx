/**
 * @libraryId react-component-library:PendingSyncBadge
 * @displayName PendingSyncBadge
 * @version 1.0.5
 * @tags ["monetization","outbox"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:PendingSyncBadge */
import { PendingSyncBadge as BasePendingSyncBadge } from "@vrooli/react-component-library/MonetizationAccount/2";
export type PendingSyncBadgeProps = { pending: number; className?: string };
export const PendingSyncBadge = withClassName(function PendingSyncBadge(
  props: PendingSyncBadgeProps,
) {
  return <BasePendingSyncBadge data-testid="monetization.pending-sync-badge" {...props} />;
});
