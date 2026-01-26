import { Search } from "lucide-react";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
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
        <Panel>
          <PanelHeader
            title="Semantic Search"
            description="Ask natural-language questions to locate related knowledge across all collections."
            icon={<Search className="h-5 w-5 ko-icon" />}
            className="mb-4"
          />
          <SearchPanelContainer />
        </Panel>
      </PageShell>
    </ErrorBoundary>
  );
}
