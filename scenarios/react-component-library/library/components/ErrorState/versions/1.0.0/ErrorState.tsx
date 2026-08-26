/** @vrooliComponentSource react-component-library:ErrorState */
import type { ReactNode } from "react";
import { AsyncBoundary } from "@vrooli/react-component-library/AsyncBoundary/1.0.0";

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
