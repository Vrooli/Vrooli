import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import { onProfilerRender } from "./lib/profiler";
import "./styles.css";

const queryClient = new QueryClient();

if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "scenario-to-cloud" });
}

// INTEROP-CRITICAL: initialize keyboard/gamepad focus routing for embedded
// and desktop-hosted surfaces before the React tree mounts.
initSpatialNav();

if (import.meta.env.PROD && "serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js").catch((error: unknown) => {
      console.warn("Service worker registration failed", error);
    });
  });
}

const root = document.getElementById("root");
if (!root) {
  throw new Error("scenario-to-cloud UI root element is missing");
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
