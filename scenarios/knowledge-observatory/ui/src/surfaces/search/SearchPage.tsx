// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
// DOC: docs/guides/getting-started.md#ui-walkthrough
import { Search, Sparkles } from "lucide-react";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import type { Route } from "../../shared/controllers/routeController";
import { DeepSearchPanelContainer } from "./DeepSearchPanelContainer";
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
        <Panel className="mt-6">
          <PanelHeader
            title="Deep Documentation Search"
            description="Spawn an agent to explore docs, follow references, and return ranked results."
            icon={<Sparkles className="h-5 w-5 ko-icon" />}
            className="mb-4"
          />
          <DeepSearchPanelContainer />
        </Panel>
      </PageShell>
    </ErrorBoundary>
  );
}
