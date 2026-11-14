import { ErrorFallback } from "../components/ErrorBoundary";

/** What the router shows when a route throws while rendering: the app's own fallback, never the library's raw error page. */
export function RouteError() {
  return <ErrorFallback onRetry={() => window.location.reload()} />;
}
