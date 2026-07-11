import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

// INTEROP-CRITICAL: iframe bridge must be initialized before React mount so
// embedded web-console/app-monitor shells can negotiate parent-child messaging.
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "test-genie" });
}

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("root element not found");
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
