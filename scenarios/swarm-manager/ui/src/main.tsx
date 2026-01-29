import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

if (window.top !== window.self) {
  initIframeBridgeChild();
}

const ensureSEO = () => {
  const head = document.head;
  const canonicalUrl = `${window.location.origin}${window.location.pathname}`;

  let canonical = head.querySelector<HTMLLinkElement>('link[rel="canonical"]');
  if (!canonical) {
    canonical = document.createElement("link");
    canonical.rel = "canonical";
    head.appendChild(canonical);
  }
  canonical.href = canonicalUrl;

  let ogUrl = head.querySelector<HTMLMetaElement>('meta[property="og:url"]');
  if (!ogUrl) {
    ogUrl = document.createElement("meta");
    ogUrl.setAttribute("property", "og:url");
    head.appendChild(ogUrl);
  }
  ogUrl.setAttribute("content", canonicalUrl);
};

ensureSEO();

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element not found - check index.html has <div id=\"root\"></div>");
}
ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
