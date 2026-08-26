/**
 * @libraryId react-component-library:ErrorState
 * @displayName ErrorState
 * @description A retryable error surface that explains failure without collapsing the surrounding product hierarchy.
 * @version 1.0.4
 * @tags ["feedback","async","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ErrorState */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import type { ReactNode } from "react";
import { AsyncBoundary } from "../../../AsyncBoundary/versions/1.0.0/AsyncBoundary";

export const ErrorState = withClassName(function ErrorState({
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
    <AsyncBoundary status="error" errorTitle={title} error={message} retry={onRetry}>
      {children}
    </AsyncBoundary>
  );
});
