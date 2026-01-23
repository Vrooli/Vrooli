import { useState, useEffect, useCallback, useMemo } from "react";
import type {
  DeploymentTab,
  LiveStateSubtab,
  ModalType,
  DeploymentUrlState,
} from "../types/url";
import {
  VALID_TABS,
  VALID_SUBTABS,
  VALID_MODALS,
  DEFAULT_DEPLOYMENT_URL_STATE,
} from "../types/url";

/**
 * Parse a deployment hash URL into structured state.
 *
 * Supported formats:
 *   #deployments                              → deployment list
 *   #deployments/<id>                         → deployment detail, overview tab
 *   #deployments/<id>?tab=live-state          → specific tab
 *   #deployments/<id>?tab=live-state&subtab=ports → tab + subtab
 *   #deployments/<id>?modal=redeploy          → modal open
 *   #deployments/<id>?modal=spawn-agent&taskType=fix → modal with params
 *
 * @param hash - The URL hash string (e.g., "#deployments/abc123?tab=live-state")
 * @returns Parsed deployment URL state
 */
export function parseDeploymentHash(hash: string): DeploymentUrlState {
  const cleanHash = hash.replace(/^#/, "");

  // Must start with "deployments"
  if (!cleanHash.startsWith("deployments")) {
    return { ...DEFAULT_DEPLOYMENT_URL_STATE };
  }

  // Split path and query string
  const [pathPart, queryPart] = cleanHash.split("?");
  const pathSegments = pathPart.split("/");

  // pathSegments[0] is "deployments"
  // pathSegments[1] would be the deployment ID if present
  const deploymentId = pathSegments[1] || null;

  // Parse query parameters
  const params = new URLSearchParams(queryPart || "");

  // Parse tab with validation
  const tabParam = params.get("tab");
  const tab: DeploymentTab = tabParam && VALID_TABS.includes(tabParam as DeploymentTab)
    ? (tabParam as DeploymentTab)
    : "overview";

  // Parse subtab with validation
  const subtabParam = params.get("subtab");
  const subtab: LiveStateSubtab = subtabParam && VALID_SUBTABS.includes(subtabParam as LiveStateSubtab)
    ? (subtabParam as LiveStateSubtab)
    : "processes";

  // Parse modal with validation
  const modalParam = params.get("modal");
  const modal: ModalType | null = modalParam && VALID_MODALS.includes(modalParam as ModalType)
    ? (modalParam as ModalType)
    : null;

  // Collect modal params (any params that aren't tab, subtab, modal)
  const modalParams: Record<string, string> = {};
  const reservedParams = ["tab", "subtab", "modal"];
  params.forEach((value, key) => {
    if (!reservedParams.includes(key)) {
      modalParams[key] = value;
    }
  });

  return {
    deploymentId,
    tab,
    subtab,
    modal,
    modalParams,
  };
}

/**
 * Build a hash URL from deployment state.
 * Only includes parameters that differ from defaults to keep URLs clean.
 *
 * @param state - Partial deployment URL state
 * @returns Hash string (e.g., "#deployments/abc123?tab=live-state")
 */
export function buildDeploymentHash(state: Partial<DeploymentUrlState>): string {
  const { deploymentId, tab, subtab, modal, modalParams } = state;

  // Base path
  let hash = "#deployments";

  // Add deployment ID if present
  if (deploymentId) {
    hash += `/${deploymentId}`;
  }

  // Build query params, only including non-default values
  const params = new URLSearchParams();

  if (tab && tab !== "overview") {
    params.set("tab", tab);
  }

  // Only include subtab if we're on live-state tab and it's not the default
  if (tab === "live-state" && subtab && subtab !== "processes") {
    params.set("subtab", subtab);
  }

  if (modal) {
    params.set("modal", modal);
  }

  // Add modal params
  if (modalParams) {
    Object.entries(modalParams).forEach(([key, value]) => {
      if (value) {
        params.set(key, value);
      }
    });
  }

  const queryString = params.toString();
  if (queryString) {
    hash += `?${queryString}`;
  }

  return hash;
}

/** Return type for the useDeploymentUrl hook */
export interface UseDeploymentUrlReturn {
  /** Current URL state */
  state: DeploymentUrlState;
  /** Navigate to a deployment (or null for list view) */
  selectDeployment: (id: string | null) => void;
  /** Change the active tab */
  setTab: (tab: DeploymentTab) => void;
  /** Change the active subtab (within live-state) */
  setSubtab: (subtab: LiveStateSubtab) => void;
  /** Open a modal with optional params */
  openModal: (modal: ModalType, params?: Record<string, string>) => void;
  /** Close the currently open modal */
  closeModal: () => void;
}

/**
 * Hook for managing deployment URL state.
 * Synchronizes React state with URL hash, supporting browser navigation.
 *
 * History behavior:
 * - selectDeployment: pushState (major navigation)
 * - setTab: pushState (users expect back to return to previous tab)
 * - setSubtab: replaceState (minor UI change)
 * - openModal: pushState (back should close modal)
 * - closeModal: history.back() if modal was pushed, else replaceState
 */
export function useDeploymentUrl(): UseDeploymentUrlReturn {
  // Track whether the current modal was opened via pushState
  const [modalWasPushed, setModalWasPushed] = useState(false);

  // Parse initial state from URL
  const [state, setState] = useState<DeploymentUrlState>(() => {
    if (typeof window === "undefined") return DEFAULT_DEPLOYMENT_URL_STATE;
    return parseDeploymentHash(window.location.hash);
  });

  // Listen for hash changes (browser back/forward, external navigation)
  useEffect(() => {
    if (typeof window === "undefined") return;

    const handleHashChange = () => {
      const newState = parseDeploymentHash(window.location.hash);
      setState(newState);
      // Reset modal pushed tracking when navigating via browser
      if (!newState.modal) {
        setModalWasPushed(false);
      }
    };

    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  // Navigate to a deployment (pushState for major navigation)
  const selectDeployment = useCallback((id: string | null) => {
    const newState: DeploymentUrlState = {
      ...DEFAULT_DEPLOYMENT_URL_STATE,
      deploymentId: id,
    };
    const hash = buildDeploymentHash(newState);
    window.history.pushState(null, "", hash);
    setState(newState);
    setModalWasPushed(false);
  }, []);

  // Change tab (pushState so back returns to previous tab)
  const setTab = useCallback((tab: DeploymentTab) => {
    setState((prev) => {
      const newState: DeploymentUrlState = {
        ...prev,
        tab,
        // Reset subtab to default when changing tabs
        subtab: tab === prev.tab ? prev.subtab : "processes",
        // Clear modal when changing tabs
        modal: null,
        modalParams: {},
      };
      const hash = buildDeploymentHash(newState);
      window.history.pushState(null, "", hash);
      setModalWasPushed(false);
      return newState;
    });
  }, []);

  // Change subtab (replaceState - minor UI change)
  const setSubtab = useCallback((subtab: LiveStateSubtab) => {
    setState((prev) => {
      const newState: DeploymentUrlState = {
        ...prev,
        subtab,
      };
      const hash = buildDeploymentHash(newState);
      window.history.replaceState(null, "", hash);
      return newState;
    });
  }, []);

  // Open modal (pushState so back closes it)
  const openModal = useCallback((modal: ModalType, params?: Record<string, string>) => {
    setState((prev) => {
      const newState: DeploymentUrlState = {
        ...prev,
        modal,
        modalParams: params || {},
      };
      const hash = buildDeploymentHash(newState);
      window.history.pushState(null, "", hash);
      setModalWasPushed(true);
      return newState;
    });
  }, []);

  // Close modal (go back if pushed, else replace)
  const closeModal = useCallback(() => {
    if (modalWasPushed) {
      // Go back to undo the pushState
      window.history.back();
      setModalWasPushed(false);
    } else {
      // Modal state came from URL directly (e.g., shared link), use replaceState
      setState((prev) => {
        const newState: DeploymentUrlState = {
          ...prev,
          modal: null,
          modalParams: {},
        };
        const hash = buildDeploymentHash(newState);
        window.history.replaceState(null, "", hash);
        return newState;
      });
    }
  }, [modalWasPushed]);

  return useMemo(
    () => ({
      state,
      selectDeployment,
      setTab,
      setSubtab,
      openModal,
      closeModal,
    }),
    [state, selectDeployment, setTab, setSubtab, openModal, closeModal]
  );
}
