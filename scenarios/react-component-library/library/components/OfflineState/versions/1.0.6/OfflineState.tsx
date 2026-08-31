/**
 * @libraryId react-component-library:OfflineState
 * @displayName OfflineState
 * @description The connectivity composition explaining what still works offline, what has been queued, when the surface last synchronized, and what reconnection will do.
 * @version 1.0.6
 * @tags ["feedback","async","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:OfflineState */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
import { AsyncBoundary } from "@vrooli/react-component-library/AsyncBoundary/1";

export const OfflineState = withClassName(function OfflineState({
  onRetry,
  children,
}: {
  onRetry?: () => void | Promise<void>;
  children?: ReactNode;
}) {
  return (
    <AsyncBoundary data-testid="feedback.offline-state" status="offline" retry={onRetry}>
      {children}
    </AsyncBoundary>
  );
});
