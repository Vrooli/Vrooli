import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1/1.0.1";
import { i18n } from "./i18n";
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
import { installChunkReloadGuard } from "@vrooli/api-base";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import { API_BASE } from "./lib/api-client";
import { describeError } from "./lib/error-utils";
import { ToastProvider } from "./components/ui/toast-provider";
import { AudioToolsProvider, createAudioToolsClient, registerVoiceTransport, useHydrateVoiceConfig } from "./audio-integration";
import { fetchAudioToolsDiscovery } from "./api/discovery";
import { AudioUnavailableBanner } from "./components/AudioUnavailableBanner";
import "./styles.css";

// Every route is code-split via lazy(); after a rebuild the old hashed
// chunks are gone, so a tab opened before the deploy would crash on its
// next navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard();

// Retry policy is a correctness concern, not a tuning knob: the default
// blind 3x retry turns a 404 or a validation refusal into three identical
// round-trips and delays the error the operator needs to see by seconds.
// Only genuinely transient failures are worth repeating.
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => failureCount < 2 && describeError(error).canRetry,
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 8000),
    },
    // Mutations are not retried automatically. Many are non-idempotent
    // (start a run, add a target) and a silent repeat could double-apply;
    // useActionMutation offers the operator an explicit Retry instead.
    mutations: { retry: false },
  },
});

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

registerVoiceTransport();

// AUDIO-TOOLS DISCOVERY: ask the swarm-manager backend whether audio-tools is
// reachable BEFORE React mounts. When it is, the browser still talks only to
// swarm-manager's same-origin API; the backend owns the audio-tools hop.
// A discovery failure mounts the provider with a stable unavailableReason so
// consumer hooks render typed errors and the Audio settings tab can surface a
// banner.
function sameOriginAudioBaseUrl(): string {
  return API_BASE;
}

async function bootstrapAudioTools(): Promise<{ baseUrl: string; unavailableReason: string }> {
  try {
    const ep = await fetchAudioToolsDiscovery();
    if (ep.available && ep.baseUrl) {
      return {
        baseUrl: sameOriginAudioBaseUrl(),
        unavailableReason: "",
      };
    }
    return { baseUrl: sameOriginAudioBaseUrl(), unavailableReason: ep.unavailableReason || "discovery_failed" };
  } catch {
    return { baseUrl: sameOriginAudioBaseUrl(), unavailableReason: "discovery_failed" };
  }
}

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element not found - check index.html has <div id=\"root\"></div>");
}

// Hydrate audio-tools' stream config into useVoiceConfigStore on mount.
// Must live inside AudioToolsProvider so the hook can read the
// unavailableReason context. The component renders no DOM itself.
// eslint-disable-next-line react-refresh/only-export-components -- entry-point helper, not an HMR boundary
function VoiceConfigHydrator({ children }: { children: React.ReactNode }) {
  useHydrateVoiceConfig();
  return <>{children}</>;
}

void bootstrapAudioTools().then(({ unavailableReason }) => {
  // Same-origin client — calls swarm-manager's own AudioAdminService +
  // AudioRuntimeService; the server owns the audio-tools hop.
  const audioToolsClient = createAudioToolsClient();

  ReactDOM.createRoot(rootElement).render(
    // vrooli:library-strings-provider start
    <LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>
<React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <AudioToolsProvider client={audioToolsClient} unavailableReason={unavailableReason || undefined}>
            <AudioUnavailableBanner reason={unavailableReason || undefined} className="m-3" />
            <VoiceConfigHydrator>
              <App />
            </VoiceConfigHydrator>
          </AudioToolsProvider>
        </ToastProvider>
      </QueryClientProvider>
    </React.StrictMode>
    </LibraryStringsProvider>,
    // vrooli:library-strings-provider end
  );
});
