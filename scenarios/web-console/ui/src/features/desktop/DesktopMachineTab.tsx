import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { KeyboardEvent, MouseEvent, PointerEvent, WheelEvent } from "react";
import type { DesktopReadiness, DesktopSession } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { DesktopPane } from "./DesktopPane";
import { canControlDesktop } from "./WorkspaceSurface";
import { createBridgeInteractiveTransport, createDesktopPeer, desktopJSON, createDesktopSessionTransport, type DesktopPeer } from "./desktopTransport";
import { boundedClipboardText, mapSurfacePoint, OrderedLeaseInput, SingleFlightSurfaceOperation } from "./remoteSurface";
import { WorkspaceSurfaceHandoff } from "./workspaceSurfaceContract";

type JsonRecord = Record<string, unknown>;

const OFFER = 1;
const ICE = 3;
const TO_DESKTOP = 1;
const FROM_DESKTOP = 2;
const SURFACE = (nodeId: string) => ({
  target: { ownerScenario: "web-console", resourceId: nodeId, hostNodeId: nodeId },
  ownerScenario: "device-control",
  surfaceId: "desktop",
});

type DesktopInputTarget = HTMLVideoElement | HTMLImageElement;
type DesktopInputEvent = MouseEvent<DesktopInputTarget> | PointerEvent<DesktopInputTarget> | KeyboardEvent<DesktopInputTarget> | WheelEvent<DesktopInputTarget>;

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
  const [frame, setFrame] = useState<string>();
  const frameControllerRef = useRef<AbortController>();
  const [error, setError] = useState("");
  const peerRef = useRef<DesktopPeer>();
  const channelGrantRef = useRef<JsonRecord>();
  const sessionRef = useRef<DesktopSession>();
  const heldPointerRef = useRef<{ id: number; button: number; point: { x: number; y: number } }>();
  const heldKeysRef = useRef(new Set<string>());
  const dataWaitersRef = useRef(new Map<string, (response: JsonRecord) => void>());
  const inputQueueRef = useRef(new OrderedLeaseInput());
  const lifecycleRef = useRef(new SingleFlightSurfaceOperation());
  const handoffRef = useMemo(() => new WorkspaceSurfaceHandoff("device-control", nodeId ?? "unselected"), [nodeId]);
  const transport = useMemo(() => nodeId ? createDesktopSessionTransport(`/api/v1/desktop/${encodeURIComponent(nodeId)}`) : null, [nodeId]);
  const bridgeTransport = useMemo(() => nodeId ? createBridgeInteractiveTransport("/api/v1/desktop-bridge") : null, [nodeId]);
  const setSessionState = (next: DesktopSession | undefined) => {
    const previous = sessionRef.current;
    if (!next || !previous || next.leaseId !== previous.leaseId || next.leaseEpoch !== previous.leaseEpoch || !next.controller) {
      heldKeysRef.current.clear();
      heldPointerRef.current = undefined;
    }
    sessionRef.current = next;
    setSession(next);
    inputQueueRef.current.updateLease(next ? { id: next.leaseId, epoch: next.leaseEpoch, controller: next.controller } : undefined);
  };

  const denyLocalLeaseInput = () => {
    const active = sessionRef.current;
    if (active) setSessionState({ ...active, connectionState: "failed", controller: false, leaseId: "", leaseEpoch: 0n });
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

  const closeOwner = useCallback(async (reason = "user_closed") => {
    frameControllerRef.current?.abort();
    frameControllerRef.current = undefined;
    setFrame(undefined);
    const peer = peerRef.current;
    peerRef.current = undefined;
    peer?.close();
    const cleanupErrors: string[] = [];
    const bridge = bridgeTransport;
    if (bridge && channelGrantRef.current) {
      try { await bridge.close({ grant: channelGrantRef.current, reason }); channelGrantRef.current = undefined; }
      catch (cause) { cleanupErrors.push(cause instanceof Error ? cause.message : "Bridge channel close failed."); }
    }
    const activeSession = sessionRef.current;
    if (transport && activeSession) {
      try { await transport.close({ session: activeSession.ref, reason }); } catch (cause) { cleanupErrors.push(cause instanceof Error ? cause.message : "Device Control close failed."); }
    }
    if (cleanupErrors.length) throw new Error(cleanupErrors.join("; "));
    setMediaStream(undefined);
    dataWaitersRef.current.clear();
    setSessionState(undefined);
  }, [bridgeTransport, transport]);

  const close = useCallback(async (reason = "user_closed") => {
    const pending = handoffRef.dispatch({ type: "request", operation: "close" });
    if (pending.pending !== "close") return;
    await lifecycleRef.current.run("close", async () => {
      try {
        await closeOwner(reason);
        handoffRef.dispatch({ type: "owner-closed", owner: "device-control" });
      } catch (cause) {
        denyLocalLeaseInput();
        handoffRef.dispatch({ type: "owner-failed", owner: "device-control", message: cause instanceof Error ? cause.message : "Device Control close failed." });
        setError(cause instanceof Error ? cause.message : "Device Control close failed.");
      }
    });
  }, [closeOwner, handoffRef]);

  const handleDataResponse = (payload: string) => {
    try {
      const response = JSON.parse(payload) as JsonRecord;
      const receipt = typeof response.receipt === "object" && response.receipt !== null ? response.receipt as JsonRecord : undefined;
      const commandId = typeof receipt?.commandId === "string" ? receipt.commandId : "";
      if (commandId) dataWaitersRef.current.get(commandId)?.(response);
      if (typeof response.error === "string" && response.error.length > 0) {
        setError(`Desktop data request was rejected: ${response.error}`);
        if (/lease|revok|controller/i.test(response.error)) {
          const current = sessionRef.current;
          if (current) setSessionState({ ...current, connectionState: "revoked", controller: false, leaseId: "", leaseEpoch: 0n });
          handoffRef.dispatch({ type: "owner-revoked", owner: "device-control" });
        }
      }
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
    const sent = await peer.sendData(desktopJSON({ kind, request }));
    if (!sent) {
      dataWaitersRef.current.delete(commandId);
      return { sent: false };
    }
    return { sent: true, response: await response };
  };

  useEffect(() => {
    void refreshReadiness();
    return () => {
      const peer = peerRef.current;
      peerRef.current = undefined;
      peer?.close();
      handoffRef.dispatch({ type: "request", operation: "close" });
      void lifecycleRef.current.run("close", async () => {
        try {
          await closeOwner("web_console_unmount");
          handoffRef.dispatch({ type: "owner-closed", owner: "device-control" });
        } catch (cause) {
          handoffRef.dispatch({ type: "owner-failed", owner: "device-control", message: cause instanceof Error ? cause.message : "Device Control cleanup failed." });
        }
      });
    };
  }, [closeOwner, handoffRef, nodeId, refreshReadiness, transport]);

  const openOwner = async (control = true, ready = readiness, preferWebRTC = false) => {
    if (!transport || !bridgeTransport || !nodeId) return;
    let openedSession: DesktopSession | undefined;
    let openedGrant: JsonRecord | undefined;
    const bridge = bridgeTransport;
    try {
      setError("");
      const opened = await transport.open({ surface: SURFACE(nodeId), displayId: ready?.selectedDisplayId ?? "", ttlSeconds: 300, control, clipboard: control && Boolean(ready?.clipboardAllowed), requestId: requestId() });
      openedSession = opened;
      setSessionState(opened);
      if (!preferWebRTC && ready?.httpsFramesAllowed && transport.readFrame) {
        const capture = new AbortController();
        frameControllerRef.current = capture;
        const read = async () => {
          const current = sessionRef.current;
          if (!current || current.ref?.sessionId !== opened.ref?.sessionId) throw new Error("Desktop session closed.");
          const result = await transport.readFrame?.({ session: current.ref, leaseId: current.leaseId, leaseEpoch: current.leaseEpoch }, capture.signal);
          if (!result?.jpeg || result.jpeg.length > 122880) throw new Error("Desktop frame was invalid.");
          if (!capture.signal.aborted) setFrame(`data:image/jpeg;base64,${result.jpeg}`);
        };
        await read();
        setSessionState({ ...opened, codec: "JPEG over HTTPS", connectionState: "connected" });
        void (async () => {
          while (!capture.signal.aborted) {
            await new Promise<void>((resolve) => setTimeout(resolve, 220));
            if (capture.signal.aborted) return;
            try {
              await read();
            } catch (cause) {
              if (capture.signal.aborted) return;
              const current = sessionRef.current;
              if (current?.ref?.sessionId !== opened.ref?.sessionId) return;
              setError(cause instanceof Error ? cause.message : "Desktop frame stream disconnected.");
              if (current) setSessionState({ ...current, connectionState: "disconnected" });
              setFrame(undefined);
              return;
            }
          }
        })();
        return;
      }
      const sessionId = opened.ref?.sessionId ?? "";
      const channel = await bridge.open({ nodeId, surface: SURFACE(nodeId), sessionId, leaseId: opened.leaseId, leaseEpoch: opened.leaseEpoch, protocol: 1, role: control ? 1 : 2, takeover: false, ttlSeconds: 300, requestId: requestId() }) as JsonRecord;
      const grant = channel.grant;
      openedGrant = grant as JsonRecord | undefined;
      channelGrantRef.current = grant as JsonRecord;
      const routes = Array.isArray(channel.routes)
        ? channel.routes.filter((route): route is JsonRecord => typeof route === "object" && route !== null)
        : [];
      let peer: DesktopPeer;
      peer = createDesktopPeer(setMediaStream, undefined, routes.map((route) => ({
        url: typeof route.url === "string" ? route.url : undefined,
        username: typeof route.username === "string" ? route.username : undefined,
        credential: typeof route.credential === "string" ? route.credential : undefined,
      })), (connectionState) => {
        // A replaced/closed peer can deliver a late state event after reconnect.
        // Only the currently owned peer may update the pane's session state.
        if (peerRef.current !== peer) return;
        const current = sessionRef.current;
        if (!current || connectionState === "connected" || connectionState === "connecting") return;
        const sessionState = connectionState === "failed" ? "failed" : "disconnected";
        setSessionState({ ...current, connectionState: sessionState });
        handoffRef.dispatch({ type: "owner-failed", owner: "device-control", message: connectionState === "failed" ? "Desktop transport failed." : "Desktop transport disconnected." });
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
      frameControllerRef.current?.abort();
      frameControllerRef.current = undefined;
      setFrame(undefined);
      const peer = peerRef.current;
      peerRef.current = undefined;
      peer?.close();
      await cleanupDesktopOpenFailure(bridge, transport, openedGrant, openedSession);
      channelGrantRef.current = undefined;
      setMediaStream(undefined);
      setSessionState(undefined);
      setError(cause instanceof Error ? cause.message : "Desktop connection failed.");
    }
  };

  const open = async (control = true, ready = readiness, preferWebRTC = false) => {
    const pending = handoffRef.dispatch({ type: "request", operation: "open" });
    if (pending.pending !== "open") return;
    await lifecycleRef.current.run("open", async () => {
      await openOwner(control, ready, preferWebRTC);
      const active = sessionRef.current;
      if (active) handoffRef.dispatch({ type: "owner-opened", owner: "device-control", canControl: true, leaseHeld: active.controller });
      else handoffRef.dispatch({ type: "owner-failed", owner: "device-control", message: "Device owner failed to open a session." });
    });
  };

  const sendDesktopInput = async (event: DesktopInputEvent) => {
    const activeSession = sessionRef.current;
    if (!transport || !activeSession || !canControlDesktop(readiness, activeSession)) return;
    const video = event.currentTarget;
    const rect = video.getBoundingClientRect();
    const action = "key" in event
      ? (() => {
        if (event.type === "keydown") heldKeysRef.current.add(event.key);
        if (event.type === "keyup") heldKeysRef.current.delete(event.key);
        return { key: { kind: event.type === "keyup" ? 2 : 1, key: event.key } };
      })()
      : (() => {
        const display = readiness?.displays.find((candidate) => candidate.id === activeSession.selectedDisplayId);
        const media = video instanceof HTMLVideoElement
          ? { width: video.videoWidth, height: video.videoHeight }
          : { width: video.naturalWidth, height: video.naturalHeight };
        const viewport = { width: display?.width ?? media.width, height: display?.height ?? media.height };
        const pointerRelease = event.type === "pointerup" || event.type === "pointercancel";
        const pointerEvent = "pointerId" in event ? event as PointerEvent<DesktopInputTarget> : undefined;
        const heldPointer = heldPointerRef.current;
        if (pointerRelease && pointerEvent && heldPointer && pointerEvent.pointerId !== heldPointer.id) return undefined;
        const point = mapSurfacePoint(event.clientX, event.clientY, rect, { width: media.width || viewport.width, height: media.height || viewport.height }, viewport, pointerRelease);
        if (!point) return undefined;
        const button = pointerRelease ? heldPointer?.button ?? event.button : event.button;
        if (event.type === "pointerdown" && pointerEvent) heldPointerRef.current = { id: pointerEvent.pointerId, button, point };
        else if (event.type === "pointermove" && pointerEvent && heldPointer?.id === pointerEvent.pointerId) heldPointerRef.current = { ...heldPointer, point };
        if (event.type === "pointerup" || event.type === "pointercancel") heldPointerRef.current = undefined;
        return "deltaY" in event
          ? { wheel: { displayId: activeSession.selectedDisplayId, ...point, horizontalTicks: Math.sign(event.deltaX) * Math.min(20, Math.max(1, Math.round(Math.abs(event.deltaX) / 100))), verticalTicks: Math.sign(event.deltaY) * Math.min(20, Math.max(1, Math.round(Math.abs(event.deltaY) / 100))) } }
          : { pointer: { kind: event.type === "pointerdown" ? 2 : pointerRelease ? 3 : event.type === "click" ? 4 : 1, displayId: activeSession.selectedDisplayId, ...point, button: event.type === "pointermove" ? 0 : button === 2 ? 2 : button === 1 ? 3 : 1 } };
      })();
    if (!action) return;
    const lease = { id: activeSession.leaseId, epoch: activeSession.leaseEpoch, controller: activeSession.controller };
    await dispatchInput(activeSession, lease, action);
  };

  const dispatchInput = async (activeSession: DesktopSession, lease: { id: string; epoch: bigint; controller: boolean }, action: unknown) => {
    if (!transport) return;
    const request = { session: activeSession.ref, leaseId: activeSession.leaseId, leaseEpoch: activeSession.leaseEpoch, commandId: requestId(), displayId: activeSession.selectedDisplayId, geometryRevision: (activeSession as DesktopSession & { geometryRevision?: string }).geometryRevision ?? "", action };
    try {
      const typedAction = action as { key?: { kind?: number }; pointer?: { kind?: number } };
      const release = typedAction.key?.kind === 2 || typedAction.pointer?.kind === 3;
      const accepted = await inputQueueRef.current.enqueue(lease, async () => {
        const latest = sessionRef.current;
        if (!latest || latest.leaseId !== lease.id || latest.leaseEpoch !== lease.epoch || !canControlDesktop(readiness, latest)) return;
        const dataResult = await sendDataRequest("input", request as JsonRecord);
        if (dataResult.sent) {
          if (!dataResult.response) setError("Desktop input outcome is unknown; do not retry automatically.");
          return;
        }
        await transport.input(request);
      }, release);
      if (!accepted) setError(release
        ? "Input release could not be queued; close or reconnect the remote session to clear held input."
        : "Desktop input was discarded because its bounded queue is full or the controller lease changed.");
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Desktop input was rejected."); }
  };

  const typeDesktopText = async (text: string): Promise<boolean> => {
    const active = sessionRef.current;
    if (!transport || !active || !canControlDesktop(readiness, active)) return false;
    if (!/^[ -~]{1,128}$/.test(text)) {
      setError("Desktop typing supports printable ASCII characters.");
      return false;
    }
    const lease = { id: active.leaseId, epoch: active.leaseEpoch, controller: active.controller };
    try {
      const queued = await inputQueueRef.current.enqueue(lease, async () => {
        for (const key of text) {
          const current = sessionRef.current;
          if (!current || current.leaseId !== lease.id || current.leaseEpoch !== lease.epoch || !canControlDesktop(readiness, current)) {
            throw new Error("Desktop control changed while typing.");
          }
          const receipt = await transport.input({
            session: current.ref, leaseId: lease.id, leaseEpoch: lease.epoch, commandId: requestId(),
            displayId: current.selectedDisplayId, geometryRevision: current.geometryRevision,
            action: { key: { kind: 3, key } },
          }) as { outcome?: unknown };
          if (receipt.outcome !== "INPUT_OUTCOME_ACCEPTED" && receipt.outcome !== 1) throw new Error("A key was refused by the desktop.");
        }
      });
      if (!queued) throw new Error("Desktop input is busy.");
      return true;
    } catch {
      setError("Desktop typing stopped. Check the focused field before retrying.");
      return false;
    }
  };

  const releaseHeldInput = async () => {
    const activeSession = sessionRef.current;
    if (!activeSession || !canControlDesktop(readiness, activeSession)) return;
    const lease = { id: activeSession.leaseId, epoch: activeSession.leaseEpoch, controller: activeSession.controller };
    const releases: Promise<void>[] = [];
    for (const key of heldKeysRef.current) releases.push(dispatchInput(activeSession, lease, { key: { kind: 2, key } }));
    heldKeysRef.current.clear();
    const held = heldPointerRef.current;
    if (held) {
      heldPointerRef.current = undefined;
      releases.push(dispatchInput(activeSession, lease, {
        pointer: { kind: 3, displayId: activeSession.selectedDisplayId, ...held.point, button: held.button === 2 ? 2 : held.button === 1 ? 3 : 1 },
      }));
    }
    await Promise.all(releases);
  };

  const takeControl = async () => {
    if (!transport || !session) return;
    const pending = handoffRef.dispatch({ type: "request", operation: "takeover" });
    if (pending.pending !== "takeover") return;
    await lifecycleRef.current.run("takeover", async () => {
      try {
        const granted = await transport.takeControl({ session: session.ref, takeover: true });
        setSessionState(granted);
        handoffRef.dispatch({ type: "owner-control-granted", owner: "device-control", leaseHeld: granted.controller });
      } catch (cause) {
        handoffRef.dispatch({ type: "owner-operation-failed", owner: "device-control", operation: "takeover", message: cause instanceof Error ? cause.message : "Control lease was not granted." });
        setError(cause instanceof Error ? cause.message : "Control lease was not granted.");
      }
    });
  };

  const releaseControl = async () => {
    const activeSession = sessionRef.current;
    if (!transport || !activeSession || !canControlDesktop(readiness, activeSession)) return;
    const pending = handoffRef.dispatch({ type: "request", operation: "handback" });
    if (pending.pending !== "handback") return;
    await lifecycleRef.current.run("handback", async () => {
      try {
        // Device Control drains accepted actions, releases native held input,
        // then advances the lease and returns this session to observer mode.
        const released = await transport.releaseControl({
          session: activeSession.ref,
          leaseId: activeSession.leaseId,
          leaseEpoch: activeSession.leaseEpoch,
        });
        setSessionState(released);
        handoffRef.dispatch({ type: "owner-handback-complete", owner: "device-control", leaseHeld: released.controller });
      } catch (cause) {
        handoffRef.dispatch({ type: "owner-operation-failed", owner: "device-control", operation: "handback", message: cause instanceof Error ? cause.message : "Control could not be released safely." });
        setError(cause instanceof Error ? cause.message : "Control could not be released safely.");
      }
    });
  };

  const sendClipboard = async () => {
    if (!transport || !session || !("clipboard" in navigator) || typeof navigator.clipboard.readText !== "function") return;
    try {
      const text = await navigator.clipboard.readText();
      boundedClipboardText(text);
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
      if (typeof receipt?.text === "string") await navigator.clipboard.writeText(boundedClipboardText(receipt.text));
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Clipboard read was rejected."); }
  };

  const reconnect = async () => {
    if (!transport || !bridgeTransport || !nodeId) return;
    const pending = handoffRef.dispatch({ type: "request", operation: "reconnect" });
    if (pending.pending !== "reconnect") return;
    await lifecycleRef.current.run("reconnect", async () => {
      const restoreController = sessionRef.current?.controller ?? false;
      try {
        await closeOwner("reconnect");
        const nextReadiness = await transport.readiness({ surface: SURFACE(nodeId) });
        setReadiness(nextReadiness);
        if (nextReadiness.state !== 6) {
          setError(nextReadiness.recovery || "Desktop is not ready to reconnect.");
          return;
        }
        await openOwner(restoreController, nextReadiness);
        const active = sessionRef.current;
        if (active) handoffRef.dispatch({ type: "owner-reconnected", owner: "device-control", access: active.controller ? "control" : "observe", leaseHeld: active.controller });
        else handoffRef.dispatch({ type: "owner-failed", owner: "device-control", message: "Desktop reconnect did not establish a session." });
      } catch (cause) {
        denyLocalLeaseInput();
        handoffRef.dispatch({ type: "owner-failed", owner: "device-control", message: cause instanceof Error ? cause.message : "Desktop reconnect failed." });
        setError(cause instanceof Error ? cause.message : "Desktop reconnect failed.");
      }
    });
  };

  if (!nodeId) return <DesktopPane readiness={{ state: 1, reasonCode: "This entry has no linked desktop", recovery: "Go back to Devices and open Desktop on the linked machine." } as DesktopReadiness} />;
  return <div className="flex flex-col gap-2">
    {error && <p role="alert" className="rounded border border-rose-400/30 bg-rose-400/10 p-2 text-xs text-rose-100">{error}</p>}
    <DesktopPane readiness={readiness} session={session} mediaStream={mediaStream} frame={frame} onTypeText={readiness?.httpsFramesAllowed ? typeDesktopText : undefined} onOpenWebRTC={readiness?.httpsFramesAllowed ? () => { void open(true, readiness, true); } : undefined} onOpen={() => { void open(true); }} onOpenViewer={() => { void open(false); }} onRefresh={() => { void refreshReadiness(); }} onReconnect={() => { void reconnect(); }} onClose={() => { void close("user_closed"); }} onTakeControl={() => { void takeControl(); }} onReleaseControl={() => { void releaseControl(); }} onRevoke={() => { void close("user_revoked"); }} onReleaseInput={() => { void releaseHeldInput(); }} onInput={(event) => { void sendDesktopInput(event); }} onClipboardToDesktop={() => { void sendClipboard(); }} onClipboardFromDesktop={() => { void readClipboard(); }} onSurfaceFocusChange={(focused) => { handoffRef.dispatch({ type: "focus", focused }); }} />
  </div>;
}
