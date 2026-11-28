import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getInjectedConfig } from "@vrooli/api-base";
import { recipientsClient, registerBrowserPushSubscription } from "../../api/notifications";
import { forceServiceWorkerUpdate, readConnectionStatus, type ConnectionStatus } from "../../api/serviceWorker";
import { devicesQuery, readFresh } from "../settings/ownerQueries";
import { base64UrlToBytes, isThisBrowser } from "./webPush";

/**
 * How an attempt to turn on notifications for this browser ended. Every ending
 * is a fact from a read-back or from the browser, never a guess from a write.
 */
export type EnrollResult =
  | "registered"
  | "registeredNoSubscription"
  | "unsupported"
  | "denied"
  | "dismissed"
  | "noKey"
  | "unconfirmed"
  | "failed";

export const CONNECTION_STATUS_KEY = ["connection-status"] as const;

/** This browser's push state, and the one flow that turns notifications on. */
export function useBrowserEnrollment() {
  const queryClient = useQueryClient();
  const status = useQuery({ queryKey: CONNECTION_STATUS_KEY, queryFn: readConnectionStatus, refetchOnWindowFocus: true });
  const update = useMutation({ mutationFn: forceServiceWorkerUpdate, onSuccess: () => void status.refetch() });
  const [phase, setPhase] = useState<"idle" | "asking" | "registering">("idle");
  const [result, setResult] = useState<EnrollResult | null>(null);

  const refreshStatus = () => queryClient.fetchQuery<ConnectionStatus>({ queryKey: CONNECTION_STATUS_KEY, queryFn: readConnectionStatus, staleTime: 0 }).catch(() => undefined);

  /** Ask for permission when needed. Returns an ending, or null to carry on. */
  const askPermission = async (browser: ConnectionStatus): Promise<EnrollResult | null> => {
    if (!browser.supported) return "unsupported";
    if (browser.permission === "denied") return "denied";
    if (browser.permission !== "default") return null;
    if (typeof Notification === "undefined") return "unsupported";
    setPhase("asking");
    let permission: NotificationPermission;
    try {
      permission = await Notification.requestPermission();
    } catch {
      return "failed";
    } finally {
      setPhase("idle");
    }
    if (permission === "granted") return null;
    await refreshStatus();
    return permission === "denied" ? "denied" : "dismissed";
  };

  const publicKey = async (): Promise<string | undefined> => {
    const cached = queryClient.getQueryData<{ vapidPublicKey?: string }>(devicesQuery.queryKey)?.vapidPublicKey;
    const fetched = cached || (await recipientsClient.listDevices({})).vapidPublicKey;
    const injected = (getInjectedConfig() as { VAPID_PUBLIC_KEY?: string } | undefined)?.VAPID_PUBLIC_KEY;
    return fetched || injected || (import.meta.env.VITE_VAPID_PUBLIC_KEY as string | undefined) || undefined;
  };

  const enable = async () => {
    setResult(null);
    const browser = status.data;
    if (!browser) { setResult("failed"); return; }
    const early = await askPermission(browser);
    if (early) { setResult(early); return; }

    let accepted = false;
    try {
      const key = await publicKey();
      if (!key) { setResult("noKey"); return; }
      setPhase("registering");
      await registerBrowserPushSubscription(base64UrlToBytes(key) as unknown as BufferSource);
      accepted = true;
    } catch {
      // The Hub's own list below says whether this browser is registered.
    } finally {
      setPhase("idle");
    }

    try {
      const latest = await readFresh(queryClient, devicesQuery);
      if (!latest.devices.some(isThisBrowser)) { setResult(accepted ? "unconfirmed" : "failed"); await refreshStatus(); return; }
    } catch {
      setResult(accepted ? "unconfirmed" : "failed");
      return;
    }
    const connection = await refreshStatus();
    setResult(connection && !connection.subscribed ? "registeredNoSubscription" : "registered");
  };

  return { status, update, phase, result, enable, dismissResult: () => setResult(null) };
}
