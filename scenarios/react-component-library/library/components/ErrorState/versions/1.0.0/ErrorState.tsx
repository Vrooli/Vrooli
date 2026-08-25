/** @vrooliComponentSource react-component-library:ErrorState */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

import type { ReactNode } from "react";
import { AsyncBoundary } from "../../../AsyncBoundary/versions/1.0.0/AsyncBoundary";

export function ErrorState({
  title = translate("feedback.error-state.title.1", "Something went wrong"),
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
