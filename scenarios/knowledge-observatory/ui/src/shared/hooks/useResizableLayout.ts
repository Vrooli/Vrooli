import { useState, useEffect, useCallback, useRef } from "react";

const STORAGE_KEYS = {
  scenarioColumnWidth: "ko.scenarioColumnWidth",
  viewerSplitRatio: "ko.viewerSplitRatio",
} as const;

const DEFAULTS = {
  scenarioColumnWidth: 300,
  viewerSplitRatio: 0.5,
} as const;

const CONSTRAINTS = {
  scenarioColumnWidth: { min: 200, max: 400 },
  viewerSplitRatio: { min: 0.25, max: 0.75 },
} as const;

function loadFromStorage<T>(key: string, defaultValue: T): T {
  if (typeof window === "undefined") return defaultValue;
  try {
    const stored = localStorage.getItem(key);
    if (stored === null) return defaultValue;
    const parsed = JSON.parse(stored) as T;
    return parsed;
  } catch {
    return defaultValue;
  }
}

function saveToStorage(key: string, value: number): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Storage might be full or disabled - ignore
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

export interface ResizableLayoutState {
  scenarioColumnWidth: number;
  viewerSplitRatio: number;
}

export interface UseResizableLayoutReturn {
  state: ResizableLayoutState;
  setScenarioColumnWidth: (width: number) => void;
  setViewerSplitRatio: (ratio: number) => void;
  handleScenarioResize: (deltaX: number) => void;
  handleViewerResize: (deltaX: number, containerWidth: number) => void;
  resetToDefaults: () => void;
}

export function useResizableLayout(): UseResizableLayoutReturn {
  const [state, setState] = useState<ResizableLayoutState>(() => ({
    scenarioColumnWidth: loadFromStorage(
      STORAGE_KEYS.scenarioColumnWidth,
      DEFAULTS.scenarioColumnWidth
    ),
    viewerSplitRatio: loadFromStorage(
      STORAGE_KEYS.viewerSplitRatio,
      DEFAULTS.viewerSplitRatio
    ),
  }));

  // Track initial load to avoid saving on mount
  const isInitialMount = useRef(true);

  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }
    saveToStorage(STORAGE_KEYS.scenarioColumnWidth, state.scenarioColumnWidth);
    saveToStorage(STORAGE_KEYS.viewerSplitRatio, state.viewerSplitRatio);
  }, [state.scenarioColumnWidth, state.viewerSplitRatio]);

  const setScenarioColumnWidth = useCallback((width: number) => {
    const clamped = clamp(
      width,
      CONSTRAINTS.scenarioColumnWidth.min,
      CONSTRAINTS.scenarioColumnWidth.max
    );
    setState((prev) => ({ ...prev, scenarioColumnWidth: clamped }));
  }, []);

  const setViewerSplitRatio = useCallback((ratio: number) => {
    const clamped = clamp(
      ratio,
      CONSTRAINTS.viewerSplitRatio.min,
      CONSTRAINTS.viewerSplitRatio.max
    );
    setState((prev) => ({ ...prev, viewerSplitRatio: clamped }));
  }, []);

  const handleScenarioResize = useCallback((deltaX: number) => {
    setState((prev) => {
      const newWidth = prev.scenarioColumnWidth + deltaX;
      const clamped = clamp(
        newWidth,
        CONSTRAINTS.scenarioColumnWidth.min,
        CONSTRAINTS.scenarioColumnWidth.max
      );
      return { ...prev, scenarioColumnWidth: clamped };
    });
  }, []);

  const handleViewerResize = useCallback((deltaX: number, containerWidth: number) => {
    if (containerWidth <= 0) return;
    setState((prev) => {
      // deltaX is the pixel change; convert to ratio change
      const currentCodeWidth = prev.viewerSplitRatio * containerWidth;
      const newCodeWidth = currentCodeWidth + deltaX;
      const newRatio = newCodeWidth / containerWidth;
      const clamped = clamp(
        newRatio,
        CONSTRAINTS.viewerSplitRatio.min,
        CONSTRAINTS.viewerSplitRatio.max
      );
      return { ...prev, viewerSplitRatio: clamped };
    });
  }, []);

  const resetToDefaults = useCallback(() => {
    setState({
      scenarioColumnWidth: DEFAULTS.scenarioColumnWidth,
      viewerSplitRatio: DEFAULTS.viewerSplitRatio,
    });
  }, []);

  return {
    state,
    setScenarioColumnWidth,
    setViewerSplitRatio,
    handleScenarioResize,
    handleViewerResize,
    resetToDefaults,
  };
}
