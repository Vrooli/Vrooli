import { Database } from "lucide-react";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import type { Route } from "../../shared/controllers/routeController";
import { MetricsPanelContainer } from "./MetricsPanelContainer";

export type CollectionDetailsPageProps = {
  collectionName: string;
  onNavigate: (route: Route) => void;
  onOpenCollection: (collectionName: string) => void;
};

export function CollectionDetailsPage({ collectionName, onNavigate, onOpenCollection }: CollectionDetailsPageProps) {
  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <PageShell>
          <SectionErrorState
            title="Collection Details Unavailable"
            description="The collection details UI failed to render. You can retry this section or return to metrics."
            errorMessage={error.message}
            actions={[
              { label: "Retry Section", onClick: reset },
              { label: "Back to Metrics", onClick: () => onNavigate("metrics"), variant: "secondary" },
            ]}
          />
        </PageShell>
      )}
    >
      <PageShell>
        <Panel>
          <PanelHeader
            title={collectionName.trim() ? `Collection: ${collectionName}` : "Collection Details"}
            description="Full diagnostics and maintenance controls for debugging embeddings, chunking, and ingest health."
            icon={<Database className="h-5 w-5 ko-icon" />}
            className="mb-4"
          />
          <MetricsPanelContainer
            mode="details"
            initialCollection={collectionName}
            onOpenCollection={onOpenCollection}
            onBackToMetrics={() => onNavigate("metrics")}
          />
        </Panel>
      </PageShell>
    </ErrorBoundary>
  );
}
