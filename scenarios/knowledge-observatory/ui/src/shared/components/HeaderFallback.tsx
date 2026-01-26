// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import { Button } from "../ui/button";
import type { Route } from "../controllers/routeController";

export type HeaderFallbackProps = {
  errorMessage: string;
  onRetry: () => void;
  onNavigate: (route: Route) => void;
};

export function HeaderFallback({ errorMessage, onRetry, onNavigate }: HeaderFallbackProps) {
  return (
    <header className="ko-app-header">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Knowledge Observatory</h1>
          <p className="ko-text-sm ko-subtle">Header temporarily unavailable.</p>
        </div>
        <Button onClick={onRetry}>Retry Header</Button>
      </div>
      <p className="ko-text-xs ko-text-danger-muted mt-2">{errorMessage}</p>
      <div className="mt-4 flex flex-wrap items-center gap-2">
        <Button variant="secondary" onClick={() => onNavigate("dashboard")}>
          Dashboard
        </Button>
        <Button variant="secondary" onClick={() => onNavigate("search")}>
          Search
        </Button>
        <Button variant="secondary" onClick={() => onNavigate("graph")}>
          Graph
        </Button>
        <Button variant="secondary" onClick={() => onNavigate("metrics")}>
          Metrics
        </Button>
      </div>
    </header>
  );
}
