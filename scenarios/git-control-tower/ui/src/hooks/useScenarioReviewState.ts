import { useState, useCallback, useRef, useEffect } from "react";
import type { ReviewTab } from "./useUrlState";
import type { ExecutionMode } from "../lib/api";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ScenarioReviewState {
  activeTab: ReviewTab;
  agentRunId: string | null;
  screenshots: {
    activePresetIndex: number;
    selectedPage: number;
  };
  workflows: {
    selectedModes: ExecutionMode[];
    viewRole: "baseline" | "capture";
  };
  codeQuality: {
    view: "changed" | "scenario";
  };
  rules: {
    jobId: string | null;
  };
}

/** Recursive partial — allows patching nested objects without supplying every key. */
export type DeepPartial<T> = {
  [K in keyof T]?: T[K] extends (infer U)[]
    ? U[] // arrays replace wholesale
    : T[K] extends object
      ? DeepPartial<T[K]>
      : T[K];
};

export const DEFAULT_STATE: ScenarioReviewState = {
  activeTab: "overview",
  agentRunId: null,
  screenshots: { activePresetIndex: 0, selectedPage: 0 },
  workflows: { selectedModes: ["observer"], viewRole: "capture" },
  codeQuality: { view: "changed" },
  rules: { jobId: null },
};

// ---------------------------------------------------------------------------
// localStorage helpers (exported for testing)
// ---------------------------------------------------------------------------

const STORAGE_PREFIX = "gct.reviewState.";
const MAX_ENTRIES = 50;
const MAX_AGE_MS = 30 * 24 * 60 * 60 * 1000; // 30 days

interface StoredEntry {
  version: 1;
  lastAccessed: number;
  state: ScenarioReviewState;
}

/**
 * Deep-merge `patch` into `base`. Arrays and primitives in `patch` replace
 * the corresponding value in `base`; plain objects are merged recursively.
 */
function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function parseStoredEntry(raw: string): StoredEntry | null {
  const parsed: unknown = JSON.parse(raw);
  if (!isPlainObject(parsed)) {
    return null;
  }
  const { version, lastAccessed, state } = parsed;
  if (version !== 1 || typeof lastAccessed !== "number" || !isPlainObject(state)) {
    return null;
  }
  return {
    version: 1,
    lastAccessed,
    state: deepMerge(DEFAULT_STATE, state as DeepPartial<ScenarioReviewState>),
  };
}

export function deepMerge<T extends object>(
  base: T,
  patch: DeepPartial<T>,
): T {
  const result = { ...base } as T;
  for (const key of Object.keys(patch) as (keyof T)[]) {
    const patchVal = patch[key];
    const baseVal = base[key];
    if (isPlainObject(patchVal) && isPlainObject(baseVal)) {
      result[key] = deepMerge(
        baseVal,
        patchVal as DeepPartial<typeof baseVal>,
      ) as T[keyof T];
      continue;
    }
    result[key] = patchVal as T[keyof T];
  }
  return result;
}

/** Load per-scenario state from localStorage, merged with defaults. */
export function loadState(slug: string): ScenarioReviewState {
  if (!slug) return { ...DEFAULT_STATE };
  try {
    const raw = localStorage.getItem(`${STORAGE_PREFIX}${slug}`);
    if (!raw) return { ...DEFAULT_STATE };
    const entry = parseStoredEntry(raw);
    if (!entry || entry.version !== 1 || !entry.state) return { ...DEFAULT_STATE };
    // Deep-merge stored partial with defaults to handle schema evolution
    return entry.state;
  } catch {
    return { ...DEFAULT_STATE };
  }
}

/** Save per-scenario state to localStorage and run opportunistic cleanup. */
export function saveState(slug: string, state: ScenarioReviewState): void {
  if (!slug) return;
  try {
    const entry: StoredEntry = {
      version: 1,
      lastAccessed: Date.now(),
      state,
    };
    localStorage.setItem(`${STORAGE_PREFIX}${slug}`, JSON.stringify(entry));
    pruneOldEntries(slug);
  } catch {
    // localStorage full or unavailable — silently ignore
  }
}

/** Remove entries older than 30 days or beyond 50 count (never prune `currentSlug`). */
export function pruneOldEntries(currentSlug: string): void {
  try {
    const now = Date.now();
    const entries: { key: string; lastAccessed: number }[] = [];

    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (!key || !key.startsWith(STORAGE_PREFIX)) continue;
      const slug = key.slice(STORAGE_PREFIX.length);
      if (slug === currentSlug) continue;

      try {
        const raw = localStorage.getItem(key);
        if (!raw) continue;
        const entry = parseStoredEntry(raw);
        if (!entry) {
          localStorage.removeItem(key);
          continue;
        }
        if (now - entry.lastAccessed > MAX_AGE_MS) {
          localStorage.removeItem(key);
        } else {
          entries.push({ key, lastAccessed: entry.lastAccessed });
        }
      } catch {
        // Corrupt entry — remove it
        localStorage.removeItem(key);
      }
    }

    // If still over the cap, evict oldest
    if (entries.length >= MAX_ENTRIES) {
      entries.sort((a, b) => a.lastAccessed - b.lastAccessed);
      const toRemove = entries.length - MAX_ENTRIES + 1;
      for (let i = 0; i < toRemove; i++) {
        const entry = entries[i];
        if (entry) localStorage.removeItem(entry.key);
      }
    }
  } catch {
    // Ignore errors during cleanup
  }
}

/** Validate that `tab` is in `visibleTabs`; fall back to "overview" if not. */
export function validateTab(tab: ReviewTab, visibleTabs: ReviewTab[]): ReviewTab {
  if (visibleTabs.length === 0) return tab; // Capabilities still loading
  return visibleTabs.includes(tab) ? tab : "overview";
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

interface UseScenarioReviewStateOptions {
  /** URL-provided overrides that take precedence on initial load. */
  urlOverrides?: { activeTab?: ReviewTab; agentRunId?: string | null };
}

interface UseScenarioReviewStateReturn {
  /** Current merged state for the active scenario. */
  state: ScenarioReviewState;
  /** Patch one or more fields (deep-merged). */
  update: (patch: DeepPartial<ScenarioReviewState>) => void;
  /** Save current → load next. Returns the loaded state. */
  switchScenario: (prevSlug: string, nextSlug: string) => ScenarioReviewState;
}

export function useScenarioReviewState(
  scenarioSlug: string,
  options?: UseScenarioReviewStateOptions,
): UseScenarioReviewStateReturn {
  const urlOverridesRef = useRef(options?.urlOverrides);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const stateRef = useRef<ScenarioReviewState>(DEFAULT_STATE);

  // Initialize state from localStorage + URL overrides on mount / slug change
  const [state, setState] = useState<ScenarioReviewState>(() => {
    const loaded = loadState(scenarioSlug);
    const overrides = urlOverridesRef.current;
    const merged = overrides
      ? deepMerge(loaded, overrides as DeepPartial<ScenarioReviewState>)
      : loaded;
    stateRef.current = merged;
    return merged;
  });

  // When scenarioSlug changes externally (not via switchScenario), re-load
  const prevSlugRef = useRef(scenarioSlug);
  useEffect(() => {
    if (scenarioSlug === prevSlugRef.current) return;
    prevSlugRef.current = scenarioSlug;
    const loaded = loadState(scenarioSlug);
    stateRef.current = loaded;
    setState(loaded);
  }, [scenarioSlug]);

  // Debounced save
  const scheduleSave = useCallback(
    (slug: string, newState: ScenarioReviewState) => {
      if (debounceRef.current !== null) {
        clearTimeout(debounceRef.current);
      }
      debounceRef.current = setTimeout(() => {
        debounceRef.current = null;
        saveState(slug, newState);
      }, 300);
    },
    [],
  );

  // Flush on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current !== null) {
        clearTimeout(debounceRef.current);
        saveState(prevSlugRef.current, stateRef.current);
      }
    };
  }, []);

  const update = useCallback(
    (patch: DeepPartial<ScenarioReviewState>) => {
      setState((prev) => {
        const next = deepMerge(prev, patch);
        stateRef.current = next;
        scheduleSave(prevSlugRef.current, next);
        return next;
      });
    },
    [scheduleSave],
  );

  const switchScenario = useCallback(
    (prevSlug: string, nextSlug: string): ScenarioReviewState => {
      // Flush any pending debounce and save current state immediately
      if (debounceRef.current !== null) {
        clearTimeout(debounceRef.current);
        debounceRef.current = null;
      }
      saveState(prevSlug, stateRef.current);

      // Load next scenario's state
      const loaded = loadState(nextSlug);
      stateRef.current = loaded;
      prevSlugRef.current = nextSlug;
      setState(loaded);
      return loaded;
    },
    [],
  );

  return { state, update, switchScenario };
}
