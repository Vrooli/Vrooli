// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { FormEvent } from "react";
import { Search } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import type { SearchMode } from "../../../shared/controllers/searchModes";
import { SearchModeSelector } from "../../../shared/components/SearchModeSelector";
import { Button } from "../../../shared/ui/button";

export type QuickSearchPanelProps = {
  mode: SearchMode;
  query: string;
  onModeChange: (mode: SearchMode) => void;
  onQueryChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  isSubmitDisabled: boolean;
};

export function QuickSearchPanel({
  mode,
  query,
  onModeChange,
  onQueryChange,
  onSubmit,
  isSubmitDisabled,
}: QuickSearchPanelProps) {
  return (
    <form onSubmit={onSubmit} className="ko-stack-sm" data-testid={selectors.dashboard.quickSearchForm}>
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h2 className="ko-text-lg font-semibold">Quick Search</h2>
          <p className="ko-text-sm ko-muted mt-1">
            Jump into documentation or semantic knowledge queries without leaving the dashboard.
          </p>
        </div>
        <SearchModeSelector
          mode={mode}
          onChange={onModeChange}
          compact
          testId={selectors.dashboard.quickSearchMode}
        />
      </div>
      <div className="flex flex-col md:flex-row gap-3">
        <div className="flex-1">
          <input
            type="text"
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="Search for docs, scenarios, or knowledge…"
            className="ko-input"
            data-testid={selectors.dashboard.quickSearchInput}
          />
          <p className="ko-input-help">Press search to open the full workspace with your query preloaded.</p>
        </div>
        <Button
          type="submit"
          variant="primary"
          data-testid={selectors.dashboard.quickSearchSubmit}
          disabled={isSubmitDisabled}
        >
          <Search className="h-4 w-4" />
          <span className="ml-2">Search</span>
        </Button>
      </div>
    </form>
  );
}
