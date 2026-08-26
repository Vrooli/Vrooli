/**
 * @libraryId react-component-library:ErrorState
 * @displayName ErrorState
 * @description
 * @version 1.0.7
 * @tags ["feedback","async","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ErrorState */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ReactNode } from "react";
import { AsyncBoundary } from "@vrooli/react-component-library/AsyncBoundary/1.0.0";

export const ErrorState = withClassName(function ErrorState({
  title,
  message = "The operation could not be completed.",
  onRetry,
  children,
}: {
  title?: ReactNode;
  message?: ReactNode;
  onRetry?: () => void | Promise<void>;
  children?: ReactNode;
}) {
  const libraryStrings = useStrings();
  title =
    title ?? libraryStrings("feedback.error-state.something-went-wrong", "Something went wrong");
  return (
    <AsyncBoundary
      data-testid="feedback.error-state"
      status="error"
      errorTitle={title}
      error={message}
      retry={onRetry}
    >
      {children}
    </AsyncBoundary>
  );
});
