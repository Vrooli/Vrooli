// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { SearchMode, SearchModeOption } from "../controllers/searchModes";
import { SEARCH_MODE_OPTIONS } from "../controllers/searchModes";
import { Button } from "../ui/button";

export type SearchModeSelectorProps = {
  mode: SearchMode;
  onChange: (mode: SearchMode) => void;
  options?: SearchModeOption[];
  compact?: boolean;
  showDescriptions?: boolean;
  testId?: string;
};

export function SearchModeSelector({
  mode,
  onChange,
  options,
  compact = false,
  showDescriptions = false,
  testId,
}: SearchModeSelectorProps) {
  const resolvedOptions = options ?? SEARCH_MODE_OPTIONS;

  return (
    <div className={"ko-mode-selector"} data-testid={testId}>
      {resolvedOptions.map((option) => {
        const isActive = option.mode === mode;
        const buttonTestId = testId ? `${testId}-${option.mode}` : undefined;
        return (
          <Button
            key={option.mode}
            type="button"
            variant={isActive ? "primary" : "secondary"}
            size={compact ? "sm" : "default"}
            className={`ko-mode-button${compact ? " ko-mode-button-compact" : ""}`}
            data-testid={buttonTestId}
            onClick={() => onChange(option.mode)}
          >
            <span className="flex flex-col items-start">
              <span className="ko-text-sm font-semibold">{option.label}</span>
              {showDescriptions && (
                <span className="ko-text-xs ko-subtle mt-0.5">{option.description}</span>
              )}
            </span>
          </Button>
        );
      })}
    </div>
  );
}
