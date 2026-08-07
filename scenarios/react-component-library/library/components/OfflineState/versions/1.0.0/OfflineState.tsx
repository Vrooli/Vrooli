/** @vrooliComponentSource react-component-library:OfflineState */
import type { ReactNode } from "react";
import { AsyncBoundary } from "../../../AsyncBoundary/versions/1.0.0/AsyncBoundary";

export function OfflineState({
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
}
