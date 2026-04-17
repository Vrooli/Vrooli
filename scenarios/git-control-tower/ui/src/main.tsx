import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import "./styles.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Pause polling when the browser tab is hidden to reduce idle API load.
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
    },
  },
});

if (window.top !== window.self) {
  // INTEROP-CRITICAL: Embedded mounts must identify themselves so the parent bridge can route events correctly.
  initIframeBridgeChild({ appId: "git-control-tower" });
}

// INTEROP-CRITICAL: Spatial navigation must be initialized at startup for iframe-hosted remote control flows.
initSpatialNav();

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element #root not found");
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    </ErrorBoundary>
  </React.StrictMode>
);
