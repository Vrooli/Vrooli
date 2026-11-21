import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import { onProfilerRender } from "./lib/profiler";
import "./styles.css";

const queryClient = new QueryClient();

if (window.top !== window.self) {
  initIframeBridgeChild();
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      {/* Top-level Profiler boundary. Inert in regular prod (react-dom strips
          the profiling hook); emits user_timing entries via onProfilerRender
          when the perf-build channel is active. See lib/profiler.ts. Add
          inner <Profiler> boundaries around heavy subtrees as needed; do
          not remove this one. */}
      <React.Profiler id="App" onRender={onProfilerRender}>
        <App />
      </React.Profiler>
    </QueryClientProvider>
  </React.StrictMode>
);
