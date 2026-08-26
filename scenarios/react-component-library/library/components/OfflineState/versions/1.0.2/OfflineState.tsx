/**
 * @libraryId react-component-library:OfflineState
 * @displayName OfflineState
 * @description An actionable offline surface that distinguishes unavailable connectivity from an application failure.
 * @version 1.0.2
 * @tags ["feedback","async","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:OfflineState */
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import type { ReactNode } from "react";
import { AsyncBoundary } from "../../../AsyncBoundary/versions/1.0.0/AsyncBoundary";

export const OfflineState = withClassName(function OfflineState({
  onRetry,
  children,
}: {
  onRetry?: () => void | Promise<void>;
  children?: ReactNode;
}) {
  return (
    <AsyncBoundary status="offline" retry={onRetry}>
      {children}
    </AsyncBoundary>
  );
});
