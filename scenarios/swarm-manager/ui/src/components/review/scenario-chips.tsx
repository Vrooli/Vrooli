/**
 * ScenarioChips — Pure display of target scenario name chips.
 *
 * Extracted from ScenarioReviewResults which mixed display + service calls.
 */

import { selectors } from "../../consts/selectors";

export interface ScenarioChipsProps {
  scenarios: string[];
  onSelect: (name: string) => void;
}

export function ScenarioChips({ scenarios, onSelect }: ScenarioChipsProps) {
  if (scenarios.length === 0) return null;

  return (
    <div data-testid={selectors.review.scenarioChips}>
      <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wider text-slate-500">
        Target Scenarios
      </p>
      <div className="flex flex-wrap gap-1.5">
        {scenarios.map((name) => (
          <button
            key={name}
            type="button"
            onClick={() => onSelect(name)}
            className="inline-flex items-center rounded-full bg-violet-500/15 px-2.5 py-1 text-xs font-medium text-violet-400 transition-colors hover:bg-violet-500/25"
          >
            {name}
          </button>
        ))}
      </div>
    </div>
  );
}
