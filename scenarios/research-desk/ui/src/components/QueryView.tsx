import type { UseQueryResult } from "@tanstack/react-query";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { ErrorState } from "@vrooli/react-component-library/ErrorState/1";
import { LoadingState } from "@vrooli/react-component-library/LoadingState/1";
import type { ReactNode } from "react";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage, failureKind } from "../lib/errorMessage";
import { absolute } from "../lib/format";

/**
 * The states every read has: loading, failed (with the failure named: desk
 * unreachable, not authorized, or the API's reason), stale (an earlier read is
 * kept on screen, marked with its time, when a later refresh fails) and
 * loaded. A page renders only the loaded branch itself.
 */
export function QueryView<T>({ query, label, children }: { query: UseQueryResult<T>; label?: string; children: (data: T) => ReactNode }) {
  const { t } = useTranslation();
  if (query.isPending) return <LoadingState label={label ?? t(strings.common.loading)} />;
  if (query.isLoadingError) return <QueryError error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <>
      {query.isRefetchError ? (
        <Alert
          tone="warning"
          title={t(strings.errors.staleTitle)}
          description={t(strings.errors.stale, { time: absolute(new Date(query.dataUpdatedAt)), reason: errorMessage(query.error, t) })}
        />
      ) : null}
      {children(query.data)}
    </>
  );
}

export function QueryError({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const { t } = useTranslation();
  const kind = failureKind(error);
  const title = kind === "unauthorized" ? t(strings.errors.unauthorizedTitle) : kind === "unreachable" ? t(strings.errors.unreachableTitle) : t(strings.errors.refusedTitle);
  return <ErrorState title={title} message={errorMessage(error, t)} onRetry={onRetry} />;
}

/** An inline failure for an action: the request was not carried out, and why. */
export function ActionError({ error }: { error: unknown }) {
  const { t } = useTranslation();
  if (!error) return null;
  return <Alert tone="danger" title={t(strings.errors.notDone)} description={errorMessage(error, t)} />;
}
