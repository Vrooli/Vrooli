import type { ReactNode } from "react";
import { LoadingState } from "@vrooli/react-component-library/LoadingState/1";
import { ErrorNotice } from "./ErrorNotice";

interface DataStateProps {
  loading: boolean;
  error?: unknown;
  /** True when there is nothing to show yet (no rows, no record). */
  isEmpty: boolean;
  /** Rendered only when the load succeeded and returned nothing. */
  empty: ReactNode;
  /** What is being loaded, e.g. "Runs"; used in loading and error copy. */
  subject: string;
  onRetry?: () => void | Promise<void>;
  /** Dense contexts (cards, side panels) get one-line loading and error rows. */
  compact?: boolean;
  /** Placeholder shaped like the loaded content; replaces the spinner on a first load. */
  skeleton?: ReactNode;
  children: ReactNode;
}

/**
 * The one loading / error / empty decision. An empty state is shown only after
 * a successful load, so an unreachable API never reads as "nothing here yet".
 * When a refresh fails but earlier data exists, the data stays with a compact
 * error above it.
 */
export function DataState({ loading, error, isEmpty, empty, subject, onRetry, compact = false, skeleton, children }: DataStateProps) {
  if (isEmpty && error) {
    return compact ? (
      <div className="p-3"><ErrorNotice compact error={error} subject={subject} onRetry={onRetry} /></div>
    ) : (
      <div className="p-3"><ErrorNotice error={error} subject={subject.toLowerCase()} onRetry={onRetry} /></div>
    );
  }
  if (isEmpty && loading && skeleton) {
    return <div role="status" aria-busy="true" aria-label={`Loading ${subject.toLowerCase()}…`} data-testid="data-state-skeleton">{skeleton}</div>;
  }
  if (isEmpty && loading) {
    return compact ? (
      <p role="status" className="flex items-center gap-2 px-4 py-6 text-sm text-muted-foreground">
        <span aria-hidden="true" className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        Loading {subject.toLowerCase()}…
      </p>
    ) : (
      <div className="p-3"><LoadingState label={`Loading ${subject.toLowerCase()}…`} /></div>
    );
  }
  if (isEmpty) return <>{empty}</>;
  return (
    <>
      {error ? <div className="px-3 pt-3"><ErrorNotice compact error={error} subject={subject} onRetry={onRetry} /></div> : null}
      {children}
    </>
  );
}
