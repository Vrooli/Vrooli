import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { KeyboardEvent, MouseEvent, WheelEvent } from "react";
import type { DesktopReadiness, DesktopSession } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { DesktopPane } from "./DesktopPane";
import { createBridgeInteractiveTransport, createDesktopPeer, createDesktopSessionTransport, type DesktopPeer } from "./desktopTransport";

type JsonRecord = Record<string, unknown>;

const OFFER = 1;
const ICE = 3;
const TO_DESKTOP = 1;
const FROM_DESKTOP = 2;
const MAX_CLIPBOARD_BYTES = 64 * 1024;
const SURFACE = (nodeId: string) => ({
  target: { ownerScenario: "web-console", resourceId: nodeId, hostNodeId: nodeId },
  ownerScenario: "device-control",
  surfaceId: "desktop",
});

type DesktopInputEvent = MouseEvent<HTMLVideoElement> | KeyboardEvent<HTMLVideoElement> | WheelEvent<HTMLVideoElement>;

export async function cleanupDesktopOpenFailure(
  bridge: Pick<ReturnType<typeof createBridgeInteractiveTransport>, "close">,
  transport: Pick<ReturnType<typeof createDesktopSessionTransport>, "close">,
  grant: JsonRecord | undefined,
  session: DesktopSession | undefined,
): Promise<void> {
  if (grant) {
    try { await bridge.close({ grant, reason: "web_console_open_failed" }); } catch { /* cleanup is best effort */ }
  }
  if (session) {
    try { await transport.close({ session: session.ref, reason: "web_console_open_failed" }); } catch { /* cleanup is best effort */ }
  }
}

function bytesToBase64(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary);
}

function base64ToJson(value: unknown): RTCSessionDescriptionInit {
  if (typeof value !== "string" || value.length === 0) throw new Error("Desktop signaling response was empty.");
  const binary = atob(value);
  const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
  return JSON.parse(new TextDecoder().decode(bytes)) as RTCSessionDescriptionInit;
}

function requestId(): string {
  return globalThis.crypto.randomUUID();
}

export function DesktopMachineTab({ nodeId }: { nodeId?: string }) {
  const [readiness, setReadiness] = useState<DesktopReadiness>();
  const [session, setSession] = useState<DesktopSession>();
  const [mediaStream, setMediaStream] = useState<MediaStream>();
  const [error, setError] = useState("");
  const peerRef = useRef<DesktopPeer>();
  const channelGrantRef = useRef<JsonRecord>();
  const sessionRef = useRef<DesktopSession>();
  const dataWaitersRef = useRef(new Map<string, (response: JsonRecord) => void>());
  const transport = useMemo(() => nodeId ? createDesktopSessionTransport(`/api/v1/desktop/${encodeURIComponent(nodeId)}`) : null, [nodeId]);
  const bridgeTransport = useMemo(() => nodeId ? createBridgeInteractiveTransport("/api/v1/desktop-bridge") : null, [nodeId]);
  const setSessionState = (next: DesktopSession | undefined) => {
    sessionRef.current = next;
    setSession(next);
  };

  const refreshReadiness = useCallback(async () => {
    if (!transport || !bridgeTransport || !nodeId) return;
    try {
      setError("");
      setReadiness(await transport.readiness({ surface: SURFACE(nodeId) }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Desktop readiness is unavailable.");
    }
  }, [bridgeTransport, nodeId, transport]);

  const close = useCallback(async () => {
    peerRef.current?.close();
    peerRef.current = undefined;
    const bridge = bridgeTransport;
    if (bridge && channelGrantRef.current) {
      try { await bridge.close({ grant: channelGrantRef.current, reason: "web_console_unmount" }); } catch { /* cleanup is best effort */ }
      channelGrantRef.current = undefined;
    }
    const activeSession = sessionRef.current;
    if (transport && activeSession) {
      try { await transport.close({ session: activeSession.ref, reason: "web_console_unmount" }); } catch { /* cleanup is best effort */ }
    }
    setMediaStream(undefined);
    dataWaitersRef.current.clear();
    setSessionState(undefined);
  }, [bridgeTransport, transport]);

  const handleDataResponse = (payload: string) => {
    try {
      const response = JSON.parse(payload) as JsonRecord;
      const receipt = typeof response.receipt === "object" && response.receipt !== null ? response.receipt as JsonRecord : undefined;
      const commandId = typeof receipt?.commandId === "string" ? receipt.commandId : "";
      if (commandId) dataWaitersRef.current.get(commandId)?.(response);
      if (typeof response.error === "string" && response.error.length > 0) setError(`Desktop data request was rejected: ${response.error}`);
    } catch {
      setError("Desktop data-channel response was invalid.");
    }
  };

  const sendDataRequest = async (kind: string, request: JsonRecord): Promise<{ sent: boolean; response?: JsonRecord }> => {
    const peer = peerRef.current;
    const commandId = request.commandId;
    if (!peer || typeof commandId !== "string") return { sent: false };
    const response = new Promise<JsonRecord | undefined>((resolve) => {
      const timer = setTimeout(() => {
        dataWaitersRef.current.delete(commandId);
        resolve(undefined);
      }, 5000);
      dataWaitersRef.current.set(commandId, (value) => {
        clearTimeout(timer);
        dataWaitersRef.current.delete(commandId);
        resolve(value);
      });
    });
    const sent = await peer.sendData(JSON.stringify({ kind, request }));
    if (!sent) {
      dataWaitersRef.current.delete(commandId);
      return { sent: false };
    }
    return { sent: true, response: await response };
  };

  useEffect(() => {
    void refreshReadiness();
    return () => {
      peerRef.current?.close();
      peerRef.current = undefined;
      void close();
    };
  }, [close, nodeId, refreshReadiness, transport]);

  const open = async (control = true) => {
    if (!transport || !bridgeTransport || !nodeId) return;
    let openedSession: DesktopSession | undefined;
    let openedGrant: JsonRecord | undefined;
    const bridge = bridgeTransport;
    try {
      setError("");
      const opened = await transport.open({ surface: SURFACE(nodeId), displayId: readiness?.selectedDisplayId ?? "", ttlSeconds: 300, control, clipboard: control, requestId: requestId() });
      openedSession = opened;
      setSessionState(opened);
      const sessionId = opened.ref?.sessionId ?? "";
      const channel = await bridge.open({ nodeId, surface: SURFACE(nodeId), sessionId, leaseId: opened.leaseId, leaseEpoch: opened.leaseEpoch, protocol: 1, role: control ? 1 : 2, takeover: false, ttlSeconds: 300, requestId: requestId() }) as JsonRecord;
      const grant = channel.grant;
      openedGrant = grant as JsonRecord | undefined;
      channelGrantRef.current = grant as JsonRecord;
      const routes = Array.isArray(channel.routes)
        ? channel.routes.filter((route): route is JsonRecord => typeof route === "object" && route !== null)
        : [];
      const peer = createDesktopPeer(setMediaStream, undefined, routes.map((route) => ({
        url: typeof route.url === "string" ? route.url : undefined,
        username: typeof route.username === "string" ? route.username : undefined,
        credential: typeof route.credential === "string" ? route.credential : undefined,
      })), (connectionState) => {
        const current = sessionRef.current;
        if (!current || connectionState === "connected" || connectionState === "new") return;
        setSessionState({ ...current, connectionState: connectionState === "failed" ? "failed" : "disconnected" });
      }, handleDataResponse);
      peerRef.current = peer;
      const generation = requestId();
      await peer.connect({
        sendOffer: async (offer) => {
          const response = await bridge.signal({ grant, kind: OFFER, generation, payload: bytesToBase64(JSON.stringify(offer)), requestId: requestId() }) as JsonRecord;
          return base64ToJson(response.payload);
        },
        sendIce: async (candidate) => {
          await bridge.signal({ grant, kind: ICE, generation, payload: bytesToBase64(JSON.stringify(candidate)), requestId: requestId() });
        },
      });
      setSessionState({ ...opened, connectionState: "connected" });
    } catch (cause) {
      peerRef.current?.close();
      peerRef.current = undefined;
      await cleanupDesktopOpenFailure(bridge, transport, openedGrant, openedSession);
      channelGrantRef.current = undefined;
      setMediaStream(undefined);
      setSessionState(undefined);
      setError(cause instanceof Error ? cause.message : "Desktop connection failed.");
    }
  };

  const sendInput = async (event: DesktopInputEvent) => {
    if (!transport || !session || event.currentTarget.tagName !== "VIDEO") return;
    const video = event.currentTarget;
    const rect = video.getBoundingClientRect();
    const action = "key" in event
      ? { key: { kind: event.type === "keyup" ? 2 : 1, key: event.key } }
      : (() => {
        const display = readiness?.displays.find((candidate) => candidate.id === session.selectedDisplayId);
        const x = Math.max(0, Math.min(display?.width ?? rect.width, ((event.clientX - rect.left) / Math.max(rect.width, 1)) * (display?.width ?? rect.width)));
        const y = Math.max(0, Math.min(display?.height ?? rect.height, ((event.clientY - rect.top) / Math.max(rect.height, 1)) * (display?.height ?? rect.height)));
        return "deltaY" in event
          ? { wheel: { displayId: session.selectedDisplayId, x, y, horizontalTicks: Math.sign(event.deltaX) * Math.min(20, Math.max(1, Math.round(Math.abs(event.deltaX) / 100))), verticalTicks: Math.sign(event.deltaY) * Math.min(20, Math.max(1, Math.round(Math.abs(event.deltaY) / 100))) } }
          : { pointer: { kind: event.type === "pointerdown" ? 2 : event.type === "pointerup" ? 3 : event.type === "click" ? 4 : 1, displayId: session.selectedDisplayId, x, y, button: event.type === "pointermove" ? 0 : event.button === 2 ? 2 : event.button === 1 ? 3 : 1 } };
      })();
    const request = { session: session.ref, leaseId: session.leaseId, leaseEpoch: session.leaseEpoch, commandId: requestId(), displayId: session.selectedDisplayId, geometryRevision: (session as DesktopSession & { geometryRevision?: string }).geometryRevision ?? "", action };
    try {
      const dataResult = await sendDataRequest("input", request as JsonRecord);
      if (dataResult.sent) {
        if (!dataResult.response) setError("Desktop input outcome is unknown; do not retry automatically.");
        return;
      }
      await transport.input(request);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Desktop input was rejected.");
    }
  };

  const takeControl = async () => {
    if (!transport || !session) return;
    try { setSessionState(await transport.takeControl({ session: session.ref, takeover: true })); } catch (cause) { setError(cause instanceof Error ? cause.message : "Control lease was not granted."); }
  };

  const sendClipboard = async () => {
    if (!transport || !session || !("clipboard" in navigator) || typeof navigator.clipboard.readText !== "function") return;
    try {
      const text = await navigator.clipboard.readText();
      if (new TextEncoder().encode(text).byteLength > MAX_CLIPBOARD_BYTES) {
        throw new Error("Clipboard text exceeds the 64 KiB desktop limit.");
      }
      const request = { session: session.ref, leaseId: session.leaseId, leaseEpoch: session.leaseEpoch, direction: TO_DESKTOP, text, commandId: requestId() };
      const dataResult = await sendDataRequest("clipboard", request as JsonRecord);
      if (dataResult.sent) {
        if (!dataResult.response) setError("Clipboard outcome is unknown; do not retry automatically.");
        return;
      }
      await transport.clipboard(request);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Clipboard write was rejected."); }
  };

  const readClipboard = async () => {
    if (!transport || !session || !("clipboard" in navigator) || typeof navigator.clipboard.writeText !== "function") return;
    try {
      const request = { session: session.ref, leaseId: session.leaseId, leaseEpoch: session.leaseEpoch, direction: FROM_DESKTOP, commandId: requestId() };
      const dataResult = await sendDataRequest("clipboard", request as JsonRecord);
      const receipt = dataResult.sent
        ? (dataResult.response?.receipt as JsonRecord | undefined)
        : await transport.clipboard(request) as JsonRecord;
      if (dataResult.sent && !dataResult.response) {
        setError("Clipboard outcome is unknown; do not retry automatically.");
        return;
      }
      if (typeof receipt?.text === "string") await navigator.clipboard.writeText(receipt.text);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Clipboard read was rejected."); }
  };

  if (!nodeId) return <DesktopPane readiness={{ state: 1, reasonCode: "node_id_missing", recovery: "Select a Bridge agent with a desktop node identity." } as DesktopReadiness} />;
  return <div className="flex flex-col gap-2">
    {error && <p role="alert" className="rounded border border-rose-400/30 bg-rose-400/10 p-2 text-xs text-rose-100">{error}</p>}
    <DesktopPane readiness={readiness} session={session} mediaStream={mediaStream} onOpen={() => { void open(true); }} onOpenViewer={() => { void open(false); }} onRefresh={() => { void refreshReadiness(); }} onReconnect={() => { void close().then(refreshReadiness); }} onTakeControl={() => { void takeControl(); }} onRevoke={() => { void close(); }} onInput={(event) => { void sendInput(event); }} onClipboardToDesktop={() => { void sendClipboard(); }} onClipboardFromDesktop={() => { void readClipboard(); }} />
  </div>;
}
