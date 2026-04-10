import { useEffect, useCallback, useRef } from "react";
import type { ViewMode } from "../lib/api";

export type ReviewTab = "overview" | "metrics" | "screenshots" | "workflows" | "tests" | "code-quality" | "rules" | "ai-provenance" | "agent";

const VALID_REVIEW_TABS: readonly string[] = ["overview", "metrics", "screenshots", "workflows", "tests", "code-quality", "rules", "ai-provenance", "agent"];

/**
 * URL state parameters for deep linking
 */
export interface UrlState {
  /** Selected file path (URL-encoded) */
  file?: string;
  /** View mode: diff, full_diff, source, preview */
  mode?: ViewMode;
  /** Whether viewing staged version */
  staged?: boolean;
  /** Active panel: changes or related */
  panel?: "changes" | "related";
  /** Commit hash when in history mode */
  commit?: string;
  /** Primary panel (when set to review with a scenario slug) */
  primary?: string;
  /** Scenario slug for review panel */
  reviewScenario?: string;
  /** Active tab within the review panel */
  reviewTab?: ReviewTab;
  /** Whether viewing an arbitrary file (not from the change list) */
  anyFile?: boolean;
  /** Agent run ID for the agent tab */
  agentRunId?: string;
}

/**
 * Parse URL search params into UrlState
 */
export function parseUrlState(search: string): UrlState {
  const params = new URLSearchParams(search);
  const state: UrlState = {};

  const file = params.get("file");
  if (file) {
    state.file = decodeURIComponent(file);
  }

  const mode = params.get("mode");
  if (mode && ["diff", "full_diff", "source", "preview"].includes(mode)) {
    state.mode = mode as ViewMode;
  }

  const staged = params.get("staged");
  if (staged === "true") {
    state.staged = true;
  } else if (staged === "false") {
    state.staged = false;
  }

  const panel = params.get("panel");
  if (panel === "changes" || panel === "related") {
    state.panel = panel;
  }

  const commit = params.get("commit");
  if (commit) {
    state.commit = commit;
  }

  const primary = params.get("primary");
  if (primary) {
    state.primary = primary;
  }

  const reviewScenario = params.get("reviewScenario");
  if (reviewScenario) {
    state.reviewScenario = decodeURIComponent(reviewScenario);
  }

  const reviewTab = params.get("reviewTab");
  if (reviewTab && VALID_REVIEW_TABS.includes(reviewTab)) {
    state.reviewTab = reviewTab as ReviewTab;
  }

  const anyFile = params.get("anyFile");
  if (anyFile === "true") {
    state.anyFile = true;
  }

  const agentRunId = params.get("agentRunId");
  if (agentRunId) {
    state.agentRunId = agentRunId;
  }

  return state;
}

/**
 * Build URL search string from state
 */
export function buildUrlSearch(state: UrlState): string {
  const params = new URLSearchParams();

  if (state.file) {
    params.set("file", encodeURIComponent(state.file));
  }

  if (state.mode && state.mode !== "diff") {
    params.set("mode", state.mode);
  }

  if (state.staged === true) {
    params.set("staged", "true");
  }

  if (state.panel === "related") {
    params.set("panel", "related");
  }

  if (state.commit) {
    params.set("commit", state.commit);
  }

  if (state.primary && state.primary !== "diff") {
    params.set("primary", state.primary);
  }

  if (state.reviewScenario) {
    params.set("reviewScenario", encodeURIComponent(state.reviewScenario));
  }

  if (state.reviewTab && state.reviewTab !== "overview") {
    params.set("reviewTab", state.reviewTab);
  }

  if (state.anyFile === true) {
    params.set("anyFile", "true");
  }

  if (state.agentRunId) {
    params.set("agentRunId", state.agentRunId);
  }

  const search = params.toString();
  return search ? `?${search}` : "";
}

/**
 * Update URL without triggering a navigation
 */
export function updateUrl(state: UrlState): void {
  const search = buildUrlSearch(state);
  const newUrl = window.location.pathname + search;

  // Use replaceState to avoid polluting browser history with every state change
  window.history.replaceState({ ...state }, "", newUrl);
}

/**
 * Push URL state (creates history entry for back/forward navigation)
 */
export function pushUrl(state: UrlState): void {
  const search = buildUrlSearch(state);
  const newUrl = window.location.pathname + search;

  window.history.pushState({ ...state }, "", newUrl);
}

interface UseUrlStateOptions {
  /** Callback when URL state changes (browser back/forward) */
  onStateChange?: (state: UrlState) => void;
  /** Debounce delay for URL updates in milliseconds */
  debounceMs?: number;
}

interface UseUrlStateReturn {
  /** Get current URL state */
  getState: () => UrlState;
  /** Update URL state (debounced, uses replaceState) */
  updateState: (state: UrlState) => void;
  /** Push URL state (creates history entry) */
  pushState: (state: UrlState) => void;
}

/**
 * Hook for URL-based state management
 *
 * Provides:
 * - Initial state from URL on mount
 * - Debounced URL updates via replaceState
 * - Browser back/forward navigation handling
 *
 * @example
 * ```tsx
 * const { getState, updateState } = useUrlState({
 *   onStateChange: (state) => {
 *     if (state.file) setSelectedFile(state.file);
 *   }
 * });
 *
 * // Update URL when file selection changes
 * useEffect(() => {
 *   updateState({ file: selectedFile, mode: viewMode });
 * }, [selectedFile, viewMode]);
 * ```
 */
export function useUrlState(options: UseUrlStateOptions = {}): UseUrlStateReturn {
  const { onStateChange, debounceMs = 100 } = options;
  const debounceRef = useRef<number | null>(null);
  const lastStateRef = useRef<UrlState>({});

  // Handle browser back/forward navigation
  useEffect(() => {
    const handlePopState = () => {
      // Parse state from URL (more reliable than event.state)
      const state = parseUrlState(window.location.search);
      lastStateRef.current = state;
      onStateChange?.(state);
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, [onStateChange]);

  const getState = useCallback((): UrlState => {
    return parseUrlState(window.location.search);
  }, []);

  const updateState = useCallback(
    (state: UrlState): void => {
      // Clear any pending debounce
      if (debounceRef.current !== null) {
        window.clearTimeout(debounceRef.current);
      }

      // Debounce the URL update
      debounceRef.current = window.setTimeout(() => {
        debounceRef.current = null;
        lastStateRef.current = state;
        updateUrl(state);
      }, debounceMs);
    },
    [debounceMs]
  );

  const pushState = useCallback((state: UrlState): void => {
    // Clear any pending debounce
    if (debounceRef.current !== null) {
      window.clearTimeout(debounceRef.current);
      debounceRef.current = null;
    }

    lastStateRef.current = state;
    pushUrl(state);
  }, []);

  // Cleanup debounce on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current !== null) {
        window.clearTimeout(debounceRef.current);
      }
    };
  }, []);

  return {
    getState,
    updateState,
    pushState
  };
}
