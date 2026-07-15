import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { installChunkReloadGuard } from "@vrooli/api-base";
import { initIframeBridgeChild, initSpatialNav } from "@vrooli/iframe-bridge";
import App from "./App";
import { onProfilerRender } from "./lib/profiler";
import { CheckMetadataProvider } from "./shared/contexts/CheckMetadataContext";
import "./shared/theme/tokens.css";
import "./styles.css";

const queryClient = new QueryClient();
const rootElement = document.getElementById("root");

// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe bridge initialization              ║
// ║                                                              ║
// ║  This must run before React mounts so embedded Vrooli hosts  ║
// ║  can establish the child bridge and storage shims before UI  ║
// ║  code starts reading browser APIs.                           ║
// ╚══════════════════════════════════════════════════════════════╝
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "vrooli-autoheal" });
}

initSpatialNav();

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard();

if (!rootElement) {
  throw new Error("Root element not found");
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <CheckMetadataProvider>
        <React.Profiler id="App" onRender={onProfilerRender}>
          <App />
        </React.Profiler>
      </CheckMetadataProvider>
    </QueryClientProvider>
  </React.StrictMode>
);
