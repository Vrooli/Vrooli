import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import App from "./App";
import "./styles.css";

declare global {
  interface Window {
    __landingManagerBridgeInitialized?: boolean;
    __landingManagerPerformance?: {
      initial_load_time_ms?: number;
      lcp_ms?: number;
    };
  }
}

// QueryClient with performance-optimized caching
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes - reduce unnecessary refetches
      gcTime: 10 * 60 * 1000,   // 10 minutes - keep data in cache longer
      refetchOnWindowFocus: false, // Don't refetch on window focus (factory UI is local)
      refetchOnReconnect: false,   // Don't refetch on reconnect (local API)
      retry: 1, // Only retry once on failure (local API should be fast)
    },
  },
});

if (typeof window !== 'undefined' && window.parent !== window && !window.__landingManagerBridgeInitialized) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    // Unable to parse parent origin - will use default
  }

  initIframeBridgeChild({
    parentOrigin,
    appId: "landing-manager",
    captureLogs: {
      enabled: true,
      streaming: true,
      levels: ['log', 'info', 'warn', 'error', 'debug'],
      bufferSize: 100
    },
    captureNetwork: {
      enabled: true,
      streaming: true,
      bufferSize: 100
    }
  });
  window.__landingManagerBridgeInitialized = true;
}

// Performance measurement utilities
if (typeof window !== 'undefined') {
  window.__landingManagerPerformance = {};

  // Measure initial load time (DOMContentLoaded to fully interactive)
  const measureLoadTime = () => {
    const perfData = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
    if (perfData) {
      window.__landingManagerPerformance!.initial_load_time_ms = Math.round(perfData.loadEventEnd - perfData.fetchStart);
    }
  };

  // Measure Largest Contentful Paint (LCP)
  const measureLCP = () => {
    if ('PerformanceObserver' in window) {
      const lcpObserver = new PerformanceObserver((entryList) => {
        const entries = entryList.getEntries();
        const lastEntry = entries[entries.length - 1] as PerformanceEntry;
        if (lastEntry && lastEntry.startTime) {
          window.__landingManagerPerformance!.lcp_ms = Math.round(lastEntry.startTime);
        }
      });

      try {
        lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true });
      } catch (e) {
        // LCP not supported - ignore
      }
    }
  };

  // Capture metrics after page fully loads
  if (document.readyState === 'complete') {
    measureLoadTime();
    measureLCP();
  } else {
    window.addEventListener('load', () => {
      setTimeout(() => {
        measureLoadTime();
        measureLCP();
      }, 0);
    });
  }
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
