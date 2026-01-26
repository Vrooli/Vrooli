import { Search } from "lucide-react";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import type { Route } from "../../shared/controllers/routeController";
import { SearchPanelContainer } from "./SearchPanelContainer";

export type SearchPageProps = {
  onNavigate: (route: Route) => void;
};

export function SearchPage({ onNavigate }: SearchPageProps) {
  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <PageShell>
          <SectionErrorState
            title="Search Panel Unavailable"
            description="The search UI encountered an unexpected error. You can retry or return to the dashboard."
            errorMessage={error.message}
            actions={[
              { label: "Retry Section", onClick: reset },
              { label: "Back to Dashboard", onClick: () => onNavigate("dashboard"), variant: "secondary" },
            ]}
          />
        </PageShell>
      )}
    >
      <PageShell>
        <section className="ko-panel ko-section">
          <div className="flex items-center gap-2 mb-2">
            <Search className="h-5 w-5 text-green-500" />
            <h2 className="ko-text-lg font-semibold">Semantic Search</h2>
          </div>
          <p className="ko-text-sm ko-muted mb-4">
            Ask natural-language questions to locate related knowledge across all collections.
          </p>
          <SearchPanelContainer />
        </section>
      </PageShell>
    </ErrorBoundary>
  );
}
