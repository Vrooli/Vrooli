import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

// INTEROP-CRITICAL: Initialize iframe bridge for web-console embedding
if (window.top !== window.self) {
  const parentOrigin = window.location.ancestorOrigins?.[0] ?? "*";
  initIframeBridgeChild({ parentOrigin, appId: "lifestyle-dashboard" });
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
