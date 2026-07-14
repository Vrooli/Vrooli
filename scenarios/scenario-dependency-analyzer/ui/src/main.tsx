import React from "react";
import ReactDOM from "react-dom/client";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import { installChunkReloadGuard } from "@vrooli/api-base";
import App from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import "./i18n";
import "./styles/global.css";

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard();

declare global {
  interface Window {
    __scenarioDependencyAnalyzerBridgeInitialized?: boolean;
  }
}

if (
  typeof window !== "undefined" &&
  window.parent !== window &&
  !window.__scenarioDependencyAnalyzerBridgeInitialized
) {
  let parentOrigin: string | undefined;

  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Fall back to default origin when parsing fails.
  }

  initIframeBridgeChild({ parentOrigin, appId: "scenario-dependency-analyzer" });
  window.__scenarioDependencyAnalyzerBridgeInitialized = true;
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>
);
