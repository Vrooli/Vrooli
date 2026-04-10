import { useState, useEffect, useCallback } from "react";

export interface UseCollapsiblePanelOptions {
  /** Storage key for localStorage persistence (will be prefixed with "agm.") */
  storageKey: string;
  /** Optional full localStorage key override (for legacy key compatibility) */
  persistKey?: string;
  /** Default collapsed state */
  defaultCollapsed?: boolean;
}

export interface UseCollapsiblePanelReturn {
  /** Whether the panel is collapsed */
  isCollapsed: boolean;
  /** Toggle the collapsed state */
  toggle: () => void;
  /** Expand the panel */
  expand: () => void;
  /** Collapse the panel */
  collapse: () => void;
}

const STORAGE_PREFIX = "agm.panel.";

/**
 * Hook for managing collapsible panel state with localStorage persistence.
 */
export function useCollapsiblePanel({
  storageKey,
  persistKey,
  defaultCollapsed = false,
}: UseCollapsiblePanelOptions): UseCollapsiblePanelReturn {
  const fullStorageKey = persistKey ?? `${STORAGE_PREFIX}${storageKey}.collapsed`;

  // Initialize from localStorage or default
  const [isCollapsed, setIsCollapsed] = useState(() => {
    if (typeof window === "undefined") return defaultCollapsed;
    const stored = localStorage.getItem(fullStorageKey);
    if (stored !== null) {
      return stored === "true";
    }
    return defaultCollapsed;
  });

  // Save to localStorage when state changes
  useEffect(() => {
    localStorage.setItem(fullStorageKey, String(isCollapsed));
  }, [fullStorageKey, isCollapsed]);

  const toggle = useCallback(() => {
    setIsCollapsed((prev) => !prev);
  }, []);

  const expand = useCallback(() => {
    setIsCollapsed(false);
  }, []);

  const collapse = useCallback(() => {
    setIsCollapsed(true);
  }, []);

  return {
    isCollapsed,
    toggle,
    expand,
    collapse,
  };
}
