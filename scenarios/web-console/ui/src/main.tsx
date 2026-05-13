import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import "./i18n";
import "./styles.css";

const queryClient = new QueryClient();

// INTEROP-CRITICAL: iframe bridge must be initialized before React mount
// so parent scenarios receive the ready signal and can coordinate routing.
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "web-console" });
}

const rootEl = document.getElementById("root");
if (!rootEl) throw new Error("Missing #root element in index.html");
ReactDOM.createRoot(rootEl).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
