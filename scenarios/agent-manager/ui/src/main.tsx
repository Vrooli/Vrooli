import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { getProxyInfo } from "@vrooli/api-base";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import "./styles/global.css";

declare global {
  interface Window {
    __agentManagerBridgeInitialized?: boolean;
  }
}

// INTEROP-CRITICAL: BrowserRouter must use proxy-aware basename so route links
// work correctly when this UI is served under /apps/<scenario>/proxy.
function getRouterBasename(): string {
  const proxyInfo = getProxyInfo();
  const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
  if (!proxyPath) {
    return "";
  }
  return proxyPath.replace(/\/+$/, "");
}

const routerBasename = getRouterBasename();

if (
  typeof window !== "undefined" &&
  window.parent !== window &&
  !window.__agentManagerBridgeInitialized
) {
  let parentOrigin: string | undefined;

  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Fall back to default origin when parsing fails.
  }

  initIframeBridgeChild({ parentOrigin, appId: "agent-manager" });
  window.__agentManagerBridgeInitialized = true;
}

initSpatialNav();

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <BrowserRouter basename={routerBasename}>
      <App />
    </BrowserRouter>
  </React.StrictMode>
);
