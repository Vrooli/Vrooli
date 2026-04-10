import { useEffect, useState } from "react";
import { AlertTriangle, RotateCcw } from "lucide-react";
import type { DocResetResponse } from "../../../shared/services/documentationApi";
import { Button } from "../../../shared/ui/button";
import { selectors } from "../../../consts/selectors";

export type ResetPanelProps = {
  canReset: boolean;
  defaults: { maxAgeDays: number; keepMinEntries: number };
  isBusy: boolean;
  resetError: string;
  result: DocResetResponse | null;
  onPreview: (config: { maxAgeDays: number; keepMinEntries: number }) => void;
  onApply: (config: { maxAgeDays: number; keepMinEntries: number }) => void;
};

export function ResetPanel({
  canReset,
  defaults,
  isBusy,
  resetError,
  result,
  onPreview,
  onApply,
}: ResetPanelProps) {
  const [maxAgeDays, setMaxAgeDays] = useState(defaults.maxAgeDays);
  const [keepMinEntries, setKeepMinEntries] = useState(defaults.keepMinEntries);

  useEffect(() => {
    setMaxAgeDays(defaults.maxAgeDays);
    setKeepMinEntries(defaults.keepMinEntries);
  }, [defaults.maxAgeDays, defaults.keepMinEntries]);

  if (!canReset) {
    return (
      <div className="ko-panel-inset p-4">
        <p className="ko-text-sm ko-subtle">Reset is available for PROBLEMS and PROGRESS docs only.</p>
      </div>
    );
  }

  return (
    <div className="ko-reset-panel" data-testid={selectors.viewer.resetPanel}>
      <div className="ko-reset-controls">
        <label className="ko-reset-label">
          Remove entries older than
          <span className="ko-reset-value">{maxAgeDays} days</span>
        </label>
        <input
          type="range"
          min={0}
          max={365}
          value={maxAgeDays}
          onChange={(event) => setMaxAgeDays(Number(event.target.value))}
          className="ko-reset-slider"
        />
        <div className="ko-reset-row">
          <label className="ko-reset-label">
            Keep at least
            <input
              type="number"
              min={0}
              value={keepMinEntries}
              onChange={(event) => setKeepMinEntries(Number(event.target.value))}
              className="ko-reset-input"
            />
            entries
          </label>
        </div>
        <div className="ko-reset-actions">
          <Button
            type="button"
            variant="outline"
            size="compact"
            onClick={() => onPreview({ maxAgeDays, keepMinEntries })}
            disabled={isBusy}
          >
            <RotateCcw className="h-4 w-4 mr-2" />
            Preview
          </Button>
          <Button
            type="button"
            variant="primary"
            size="compact"
            onClick={() => onApply({ maxAgeDays, keepMinEntries })}
            disabled={isBusy}
          >
            Apply Reset
          </Button>
        </div>
      </div>

      {resetError ? (
        <div className="ko-reset-error">
          <AlertTriangle className="h-4 w-4" />
          <span>{resetError}</span>
        </div>
      ) : null}

      {result ? (
        <div className="ko-reset-summary">
          <p className="ko-text-sm">
            Removed <strong>{result.removed_count}</strong> • Kept{" "}
            <strong>{result.kept_count}</strong>
          </p>
          {result.removed_entries && result.removed_entries.length > 0 ? (
            <ul className="ko-reset-list">
              {result.removed_entries.map((entry) => (
                <li key={entry}>{entry}</li>
              ))}
            </ul>
          ) : null}
          {result.new_content && result.preview_only ? (
            <details className="ko-reset-preview">
              <summary>Preview content</summary>
              <pre>{result.new_content}</pre>
            </details>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
