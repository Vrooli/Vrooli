import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

if (typeof window !== "undefined" && window.parent !== window) {
  initIframeBridgeChild({
    appId: "secrets-manager",
    enableConsoleCapture: true,
    enableNetworkCapture: true
  });
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
