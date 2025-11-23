import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

// Initialize iframe bridge (unconditional for UI smoke tests)
initIframeBridgeChild({ appId: "deployment-manager" });
// Manually set bridge flag for non-iframe contexts (e.g., Browserless smoke tests)
if (typeof window !== "undefined") {
  (window as any).__vrooliBridgeChildInstalled = true;
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
