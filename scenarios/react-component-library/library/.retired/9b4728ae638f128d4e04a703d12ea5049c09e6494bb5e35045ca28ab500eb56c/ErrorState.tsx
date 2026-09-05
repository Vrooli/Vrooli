/** @vrooliComponentSource react-component-library:ErrorState */
import type { ReactNode } from "react";
import { AsyncBoundary } from "../../../AsyncBoundary/versions/1.0.0/AsyncBoundary";

export function ErrorState({
  title = "Something went wrong",
  message = "The operation could not be completed.",
  onRetry,
  children,
}: {
  title?: ReactNode;
  message?: ReactNode;
  onRetry?: () => void | Promise<void>;
  children?: ReactNode;
}) {
  return (
    <AsyncBoundary
      status="error"
      errorTitle={title}
      error={message}
      retry={onRetry}
    >
      {children}
    </AsyncBoundary>
  );
}
