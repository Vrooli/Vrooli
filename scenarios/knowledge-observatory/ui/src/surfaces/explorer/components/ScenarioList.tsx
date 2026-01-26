// DOC: docs/reference/api-endpoints.md#scenario-list
import { RefreshCw, Search } from "lucide-react";
import { Button } from "../../../shared/ui/button";
import { selectors } from "../../../consts/selectors";
import type { ScenarioSummaryView } from "../../../shared/controllers/documentationController";

const toneClasses: Record<ScenarioSummaryView["healthTone"], string> = {
  good: "ko-tone-good",
  medium: "ko-tone-medium",
  poor: "ko-tone-poor",
};

export type ScenarioListProps = {
  scenarios: ScenarioSummaryView[];
  filter: string;
  onFilterChange: (value: string) => void;
  selectedScenario: string | null;
  onSelectScenario: (name: string) => void;
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
  onRefresh: () => void;
};

export function ScenarioList({
  scenarios,
  filter,
  onFilterChange,
  selectedScenario,
  onSelectScenario,
  isLoading,
  hasError,
  errorMessage,
  onRefresh,
}: ScenarioListProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <RefreshCw className="h-5 w-5 ko-icon animate-spin" />
        <span className="ml-3 ko-text-sm ko-muted">Loading scenarios...</span>
      </div>
    );
  }

  if (hasError) {
    return (
      <div className="ko-alert ko-alert-danger">
        <div className="flex-1">
          <p className="ko-alert-title ko-text-danger-strong">Failed to load scenarios</p>
          <p className="ko-text-sm ko-text-danger-muted mt-1">{errorMessage}</p>
          <Button onClick={onRefresh} variant="danger" className="mt-3">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="ko-stack-sm" data-testid={selectors.explorer.scenarioList}>
      <div className="ko-input-group">
        <Search className="h-4 w-4 ko-icon-muted" />
        <input
          className="ko-input ko-input-compact"
          placeholder="Filter scenarios"
          value={filter}
          onChange={(event) => onFilterChange(event.target.value)}
          data-testid={selectors.explorer.scenarioFilter}
        />
        <Button onClick={onRefresh} variant="outline" size="compact" aria-label="Refresh scenarios">
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      {scenarios.length === 0 ? (
        <div className="ko-panel p-4 text-center ko-text-sm ko-muted">No scenarios found.</div>
      ) : (
        <div className="ko-stack-sm">
          {scenarios.map((scenario) => {
            const isActive = scenario.name === selectedScenario;
            const toneClass = toneClasses[scenario.healthTone] ?? toneClasses.medium;
            return (
              <button
                key={scenario.name}
                type="button"
                className={[
                  "ko-card ko-card-interactive ko-scenario-item",
                  isActive ? "ko-scenario-item-active" : "",
                ].join(" ")}
                onClick={() => onSelectScenario(scenario.name)}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold ko-text-strong">{scenario.name}</p>
                    <p className="ko-text-xs ko-subtle mt-1">{scenario.path}</p>
                  </div>
                  <span className={`ko-health-badge ${toneClass}`}>{scenario.healthScoreLabel}</span>
                </div>
                <div className="mt-3 flex flex-wrap items-center gap-2 ko-text-xs ko-muted">
                  <span>{scenario.docCountLabel}</span>
                  <span className={scenario.hasReadme ? "ko-pill ko-pill-good" : "ko-pill ko-pill-muted"}>
                    README
                  </span>
                  <span className={scenario.hasManifest ? "ko-pill ko-pill-good" : "ko-pill ko-pill-muted"}>
                    Manifest
                  </span>
                  <span className="ko-pill">Updated {scenario.lastModifiedLabel}</span>
                </div>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
