/**
 * @libraryId react-component-library:OfflineState
 * @displayName OfflineState
 * @description An actionable offline surface that distinguishes unavailable connectivity from an application failure.
 * @version 1.0.3
 * @tags ["feedback","async","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:OfflineState */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ReactNode } from "react";
import { AsyncBoundary } from "@vrooli/react-component-library/AsyncBoundary/1.0.0";

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
