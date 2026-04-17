import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();
// spatial-nav: disabled

// INTEROP-CRITICAL: iframe bridge initialization for App Monitor embedding
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "vrooli-onboarding" });
}

const rootElement = document.getElementById("root");

if (!rootElement) {
  throw new Error("Root element '#root' was not found");
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
