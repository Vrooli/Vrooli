// ╔══════════════════════════════════════════════════════════════╗
// ║  Runtime polyfills for older embedded browsers               ║
// ║                                                              ║
// ║  Vite's build.target only transpiles syntax — it does NOT    ║
// ║  polyfill missing runtime APIs. These polyfills fill gaps    ║
// ║  for browsers like Google TV (Chromium <76).                 ║
// ║                                                              ║
// ║  MUST run before any imports that may use these APIs.        ║
// ╚══════════════════════════════════════════════════════════════╝

if (typeof Promise.allSettled !== "function") {
  Promise.allSettled = function allSettled<T extends readonly unknown[]>(
    promises: T,
  ): Promise<{ -readonly [K in keyof T]: PromiseSettledResult<Awaited<T[K]>> }> {
    return Promise.all(
      Array.from(promises).map((p) =>
        Promise.resolve(p).then(
          (value) => ({ status: "fulfilled" as const, value }),
          (reason: unknown) => ({ status: "rejected" as const, reason }),
        ),
      ),
    ) as never;
  };
}

import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import { AudioToolsProvider, createAudioToolsClient } from "./audio-integration";
import { fetchAudioToolsDiscovery } from "./api/discovery";
import "./styles.css";

const queryClient = new QueryClient();

// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe bridge initialization              ║
// ║                                                              ║
// ║  Must run BEFORE React mount so that:                        ║
// ║  1. Storage shimming is in place before any component        ║
// ║     accesses localStorage/sessionStorage                     ║
// ║  2. The bridge message channel is ready for host commands    ║
// ║                                                              ║
// ║  The window.parent check ensures this is a no-op when        ║
// ║  running outside an iframe (localhost, tunnel).              ║
// ╚══════════════════════════════════════════════════════════════╝

declare global {
  interface Window {
    __swarmManagerBridgeInitialized?: boolean;
  }
}

if (
  typeof window !== "undefined" &&
  window.parent !== window &&
  !window.__swarmManagerBridgeInitialized
) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Fall back to default origin when parsing fails.
  }

  initIframeBridgeChild({ parentOrigin, appId: "swarm-manager" });
  window.__swarmManagerBridgeInitialized = true;
}

initSpatialNav();

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

// AUDIO-TOOLS DISCOVERY: resolve audio-tools' base URL via the
// swarm-manager backend BEFORE React mounts so AudioToolsProvider
// wires the client against the right host. A discovery failure mounts
// the provider with a sentinel base URL ("http://localhost:0") + a
// stable unavailableReason so consumer hooks render typed errors and
// the Audio settings tab can surface a banner.
async function bootstrapAudioTools(): Promise<{ baseUrl: string; unavailableReason: string }> {
  try {
    const ep = await fetchAudioToolsDiscovery();
    if (ep.available && ep.baseUrl) {
      return { baseUrl: ep.baseUrl, unavailableReason: "" };
    }
    return { baseUrl: "", unavailableReason: ep.unavailableReason || "discovery_failed" };
  } catch {
    return { baseUrl: "", unavailableReason: "discovery_failed" };
  }
}

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element not found - check index.html has <div id=\"root\"></div>");
}

void bootstrapAudioTools().then(({ baseUrl, unavailableReason }) => {
  const audioToolsClient = createAudioToolsClient({
    baseUrl: baseUrl || "http://localhost:0",
  });

  ReactDOM.createRoot(rootElement).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <AudioToolsProvider client={audioToolsClient} unavailableReason={unavailableReason || undefined}>
          <App />
        </AudioToolsProvider>
      </QueryClientProvider>
    </React.StrictMode>
  );
});
