/**
 * Service-worker + push observability for the Settings "Connection status"
 * panel. iOS home-screen web apps are opaque about whether a new worker
 * actually activated and whether this device holds a live push subscription —
 * so these helpers read that state directly and let the UI show it (and force a
 * stuck update through) instead of the operator guessing.
 */

export interface ConnectionStatus {
  /** The browser supports the APIs push requires at all (false in a plain iOS Safari tab). */
  supported: boolean;
  /** Notification.permission: "granted" | "denied" | "default" | "unsupported". */
  permission: string;
  /** A worker is registered and controlling this page. */
  controlled: boolean;
  /** Version reported by the RUNNING worker (null if none answered). */
  runningVersion: string | null;
  /** Version in the freshly-fetched sw.js on the server (null if unreadable). */
  availableVersion: string | null;
  /** A newer worker is deployed than the one running — the operator should update. */
  stale: boolean;
  /** This device holds a push subscription (endpoint present). */
  subscribed: boolean;
  /** The push endpoint host, for display (e.g. "web.push.apple.com"); "" when none. */
  endpointHost: string;
}

const VERSION_TIMEOUT_MS = 2000;

function permissionState(): string {
  if (typeof Notification === "undefined") return "unsupported";
  return Notification.permission;
}

/** Ask the active worker its version over a MessageChannel; null if it never answers. */
export async function runningServiceWorkerVersion(): Promise<string | null> {
  if (!("serviceWorker" in navigator)) return null;
  const registration = await navigator.serviceWorker.getRegistration();
  const active = registration?.active;
  if (!active) return null;
  return new Promise<string | null>((resolve) => {
    const channel = new MessageChannel();
    const timer = setTimeout(() => resolve(null), VERSION_TIMEOUT_MS);
    channel.port1.onmessage = (event: MessageEvent) => {
      clearTimeout(timer);
      const version = (event.data as { version?: string } | null)?.version;
      resolve(typeof version === "string" ? version : null);
    };
    active.postMessage({ type: "GET_VERSION" }, [channel.port2]);
  });
}

/** Read the deployed sw.js (bypassing cache) and parse its SW_VERSION literal. */
export async function availableServiceWorkerVersion(): Promise<string | null> {
  try {
    const response = await fetch("sw.js", { cache: "no-store" });
    if (!response.ok) return null;
    const source = await response.text();
    const match = source.match(/SW_VERSION\s*=\s*["']([^"']+)["']/);
    return match?.[1] ?? null;
  } catch {
    return null;
  }
}

async function pushEndpointHost(): Promise<string> {
  if (!("serviceWorker" in navigator)) return "";
  const registration = await navigator.serviceWorker.getRegistration();
  const subscription = await registration?.pushManager?.getSubscription?.();
  const endpoint = subscription?.endpoint;
  if (!endpoint) return "";
  try {
    return new URL(endpoint).host;
  } catch {
    return "";
  }
}

/** Snapshot every signal the Connection status panel shows. Never throws. */
export async function readConnectionStatus(): Promise<ConnectionStatus> {
  const supported = "serviceWorker" in navigator && typeof PushManager !== "undefined";
  const controlled = "serviceWorker" in navigator && Boolean(navigator.serviceWorker.controller);
  const [runningVersion, availableVersion, endpointHost] = await Promise.all([
    runningServiceWorkerVersion(),
    availableServiceWorkerVersion(),
    pushEndpointHost(),
  ]);
  // Stale when a worker is in control, we know what version is DEPLOYED, and
  // the running worker isn't it — including a running worker too old to report
  // a version at all (runningVersion null), which is the common iOS "it never
  // swapped" case. Only an unreadable deployed version (offline) suppresses the
  // flag, since then we genuinely can't tell.
  const stale = Boolean(controlled && availableVersion && runningVersion !== availableVersion);
  return {
    supported,
    permission: permissionState(),
    controlled,
    runningVersion,
    availableVersion,
    stale,
    subscribed: endpointHost !== "",
    endpointHost,
  };
}

/**
 * Force a stuck worker update through: re-check the server, promote a waiting
 * worker with SKIP_WAITING, and reload once it takes control so the new code is
 * actually running. Resolves to true if an update path was triggered.
 */
export async function forceServiceWorkerUpdate(): Promise<boolean> {
  if (!("serviceWorker" in navigator)) return false;
  const registration = await navigator.serviceWorker.getRegistration();
  if (!registration) return false;
  navigator.serviceWorker.addEventListener(
    "controllerchange",
    () => window.location.reload(),
    { once: true },
  );
  await registration.update();
  const pending = registration.waiting ?? registration.installing;
  if (pending) {
    pending.postMessage({ type: "SKIP_WAITING" });
    return true;
  }
  return false;
}
