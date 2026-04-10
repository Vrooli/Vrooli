import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
// INTEROP-CRITICAL: iframe-bridge must be initialized before React render for parent-frame communication
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

// INTEROP-CRITICAL: detect iframe embedding and initialize bridge for web-console integration
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "tunnel-manager" });
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
