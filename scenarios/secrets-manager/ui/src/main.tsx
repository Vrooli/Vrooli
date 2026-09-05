// INTEROP-CRITICAL: interop-sensitive configuration below — do not remove without checking host-frame embedding.
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { HashRouter } from "react-router-dom";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import "./styles.css";
import { onProfilerRender } from "./lib/profiler";

const queryClient = new QueryClient();

initSpatialNav();

if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("./sw.js", { scope: "./" }).catch((error: unknown) => {
      console.warn("Secrets Manager service worker registration failed", error);
    });
  });
}

if (typeof window !== "undefined" && window.parent !== window) {
  initIframeBridgeChild(
    {
      appId: "secrets-manager",
      captureLogs: true,
      captureNetwork: true
    }
  );
}

const root = document.getElementById("root");
if (!root) {
  throw new Error("Secrets Manager could not find its root element");
}

ReactDOM.createRoot(root).render(
  <React.StrictMode>
    <HashRouter>
      <QueryClientProvider client={queryClient}>
        <React.Profiler id="App" onRender={onProfilerRender}>
          <App />
        </React.Profiler>
      </QueryClientProvider>
    </HashRouter>
  </React.StrictMode>
);
