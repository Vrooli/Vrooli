import type { ReactNode } from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

interface QueryStateProps {
  isLoading: boolean;
  error: unknown;
  /** True when the query resolved but produced no rows. */
  isEmpty?: boolean;
  /** Override copy for each state; defaults to the shared `common.*` strings. */
  loadingLabel?: string;
  errorLabel?: string;
  emptyLabel?: string;
  children: ReactNode;
}

/**
 * QueryState renders the canonical loading / error / empty fallbacks for a
 * react-query result, or its children when data is present. Every surface
 * routes its query results through this so the three states look and read the
 * same across the app and so tests have stable selectors to assert on.
 *
 * Pass `error` straight from `useQuery`; it is normalised through
 * `errorMessage` (Connect codes → translated copy) when no `errorLabel`
 * override is given.
 */
export function QueryState({
  isLoading,
  error,
  isEmpty = false,
  loadingLabel,
  errorLabel,
  emptyLabel,
  children,
}: QueryStateProps) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <p data-testid={selectors.queryState.loading} className="text-sm text-app-muted-foreground">
        {loadingLabel ?? t(strings.common.loading)}
      </p>
    );
  }

  if (error) {
    return (
      <p data-testid={selectors.queryState.error} role="alert" className="text-sm text-app-danger">
        {errorLabel ?? errorMessage(error, t)}
      </p>
    );
  }

  if (isEmpty) {
    return (
      <p data-testid={selectors.queryState.empty} className="text-sm text-app-muted-foreground">
        {emptyLabel ?? t(strings.common.empty)}
      </p>
    );
  }

  return <>{children}</>;
}
