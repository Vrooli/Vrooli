import { useCallback, useEffect, useMemo, useState, type ChangeEvent, type FC } from "react";
import { ChevronDown, ChevronRight, ShieldCheck } from "lucide-react";
import type { ResilienceSettings } from '@/types/workflow';

interface ResiliencePanelProps {
  value?: ResilienceSettings;
  onChange: (next?: ResilienceSettings) => void;
}

const isEmptyConfig = (value?: ResilienceSettings): boolean => {
  if (!value) {
    return true;
  }
  return !Object.values(value).some((entry) =>
    typeof entry === "number" ? !Number.isNaN(entry) : Boolean(entry && String(entry).trim().length > 0),
  );
};

const numberInputValue = (value?: number) =>
  typeof value === "number" && !Number.isNaN(value) ? String(value) : "";

const ResiliencePanel: FC<ResiliencePanelProps> = ({ value, onChange }) => {
  const [expanded, setExpanded] = useState(() => !isEmptyConfig(value));

  useEffect(() => {
    if (!isEmptyConfig(value)) {
      setExpanded(true);
    }
  }, [value]);

  const updateField = useCallback(
    (field: keyof ResilienceSettings, nextValue?: number | string) => {
      const base: ResilienceSettings = { ...(value ?? {}) };
      if (nextValue === undefined || nextValue === "") {
        delete base[field];
      } else {
        base[field] = nextValue as never;
      }
      onChange(isEmptyConfig(base) ? undefined : base);
    },
    [onChange, value],
  );

  const handleIntegerChange = useCallback(
    (field: keyof ResilienceSettings, minimum = 0) =>
      (event: ChangeEvent<HTMLInputElement>) => {
        const raw = event.target.value;
        if (!raw.trim()) {
          updateField(field, undefined);
          return;
        }
        const parsed = Number(raw);
        if (Number.isNaN(parsed)) {
          return;
        }
        const normalized = Math.max(minimum, Math.round(parsed));
        updateField(field, normalized);
      },
    [updateField],
  );

  const handleFloatChange = useCallback(
    (field: keyof ResilienceSettings, minimum = 0) =>
      (event: ChangeEvent<HTMLInputElement>) => {
        const raw = event.target.value;
        if (!raw.trim()) {
          updateField(field, undefined);
          return;
        }
        const parsed = Number(raw);
        if (Number.isNaN(parsed)) {
          return;
        }
        const normalized = Math.max(minimum, parsed);
        updateField(field, normalized);
      },
    [updateField],
  );

  const handleStringChange = useCallback(
    (field: keyof ResilienceSettings) =>
      (event: ChangeEvent<HTMLInputElement>) => {
        const raw = event.target.value;
        updateField(field, raw.trim() ? raw : undefined);
      },
    [updateField],
  );

  const clearDisabled = useMemo(() => isEmptyConfig(value), [value]);

  return (
    <div className="border border-gray-800 rounded-lg bg-flow-bg/60 mt-3" data-testid="node-resilience-panel">
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className="w-full flex items-center justify-between px-3 py-2 text-xs text-gray-300 border-b border-gray-800 bg-flow-bg/60 hover:bg-flow-bg/80 transition-colors"
      >
        <span className="flex items-center gap-2">
          {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          <ShieldCheck size={14} className="text-emerald-400" />
          Resilience
        </span>
        <button
          type="button"
          className={`text-[11px] ${
            clearDisabled
              ? "text-gray-600 cursor-not-allowed"
              : "text-gray-400 hover:text-gray-100"
          }`}
          onClick={(event) => {
            event.stopPropagation();
            if (!clearDisabled) {
              onChange(undefined);
            }
          }}
          disabled={clearDisabled}
        >
          Reset
        </button>
      </button>

      {expanded && (
        <div className="px-3 py-3 space-y-3 text-[11px] text-gray-300">
          <p className="text-[10px] text-gray-500 leading-relaxed">
            Without overrides, BAS automatically waits for the node selector, retries up to three attempts with a 750 ms pause, and reuses the node timeout for gating. Override any field below—or set max attempts to 1—to change that behavior.
          </p>
          <div>
            <p className="uppercase tracking-wide text-[10px] text-gray-500 mb-1">Retry policy</p>
            <div className="grid grid-cols-3 gap-2">
              <input
                type="number"
                min={1}
                placeholder="Max attempts"
                className="px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={numberInputValue(value?.maxAttempts)}
                onChange={handleIntegerChange("maxAttempts", 1)}
              />
              <input
                type="number"
                min={0}
                placeholder="Delay (ms)"
                className="px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={numberInputValue(value?.delayMs)}
                onChange={handleIntegerChange("delayMs", 0)}
              />
              <input
                type="number"
                step="0.1"
                min={0}
                placeholder="Backoff"
                className="px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={numberInputValue(value?.backoffFactor)}
                onChange={handleFloatChange("backoffFactor", 0)}
              />
            </div>
          </div>

          <div>
            <p className="uppercase tracking-wide text-[10px] text-gray-500 mb-1">Precondition</p>
            <div className="space-y-2">
              <input
                type="text"
                placeholder="Wait for selector before running"
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={value?.preconditionSelector ?? ""}
                onChange={handleStringChange("preconditionSelector")}
              />
              <div className="grid grid-cols-2 gap-2">
                <input
                  type="number"
                  min={0}
                  placeholder="Timeout (ms)"
                  className="px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                  value={numberInputValue(value?.preconditionTimeoutMs)}
                  onChange={handleIntegerChange("preconditionTimeoutMs", 0)}
                />
                <input
                  type="number"
                  min={0}
                  placeholder="Wait after ready (ms)"
                  className="px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                  value={numberInputValue(value?.preconditionWaitMs)}
                  onChange={handleIntegerChange("preconditionWaitMs", 0)}
                />
              </div>
            </div>
          </div>

          <div>
            <p className="uppercase tracking-wide text-[10px] text-gray-500 mb-1">Success validation</p>
            <div className="space-y-2">
              <input
                type="text"
                placeholder="Verify selector after action"
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={value?.successSelector ?? ""}
                onChange={handleStringChange("successSelector")}
              />
              <div className="grid grid-cols-2 gap-2">
                <input
                  type="number"
                  min={0}
                  placeholder="Timeout (ms)"
                  className="px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                  value={numberInputValue(value?.successTimeoutMs)}
                  onChange={handleIntegerChange("successTimeoutMs", 0)}
                />
                <input
                  type="number"
                  min={0}
                  placeholder="Wait after success (ms)"
                  className="px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                  value={numberInputValue(value?.successWaitMs)}
                  onChange={handleIntegerChange("successWaitMs", 0)}
                />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ResiliencePanel;
