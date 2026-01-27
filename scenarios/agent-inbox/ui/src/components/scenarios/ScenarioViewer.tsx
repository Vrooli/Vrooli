/**
 * ScenarioViewer - Fullscreen iframe page for viewing embedded scenario UIs.
 *
 * This component renders a proxied iframe of a scenario's UI, allowing users
 * to interact with scenario UIs directly from the agent-inbox. The iframe
 * loads through the proxy endpoints (/embedded/{scenario}/*) to ensure the
 * scenario UI works in any context (localhost, tunnel, remote).
 *
 * Usage:
 * - URL: /scenarios/{name}/view
 * - Optional query param: ?path={slug} for deep linking into the scenario UI
 */

import { useState, useEffect, useCallback } from "react";
import { ArrowLeft, ExternalLink, RefreshCw, AlertCircle, Loader2 } from "lucide-react";
import { Button } from "../ui/button";

interface ScenarioViewerProps {
  scenarioName: string;
  path?: string;
  onBack: () => void;
}

type LoadState = "loading" | "ready" | "error";

export function ScenarioViewer({ scenarioName, path, onBack }: ScenarioViewerProps) {
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [errorMessage, setErrorMessage] = useState<string>("");
  const [iframeKey, setIframeKey] = useState(0);

  // Build the iframe src URL
  const iframeSrc = path
    ? `/embedded/${scenarioName}${path.startsWith("/") ? path : `/${path}`}`
    : `/embedded/${scenarioName}/`;

  // Check if scenario is available
  useEffect(() => {
    const checkScenarioAvailability = async () => {
      try {
        const response = await fetch(`/embedded/${scenarioName}/target`);
        if (!response.ok) {
          const data = await response.json().catch(() => ({}));
          setErrorMessage(data.detail || "Scenario is not available");
          setLoadState("error");
          return;
        }
        setLoadState("ready");
      } catch {
        setErrorMessage("Failed to connect to scenario");
        setLoadState("error");
      }
    };

    setLoadState("loading");
    checkScenarioAvailability();
  }, [scenarioName, iframeKey]);

  const handleRefresh = useCallback(() => {
    setLoadState("loading");
    setErrorMessage("");
    setIframeKey((k) => k + 1);
  }, []);

  const handleIframeLoad = useCallback(() => {
    // The iframe has loaded successfully
    setLoadState("ready");
  }, []);

  const handleIframeError = useCallback(() => {
    setErrorMessage("Failed to load scenario UI");
    setLoadState("error");
  }, []);

  // Open in new browser tab (direct to scenario)
  const handleOpenExternal = useCallback(async () => {
    try {
      const response = await fetch(`/embedded/${scenarioName}/target`);
      if (response.ok) {
        const data = await response.json();
        window.open(data.url, "_blank");
      }
    } catch {
      // Fallback: open proxy URL
      window.open(iframeSrc, "_blank");
    }
  }, [scenarioName, iframeSrc]);

  // Format scenario name for display
  const displayName = scenarioName
    .split("-")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");

  return (
    <div className="h-screen flex flex-col bg-slate-950">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-800 bg-slate-900">
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="sm"
            onClick={onBack}
            className="gap-2 text-slate-300 hover:text-white"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
          <div className="h-6 w-px bg-slate-700" />
          <h1 className="text-sm font-medium text-white">{displayName}</h1>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleRefresh}
            className="gap-2 text-slate-300 hover:text-white"
            disabled={loadState === "loading"}
          >
            <RefreshCw className={`h-4 w-4 ${loadState === "loading" ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleOpenExternal}
            className="gap-2 text-slate-300 hover:text-white"
          >
            <ExternalLink className="h-4 w-4" />
            Open Direct
          </Button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 relative">
        {/* Loading state */}
        {loadState === "loading" && (
          <div className="absolute inset-0 flex items-center justify-center bg-slate-950">
            <div className="text-center">
              <Loader2 className="h-8 w-8 text-slate-400 animate-spin mx-auto mb-3" />
              <p className="text-sm text-slate-400">Loading {displayName}...</p>
            </div>
          </div>
        )}

        {/* Error state */}
        {loadState === "error" && (
          <div className="absolute inset-0 flex items-center justify-center bg-slate-950">
            <div className="text-center max-w-md px-4">
              <div className="mb-4 p-3 rounded-full bg-red-500/10 inline-block">
                <AlertCircle className="h-8 w-8 text-red-400" />
              </div>
              <h2 className="text-lg font-medium text-white mb-2">Scenario Unavailable</h2>
              <p className="text-sm text-slate-400 mb-4">{errorMessage}</p>
              <div className="flex items-center justify-center gap-3">
                <Button variant="outline" size="sm" onClick={onBack}>
                  Go Back
                </Button>
                <Button variant="default" size="sm" onClick={handleRefresh}>
                  Try Again
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Iframe - always render but hide during loading/error */}
        <iframe
          key={iframeKey}
          src={loadState === "error" ? "about:blank" : iframeSrc}
          className={`w-full h-full border-0 ${loadState !== "ready" ? "invisible" : ""}`}
          title={`${displayName} UI`}
          onLoad={handleIframeLoad}
          onError={handleIframeError}
          sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-modals"
        />
      </div>
    </div>
  );
}

/**
 * Hook to detect if the current URL is a scenario viewer route.
 * Returns the scenario name and optional path if matched.
 */
export function useScenarioViewerRoute(): {
  isScenarioViewer: boolean;
  scenarioName: string | null;
  path: string | null;
} {
  const [routeInfo, setRouteInfo] = useState(() => parseScenarioViewerRoute());

  useEffect(() => {
    const handlePopState = () => {
      setRouteInfo(parseScenarioViewerRoute());
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  return routeInfo;
}

function parseScenarioViewerRoute(): {
  isScenarioViewer: boolean;
  scenarioName: string | null;
  path: string | null;
} {
  const pathname = window.location.pathname;
  const match = pathname.match(/^\/scenarios\/([a-zA-Z0-9-]+)\/view\/?$/);

  if (!match) {
    return { isScenarioViewer: false, scenarioName: null, path: null };
  }

  const searchParams = new URLSearchParams(window.location.search);
  const path = searchParams.get("path");

  return {
    isScenarioViewer: true,
    scenarioName: match[1] ?? null,
    path,
  };
}

/**
 * Navigate to the scenario viewer page.
 */
export function navigateToScenarioViewer(scenarioName: string, path?: string): void {
  const url = path
    ? `/scenarios/${scenarioName}/view?path=${encodeURIComponent(path)}`
    : `/scenarios/${scenarioName}/view`;

  window.history.pushState({ scenarioName, path }, "", url);
  // Dispatch a custom event so components can react to the navigation
  window.dispatchEvent(new PopStateEvent("popstate"));
}

/**
 * Open scenario viewer in a new tab.
 */
export function openScenarioViewerInNewTab(scenarioName: string, path?: string): void {
  const url = path
    ? `/scenarios/${scenarioName}/view?path=${encodeURIComponent(path)}`
    : `/scenarios/${scenarioName}/view`;

  window.open(url, "_blank");
}
