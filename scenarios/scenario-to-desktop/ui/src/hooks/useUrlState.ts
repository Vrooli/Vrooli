/**
 * Hook for managing URL-based state synchronization.
 * Provides shareable URLs and browser back/forward navigation support.
 */

import { useCallback, useEffect, useMemo, useState } from "react";

export type ViewMode = "generator" | "inventory" | "docs" | "records" | "signing";

interface UrlParams {
  view?: ViewMode;
  scenario?: string;
  doc?: string;
}

/**
 * Parse URL search parameters into typed state.
 */
export function parseSearchParams(): UrlParams {
  if (typeof window === "undefined") return {};
  const params = new URLSearchParams(window.location.search);
  const view = params.get("view") as ViewMode | null;
  const scenario = params.get("scenario") || undefined;
  const doc = params.get("doc") || undefined;
  return { view: view || undefined, scenario, doc };
}

interface UseUrlStateOptions {
  defaultView?: ViewMode;
  onViewChange?: (view: ViewMode) => void;
  onScenarioChange?: (scenario: string) => void;
  onDocChange?: (doc: string | null) => void;
}

interface UseUrlStateReturn {
  viewMode: ViewMode;
  setViewMode: (view: ViewMode) => void;
  scenarioName: string;
  setScenarioName: (name: string) => void;
  docPath: string | null;
  setDocPath: (path: string | null) => void;
  initialParams: UrlParams;
}

/**
 * Hook that synchronizes state with URL search parameters.
 * Provides shareable URLs and handles browser navigation.
 */
export function useUrlState(options: UseUrlStateOptions = {}): UseUrlStateReturn {
  const { defaultView = "inventory", onViewChange, onScenarioChange, onDocChange } = options;

  const initialParams = useMemo(() => parseSearchParams(), []);

  const [viewMode, setViewModeState] = useState<ViewMode>(
    initialParams.view ?? defaultView
  );
  const [scenarioName, setScenarioNameState] = useState(
    initialParams.scenario ?? ""
  );
  const [docPath, setDocPathState] = useState<string | null>(
    initialParams.doc ?? null
  );

  // Wrapped setters that also trigger callbacks
  const setViewMode = useCallback((view: ViewMode) => {
    setViewModeState(view);
    onViewChange?.(view);
  }, [onViewChange]);

  const setScenarioName = useCallback((name: string) => {
    setScenarioNameState(name);
    onScenarioChange?.(name);
  }, [onScenarioChange]);

  const setDocPath = useCallback((path: string | null) => {
    setDocPathState(path);
    onDocChange?.(path);
  }, [onDocChange]);

  // Sync state from URL on popstate (back/forward)
  useEffect(() => {
    if (typeof window === "undefined") return;

    const syncFromLocation = () => {
      const { view, scenario, doc } = parseSearchParams();
      if (view) setViewModeState(view);
      if (scenario !== undefined) setScenarioNameState(scenario);
      if (doc !== undefined) setDocPathState(doc ?? null);
    };

    syncFromLocation();
    window.addEventListener("popstate", syncFromLocation);
    return () => window.removeEventListener("popstate", syncFromLocation);
  }, []);

  // Persist state to URL for shareable routing
  useEffect(() => {
    if (typeof window === "undefined") return;

    const url = new URL(window.location.href);
    const params = new URLSearchParams(url.search);
    params.set("view", viewMode);

    if (scenarioName) {
      params.set("scenario", scenarioName);
    } else {
      params.delete("scenario");
    }

    if (docPath) {
      params.set("doc", docPath);
    } else {
      params.delete("doc");
    }

    url.search = params.toString();
    const newUrl = url.toString();
    if (window.location.href !== newUrl) {
      window.history.replaceState(null, "", newUrl);
    }
  }, [viewMode, scenarioName, docPath]);

  return {
    viewMode,
    setViewMode,
    scenarioName,
    setScenarioName,
    docPath,
    setDocPath,
    initialParams,
  };
}
