// DOC: docs/reference/api-endpoints.md#documentation-health
import { AlertCircle, RefreshCw, Wrench } from "lucide-react";
import { Button } from "../../../shared/ui/button";
import type { DocHealthViewModel, HealthTone } from "../../../shared/controllers/documentationController";

const toneClasses: Record<HealthTone, string> = {
  good: "ko-tone-good",
  medium: "ko-tone-medium",
  poor: "ko-tone-poor",
};

export type HealthPanelProps = {
  scenarioName: string | null;
  healthViewModel: DocHealthViewModel;
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
  onRefresh: () => void;
};

export function HealthPanel({
  scenarioName,
  healthViewModel,
  isLoading,
  hasError,
  errorMessage,
  onRefresh,
}: HealthPanelProps) {
  if (!scenarioName) {
    return <div className="ko-panel p-4 ko-text-sm ko-muted">Select a scenario to view health details.</div>;
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-10">
        <RefreshCw className="h-5 w-5 ko-icon animate-spin" />
        <span className="ml-3 ko-text-sm ko-muted">Loading doc health...</span>
      </div>
    );
  }

  if (hasError) {
    return (
      <div className="ko-alert ko-alert-danger">
        <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
        <div className="flex-1">
          <p className="ko-alert-title ko-text-danger-strong">Failed to load health</p>
          <p className="ko-text-sm ko-text-danger-muted mt-1">{errorMessage}</p>
          <Button onClick={onRefresh} variant="danger" className="mt-3">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const toneClass = toneClasses[healthViewModel.healthTone] ?? toneClasses.medium;

  return (
    <div className="ko-stack">
      <div className="flex items-center justify-between">
        <div>
          <p className="ko-text-sm ko-subtle">Scenario</p>
          <p className="font-semibold ko-text-strong">{scenarioName}</p>
        </div>
        <span className={`ko-health-badge ${toneClass}`}>{healthViewModel.healthScoreLabel}</span>
      </div>

      <div className="ko-card p-3 ko-text-xs ko-subtle">
        {healthViewModel.totalDocsLabel} • {healthViewModel.warningCount} warnings
      </div>

      {healthViewModel.hasIssues ? (
        <div className="ko-stack-sm">
          {healthViewModel.missingDocs.length > 0 && (
            <div className="ko-issue-block">
              <p className="ko-text-sm font-semibold ko-text-strong">Missing Docs</p>
              <ul className="ko-list">
                {healthViewModel.missingDocs.map((doc) => (
                  <li key={doc}>{doc}</li>
                ))}
              </ul>
            </div>
          )}

          {healthViewModel.misplacedDocs.length > 0 && (
            <div className="ko-issue-block">
              <p className="ko-text-sm font-semibold ko-text-strong">Misplaced Docs</p>
              <ul className="ko-list">
                {healthViewModel.misplacedDocs.map((doc) => (
                  <li key={`${doc.actualPath}-${doc.expectedPath}`}>
                    <span className="ko-text-strong">{doc.actualPath}</span> → {doc.expectedPath}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {healthViewModel.extraDocs.length > 0 && (
            <div className="ko-issue-block">
              <p className="ko-text-sm font-semibold ko-text-strong">Extra Docs</p>
              <ul className="ko-list">
                {healthViewModel.extraDocs.map((doc) => (
                  <li key={doc}>{doc}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      ) : (
        <div className="ko-panel p-4 ko-text-sm ko-muted">
          No documentation issues detected. Everything is aligned with the standard layout.
        </div>
      )}

      <div className="ko-card p-3 flex items-center justify-between">
        <div>
          <p className="ko-text-sm font-semibold ko-text-strong">Heal Documentation</p>
          <p className="ko-text-xs ko-muted">Automation unlocks in Phase 5.</p>
        </div>
        <Button variant="outline" size="sm" disabled>
          <Wrench className="h-4 w-4 mr-2" />
          Fix with Agent
        </Button>
      </div>
    </div>
  );
}
