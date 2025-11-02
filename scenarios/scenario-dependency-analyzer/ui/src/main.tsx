import React from "react";
import ReactDOM from "react-dom/client";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import App from "./App";
import "./styles/global.css";

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
  } catch (error) {
    // Fall back to default origin when parsing fails.
  }

  initIframeBridgeChild({ parentOrigin, appId: "scenario-dependency-analyzer" });
  window.__scenarioDependencyAnalyzerBridgeInitialized = true;
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
