import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { onProfilerRender } from "./lib/profiler";
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
        {/* Top-level Profiler boundary. Inert in regular prod (react-dom strips
            the profiling hook); emits user_timing entries via onProfilerRender
            when the perf-build channel is active. See lib/profiler.ts. */}
        <React.Profiler id="App" onRender={onProfilerRender}>
          <App />
        </React.Profiler>
      </QueryClientProvider>
    </ErrorBoundary>
  </React.StrictMode>
);
