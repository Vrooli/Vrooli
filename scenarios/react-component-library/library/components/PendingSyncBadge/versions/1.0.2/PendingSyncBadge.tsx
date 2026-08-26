/**
 * @libraryId react-component-library:PendingSyncBadge
 * @displayName PendingSyncBadge
 * @description Durable Class B outbox backlog indicator.
 * @version 1.0.2
 * @tags ["monetization","outbox"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:PendingSyncBadge */
import { PendingSyncBadge as BasePendingSyncBadge } from "../../../MonetizationAccount/versions/1.0.0/MonetizationAccount";
export type PendingSyncBadgeProps = { pending: number; className?: string };
export function PendingSyncBadge(props: PendingSyncBadgeProps) {
  return <BasePendingSyncBadge {...props} />;
}
