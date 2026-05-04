import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { ErrorBoundary } from "./components/ErrorBoundary";
import App from "./App";
import { onProfilerRender } from "./lib/profiler";
import "./styles.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Prevent queries from retrying indefinitely on errors
      retry: 2,
      // Don't refetch on window focus if we have an error
      refetchOnWindowFocus: (query) => query.state.status !== "error",
    },
    mutations: {
      // Prevent mutations from retrying indefinitely
      retry: 1,
    },
  },
});

// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe bridge initialization              ║
// ║                                                              ║
// ║  Must run BEFORE React mount so that:                        ║
// ║  1. Storage shimming is in place before any component        ║
// ║     accesses localStorage/sessionStorage                     ║
// ║  2. The bridge message channel is ready for host commands    ║
// ║                                                              ║
// ║  The window.top check ensures this is a no-op when           ║
// ║  running outside an iframe (localhost, tunnel).              ║
// ╚══════════════════════════════════════════════════════════════╝
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "agent-inbox" });
}

// StrictMode disabled: double-renders push borderline render counts over
// React's 50-render limit during rapid state transitions (e.g. fresh chat send).
const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element not found");
}
ReactDOM.createRoot(rootElement).render(
  <ErrorBoundary
    critical
    name="Root"
    onError={(error, errorInfo) => {
      // Log critical errors for debugging
      console.error("[CriticalError] App crashed:", error, errorInfo);
    }}
  >
    <QueryClientProvider client={queryClient}>
      <React.Profiler id="App" onRender={onProfilerRender}>
        <App />
      </React.Profiler>
    </QueryClientProvider>
  </ErrorBoundary>
);
