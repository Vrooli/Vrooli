import { useState, useEffect, useCallback } from "react";

export type View = "dashboard" | "wizard" | "deployments" | "docs";

export interface RouterState {
  view: View;
  docPath: string | null;
}

const DEFAULT_STATE: RouterState = {
  view: "dashboard",
  docPath: null,
};

/**
 * Parse URL hash to extract view and doc path
 * Examples:
 *   #dashboard -> { view: "dashboard", docPath: null }
 *   #deployments -> { view: "deployments", docPath: null }
 *   #docs -> { view: "docs", docPath: null }
 *   #docs/guides/vps-setup -> { view: "docs", docPath: "guides/vps-setup" }
 *   #wizard -> { view: "wizard", docPath: null }
 */
function parseHash(hash: string): RouterState {
  const cleanHash = hash.replace(/^#/, "");
  if (!cleanHash) return DEFAULT_STATE;

  const parts = cleanHash.split("/");
  const firstPart = parts[0];

  // Check if first part is a valid view
  if (["dashboard", "wizard", "deployments", "docs"].includes(firstPart)) {
    const view = firstPart as View;

    // For docs view, extract the doc path
    if (view === "docs" && parts.length > 1) {
      const docPath = parts.slice(1).join("/");
      return { view, docPath: docPath || null };
    }

    return { view, docPath: null };
  }

  return DEFAULT_STATE;
}

/**
 * Build URL hash from state
 */
function buildHash(state: RouterState): string {
  if (state.view === "docs" && state.docPath) {
    return `${state.view}/${state.docPath}`;
  }
  return state.view;
}

/**
 * Hook for hash-based routing with doc path support
 */
export function useHashRouter() {
  const [state, setState] = useState<RouterState>(() => {
    if (typeof window === "undefined") return DEFAULT_STATE;
    return parseHash(window.location.hash);
  });

  // Sync URL hash when state changes
  const updateHash = useCallback((newState: RouterState) => {
    if (typeof window === "undefined") return;
    const hash = buildHash(newState);
    window.history.replaceState(null, "", `#${hash}`);
  }, []);

  // Navigate to a view
  const navigate = useCallback((view: View, docPath?: string | null) => {
    const newState: RouterState = {
      view,
      docPath: view === "docs" ? (docPath ?? null) : null,
    };
    setState(newState);
    updateHash(newState);
  }, [updateHash]);

  // Navigate to a specific doc
  const navigateToDoc = useCallback((docPath: string) => {
    const newState: RouterState = { view: "docs", docPath };
    setState(newState);
    updateHash(newState);
  }, [updateHash]);

  // Listen for hash changes (browser back/forward)
  useEffect(() => {
    if (typeof window === "undefined") return;

    const handleHashChange = () => {
      const newState = parseHash(window.location.hash);
      setState(newState);
    };

    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  return {
    view: state.view,
    docPath: state.docPath,
    navigate,
    navigateToDoc,
  };
}

/**
 * Build a hash URL for linking
 */
export function buildDocUrl(docPath: string): string {
  return `#docs/${docPath}`;
}
