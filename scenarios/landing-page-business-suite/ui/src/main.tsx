import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import { installChunkReloadGuard } from "@vrooli/api-base";
import App from "./App";
import { onProfilerRender } from "./lib/profiler";
import "./styles.css";

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard();

declare global {
  interface Window {
    __landingManagerBridgeInitialized?: boolean;
  }
}

const queryClient = new QueryClient();

if (typeof window !== 'undefined' && window.parent !== window && !window.__landingManagerBridgeInitialized) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Unable to parse parent origin - will use default
  }

  initIframeBridgeChild({
    parentOrigin,
    appId: "landing-page-business-suite",
    captureLogs: {
      enabled: true,
      streaming: true,
      levels: ['log', 'info', 'warn', 'error', 'debug'],
      bufferSize: 100
    },
    captureNetwork: {
      enabled: true,
      streaming: true,
      bufferSize: 100
    }
  });
  window.__landingManagerBridgeInitialized = true;
}

const root = document.getElementById("root");
if (!root) {
  throw new Error("Root element not found");
}

ReactDOM.createRoot(root).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <React.Profiler id="App" onRender={onProfilerRender}>
        <App />
      </React.Profiler>
    </QueryClientProvider>
  </React.StrictMode>
);
