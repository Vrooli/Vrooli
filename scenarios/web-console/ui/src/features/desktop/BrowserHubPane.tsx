import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { BrowserPane, type BrowserPaneProps, type BrowserPaneSession } from "./BrowserPane";
import { encodeBASHubInput, type BASHubInputSocket } from "./browserHubInput";
import { SingleFlightSurfaceOperation } from "./remoteSurface";
import { WorkspaceSurfaceHandoff } from "./workspaceSurfaceContract";
import { LatestFrameRenderQueue } from "./frameRenderQueue";
import { BrowserHubTelemetry } from "./browserHubTelemetry";

export type HubMessage = {
  type?: string;
  session_id?: string;
  data?: string;
  media_type?: string;
  width?: number;
  height?: number;
  captured_at?: string;
  timestamp?: string;
  input_id?: string;
};

// Keep decoded canvas backing stores at or below 64 MiB RGBA per frame.
export const MAX_BROWSER_FRAME_PIXELS = 16_777_216;

type QueuedBrowserFrame = {
  data: Uint8Array;
  mediaType: string;
  width: number;
  height: number;
  capturedAt?: string;
  streamGeneration: number;
};

export function validateDecodedBrowserFrame(
  decodedWidth: number,
  decodedHeight: number,
  declaredWidth = 0,
  declaredHeight = 0,
): boolean {
  if (
    !Number.isInteger(decodedWidth)
    || !Number.isInteger(decodedHeight)
    || decodedWidth <= 0
    || decodedHeight <= 0
  ) return false;
  if (decodedWidth * decodedHeight > MAX_BROWSER_FRAME_PIXELS) return false;
  return (declaredWidth <= 0 || declaredWidth === decodedWidth)
    && (declaredHeight <= 0 || declaredHeight === decodedHeight);
}

export function decodeBrowserHubSessionFrame(message: HubMessage, sessionID: string): { image: Uint8Array; mediaType: string; width: number; height: number; capturedAt?: string } | undefined {
  const { data, media_type: mediaType, width, height, captured_at: capturedAt } = message;
  if (message.type !== "session_frame" || message.session_id !== sessionID || !data || !mediaType || !["image/png", "image/jpeg", "image/webp"].includes(mediaType) || typeof width !== "number" || typeof height !== "number" || !validateDecodedBrowserFrame(width, height, width, height) || data.length > 8 * 1024 * 1024 || (capturedAt !== undefined && (typeof capturedAt !== "string" || capturedAt.length > 64)) || (message.timestamp !== undefined && (typeof message.timestamp !== "string" || message.timestamp.length > 64)) || Object.keys(message).some((key) => !["type", "session_id", "data", "media_type", "width", "height", "captured_at", "timestamp"].includes(key))) return undefined;
  try {
    const raw = atob(data);
    const image = new Uint8Array(raw.length);
    for (let index = 0; index < raw.length; index += 1) image[index] = raw.charCodeAt(index);
    return { image, mediaType, width, height, capturedAt };
  } catch {
    return undefined;
  }
}

export function decodeBrowserHubBinaryFrame(buffer: ArrayBuffer, sessionID: string): { jpeg: Uint8Array; capturedAt?: string } | undefined {
  if (buffer.byteLength < 6 || buffer.byteLength > 8 * 1024 * 1024) return undefined;
  const bytes = new Uint8Array(buffer);
  const headerLength = new DataView(buffer).getUint32(0, false);
  if (headerLength <= 0 || headerLength > 16 * 1024 || 4 + headerLength + 2 > bytes.length) return undefined;
  let header: { version?: number; session_id?: string; captured_at?: string };
  try { header = JSON.parse(new TextDecoder().decode(bytes.subarray(4, 4 + headerLength))) as typeof header; }
  catch { return undefined; }
  if (header.version !== 1 || header.session_id !== sessionID || (header.captured_at !== undefined && typeof header.captured_at !== "string") || Object.keys(header).some((key) => !["version", "session_id", "captured_at"].includes(key))) return undefined;
  const jpeg = bytes.subarray(4 + headerLength);
  if (jpeg[0] !== 0xff || jpeg[1] !== 0xd8) return undefined;
  // The render queue copies once when it prepares a frame. Keep this bounded
  // view into the WebSocket message so decode does not allocate another JPEG
  // payload buffer before the queue's owned copy.
  return { jpeg, capturedAt: header.captured_at };
}

export interface BrowserHubPaneProps extends Omit<BrowserPaneProps, "frame" | "onInput" | "loading" | "error"> {
  onControlReturned?: () => Promise<void>;
  /** Rechecks the exact Web Console/run/thread owner chain after stream loss. */
  onRevalidateOwner?: (reason: "stream-lost" | "owner-refused") => Promise<boolean>;
}

let nextLocalControlGeneration = 0;
let nextBrowserInputID = 0;

/**
 * BrowserPane wired to the Web Console same-origin BAS stream facade. BAS
 * validates the role receipt and remains the only session/input owner.
 */
export function BrowserHubPane({ session, onControlReturned, onRevalidateOwner, ...paneProps }: BrowserHubPaneProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const socketRef = useRef<WebSocket>();
  const telemetry = useMemo(() => new BrowserHubTelemetry(), [session?.sessionId]);
  const [stats, setStats] = useState<NonNullable<BrowserPaneSession["stats"]>>(() => telemetry.snapshot());
  const [frameSize, setFrameSize] = useState<BrowserPaneSession["frameSize"]>();
  const [frameAgeMs, setFrameAgeMs] = useState<number>();
  const [streamError, setStreamError] = useState("");
  const [streamRevoked, setStreamRevoked] = useState(false);
  const streamRevokedRef = useRef(false);
  const frameContextRef = useRef<{
    streamGeneration: number;
    setError: (message: string) => void;
    render: (frame: QueuedBrowserFrame, bitmap: ImageBitmap) => void;
  }>({ streamGeneration: 0, setError: () => undefined, render: () => undefined });
  frameContextRef.current.setError = setStreamError;
  const frameQueueRef = useRef<LatestFrameRenderQueue<QueuedBrowserFrame, ImageBitmap>>();
  if (!frameQueueRef.current) {
    frameQueueRef.current = new LatestFrameRenderQueue(
      async (frame) => {
        const copy = new Uint8Array(frame.data.byteLength);
        copy.set(frame.data);
        return createImageBitmap(new Blob([copy.buffer as ArrayBuffer], { type: frame.mediaType }));
      },
      (frame, bitmap) => {
        if (frame.streamGeneration !== frameContextRef.current.streamGeneration) {
          bitmap.close();
          return;
        }
        if (!validateDecodedBrowserFrame(bitmap.width, bitmap.height, frame.width, frame.height)) {
          bitmap.close();
          frameContextRef.current.setError("The browser owner sent a frame with invalid dimensions.");
          return;
        }
        frameContextRef.current.render(frame, bitmap);
      },
      (bitmap) => bitmap.close(),
      (_error, frame) => {
        if (frame.streamGeneration === frameContextRef.current.streamGeneration) {
          frameContextRef.current.setError("The browser owner sent a frame that could not be decoded.");
        }
      },
    );
  }
  const [connectionState, setConnectionState] = useState<BrowserPaneSession["state"]>("connecting");
  const [reconnectAllowed, setReconnectAllowed] = useState(false);
  const [reconnectCount, setReconnectCount] = useState(0);
  const returnStartedRef = useRef(false);
  const revalidatePendingRef = useRef(false);
  const returnControlRef = useRef(onControlReturned);
  returnControlRef.current = onControlReturned;
  const closeAfterReturnRef = useRef(false);
  // BAS owns the session. This guard only coalesces pane requests that would
  // otherwise trigger duplicate role-stream transitions from rapid clicks.
  const lifecycleRef = useRef(new SingleFlightSurfaceOperation());
  const handoff = useMemo(() => {
    const owner = new WorkspaceSurfaceHandoff("bas", session?.sessionId ?? "unbound", Boolean(session?.capabilities.control));
    if (session) owner.dispatch({ type: "owner-opened", owner: "bas", canControl: Boolean(session.capabilities.control), leaseHeld: Boolean(session.controller) });
    return owner;
  }, [session?.sessionId]);
  const [handoffState, setHandoffState] = useState(handoff.state);
  const currentHandoffState = handoffState.id === handoff.state.id ? handoffState : handoff.state;
  const handoffStateRef = useRef(currentHandoffState);
  handoffStateRef.current = currentHandoffState;
  const dispatchHandoff = useCallback((event: Parameters<WorkspaceSurfaceHandoff["dispatch"]>[0]) => {
    const next = handoff.dispatch(event);
    setHandoffState({ ...next });
    return next;
  }, [handoff]);
  const handbackPending = currentHandoffState.pending === "handback" || (currentHandoffState.pending === "close" && closeAfterReturnRef.current);
  const controlRoleRequested = Boolean(session?.capabilities.control && !handbackPending && (currentHandoffState.pending === "takeover" || (currentHandoffState.access === "control" && currentHandoffState.pending !== "close")));
  const controllerActive = currentHandoffState.access === "control" && !handbackPending && currentHandoffState.pending !== "close";
  // This is only an in-memory queue fence. The BAS stream cookie and owner
  // receipt remain the authority for attaching and sending browser input.
  const localControlLease = useMemo(() => {
    const generation = ++nextLocalControlGeneration;
    return { receipt: `local-control-${generation}`, epoch: BigInt(generation) };
  }, [session?.sessionId, reconnectCount, controlRoleRequested]);
  const currentControlRef = useRef({
    sessionId: session?.sessionId,
    receipt: localControlLease.receipt,
    epoch: localControlLease.epoch,
    active: controllerActive,
  });
  currentControlRef.current = {
    sessionId: session?.sessionId,
    receipt: localControlLease.receipt,
    epoch: localControlLease.epoch,
    active: controllerActive,
  };
  const role = controlRoleRequested ? "control" : "observe";
  const canObserve = Boolean(session && session.association.source === "agent-run" && (session.state === "viewer" || session.state === "controller") && session.capabilities.observe && session.association.verified);
  const streamURL = useMemo(() => {
    if (typeof window === "undefined") return "";
    const scheme = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${scheme}//${window.location.host}/api/v1/browser/stream`;
  }, []);
  const reportStreamFailure = useCallback((message: string) => {
    if (role === "control") dispatchHandoff({ type: "request", operation: "handback" });
    dispatchHandoff({ type: "owner-failed", owner: "bas", message });
  }, [dispatchHandoff, role]);
  const revalidateOwner = useCallback(async (reason: "stream-lost" | "owner-refused") => {
    if (revalidatePendingRef.current) return;
    revalidatePendingRef.current = true;
    setReconnectAllowed(false);
    if (!onRevalidateOwner) { revalidatePendingRef.current = false; return; }
    try {
      const active = await onRevalidateOwner(reason);
      if (active && reason === "stream-lost") setReconnectAllowed(true);
    } catch {
      // An owner lookup failure is not evidence that a run or session remains active.
      setReconnectAllowed(false);
    } finally {
      revalidatePendingRef.current = false;
    }
  }, [onRevalidateOwner]);

  frameContextRef.current.render = ({ width, height, capturedAt }, bitmap) => {
    try {
      const target = canvasRef.current;
      if (!target || target.width !== bitmap.width || target.height !== bitmap.height) {
        if (target) { target.width = bitmap.width; target.height = bitmap.height; }
      }
      const context = target?.getContext("2d");
      if (context) {
        context.drawImage(bitmap, 0, 0, bitmap.width, bitmap.height);
        telemetry.recordFramePainted(performance.now(), capturedAt, Date.now());
      }
      setFrameSize({ width: bitmap.width || width, height: bitmap.height || height });
      setFrameAgeMs(telemetry.snapshot().frameAgeMs);
      setStats(telemetry.snapshot());
      if (handoffStateRef.current.pending === "takeover") dispatchHandoff({ type: "owner-control-granted", owner: "bas", leaseHeld: true });
      else if (handoffStateRef.current.pending === "reconnect") dispatchHandoff({ type: "owner-reconnected", owner: "bas", access: controlRoleRequested ? "control" : "observe", leaseHeld: controlRoleRequested });
      else if (handoffStateRef.current.pending === "open") dispatchHandoff({ type: "owner-opened", owner: "bas", canControl: Boolean(session?.capabilities.control), leaseHeld: controlRoleRequested });
      if (handbackPending && !controlRoleRequested && !returnStartedRef.current && returnControlRef.current) {
        returnStartedRef.current = true;
        void returnControlRef.current().then(() => {
          if (handoffStateRef.current.pending === "handback" || handoffStateRef.current.pending === "close" || handoffStateRef.current.lifecycle === "failed") dispatchHandoff({ type: "owner-handback-complete", owner: "bas", leaseHeld: false });
          if (closeAfterReturnRef.current) {
            closeAfterReturnRef.current = false;
            paneProps.onClose?.();
            dispatchHandoff({ type: "owner-closed", owner: "bas" });
          }
        }).catch(() => {
          dispatchHandoff({ type: "owner-operation-failed", owner: "bas", operation: "handback", message: "Run handback was not confirmed." });
          returnStartedRef.current = false;
          setStreamError("The browser is observing, but Agent Manager has not confirmed return to the run.");
        });
      }
    } finally {
      bitmap.close();
    }
  };

  const paintFrame = (data: Uint8Array, mediaType: string, width: number, height: number, capturedAt?: string) => {
    frameQueueRef.current?.enqueue({
      data,
      mediaType,
      width,
      height,
      capturedAt,
      streamGeneration: frameContextRef.current.streamGeneration,
    });
  };

  useEffect(() => {
    frameContextRef.current.streamGeneration += 1;
    frameQueueRef.current?.clear();
    const current = canvasRef.current;
    const sessionID = session?.sessionId;
    dispatchHandoff({ type: "request", operation: reconnectCount > 0 ? "reconnect" : role === "control" ? "takeover" : "open" });
    if (current) {
      current.width = 0;
      current.height = 0;
    }
    setStats(telemetry.snapshot());
    setFrameSize(undefined);
    setFrameAgeMs(undefined);
    setConnectionState("connecting");
    streamRevokedRef.current = false;
    setStreamRevoked(false);
    returnStartedRef.current = false;
    if (!canObserve || !sessionID || !streamURL || !current) return;
    setStreamError("");
    telemetry.beginAttach(performance.now());
    const socket = new WebSocket(streamURL);
    socket.binaryType = "arraybuffer";
    socketRef.current = socket;
    socket.addEventListener("open", () => {
      setConnectionState("viewer");
      const attach = JSON.stringify({ type: "attach", role, pane_handle: sessionID });
      const subscribe = JSON.stringify({ type: "subscribe_recording", session_id: sessionID });
      socket.send(attach);
      socket.send(subscribe);
      telemetry.addBytes(new TextEncoder().encode(attach).byteLength + new TextEncoder().encode(subscribe).byteLength);
      setStats(telemetry.snapshot());
    });
    socket.addEventListener("message", (event) => {
      if (event.data instanceof ArrayBuffer) {
        telemetry.addBytes(event.data.byteLength);
        const frame = decodeBrowserHubBinaryFrame(event.data, sessionID);
        if (!frame) { setStreamError("The browser owner sent a frame outside this session."); return; }
        paintFrame(frame.jpeg, "image/jpeg", 0, 0, frame.capturedAt);
        return;
      }
      if (typeof event.data !== "string") return;
      let message: HubMessage;
      try { message = JSON.parse(event.data) as HubMessage; }
      catch { setStreamError("The browser owner sent an unreadable stream message."); return; }
      telemetry.addBytes(new TextEncoder().encode(event.data).byteLength);
      if (message.type === "browser_stream_error") {
        void revalidateOwner("owner-refused");
        reportStreamFailure("BAS refused or lost the authorized browser stream.");
        setStreamError("The browser owner refused or lost the authorized stream.");
        setConnectionState("failed");
        return;
      }
      if (message.type === "session_access_revoked" && message.session_id === sessionID
        && Object.keys(message).every((key) => key === "type" || key === "session_id")) {
        streamRevokedRef.current = true;
        setStreamRevoked(true);
        setStreamError("");
        setConnectionState("revoked");
        return;
      }
      if (message.type === "recording_input_applied" && message.session_id === sessionID) {
        telemetry.recordInputApplied(typeof message.input_id === "string" ? message.input_id : undefined, performance.now());
        setStats(telemetry.snapshot());
        return;
      }
      const frame = decodeBrowserHubSessionFrame(message, sessionID);
      if (!frame) return;
      paintFrame(frame.image, frame.mediaType, frame.width, frame.height, frame.capturedAt);
    });
    socket.addEventListener("close", () => {
      if (socketRef.current === socket && !streamRevokedRef.current) { void revalidateOwner("stream-lost"); telemetry.markDisconnected(performance.now()); setStats(telemetry.snapshot()); reportStreamFailure("Browser stream disconnected."); setStreamError("The browser stream disconnected. Checking whether the run still owns an active browser session."); setConnectionState("disconnected"); }
    });
    socket.addEventListener("error", () => { if (streamRevokedRef.current) return; void revalidateOwner("stream-lost"); telemetry.markDisconnected(performance.now()); setStats(telemetry.snapshot()); reportStreamFailure("Browser stream connection failed."); setStreamError("The same-origin browser stream could not connect. Checking whether the run still owns an active browser session."); setConnectionState("disconnected"); });
    return () => {
      frameContextRef.current.streamGeneration += 1;
      frameQueueRef.current?.clear();
      socketRef.current = undefined;
      if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) socket.close();
    };
  }, [canObserve, reconnectCount, role, session?.sessionId, streamURL, controlRoleRequested, handoff, dispatchHandoff, reportStreamFailure, revalidateOwner, telemetry]);

  const onInput: BrowserPaneProps["onInput"] = async (input, _order, lease) => {
    const current = currentControlRef.current;
    const socket = socketRef.current as (WebSocket & BASHubInputSocket) | undefined;
    const inputID = `wc-input-${++nextBrowserInputID}`;
    const encoded = current.sessionId ? encodeBASHubInput(current.sessionId, input, inputID) : undefined;
    if (!current.active || !current.sessionId || lease.receipt !== current.receipt || lease.epoch !== current.epoch || socket?.readyState !== WebSocket.OPEN || !encoded) {
      throw new Error("The owner-admitted browser stream is not ready for input.");
    }
    socket.send(encoded);
    telemetry.addBytes(new TextEncoder().encode(encoded).byteLength);
    telemetry.recordInputSent(inputID, performance.now());
    setStats(telemetry.snapshot());
  };

  const ownerTerminalState = session && ["offline", "blocked", "revoked", "failed"].includes(session.state) ? session.state : undefined;
  const liveState: BrowserPaneSession["state"] = ownerTerminalState ?? (streamRevoked ? "revoked" : connectionState === "viewer" && controllerActive ? "controller" : connectionState);
  const liveSession = session ? { ...session, state: liveState, controller: controllerActive, controlReceipt: controllerActive ? localControlLease.receipt : undefined, sessionEpoch: controllerActive ? localControlLease.epoch : undefined, frameSize: frameSize ?? session.frameSize, stats, frameAgeMs } : undefined;
  const reconnect = () => { void lifecycleRef.current.run("reconnect", async () => { telemetry.startReconnect(performance.now()); setStats(telemetry.snapshot()); dispatchHandoff({ type: "request", operation: "reconnect" }); setStreamError(""); setConnectionState("connecting"); setReconnectCount((count)=>count+1); }); };
  const takeControl = () => { void lifecycleRef.current.run("takeover", async () => { const pending = dispatchHandoff({ type: "request", operation: "takeover" }); if (pending.pending !== "takeover") return; setStreamError(""); setConnectionState("connecting"); }); };
  const beginHandback = () => {
    const pending = dispatchHandoff({ type: "request", operation: "handback" });
    if (pending.pending !== "handback" && pending.pending !== "close") return false;
    setStreamError(""); returnStartedRef.current = false; setConnectionState("connecting");
    return true;
  };
  const releaseControl = () => { void lifecycleRef.current.run("handback", async () => { beginHandback(); }); };
  const closePane = () => {
    if (controlRoleRequested) { closeAfterReturnRef.current = true; beginHandback(); dispatchHandoff({ type: "request", operation: "close" }); return; }
    dispatchHandoff({ type: "request", operation: "close" });
    paneProps.onClose?.();
    dispatchHandoff({ type: "owner-closed", owner: "bas" });
  };
  return <BrowserPane {...paneProps} onClose={closePane} session={liveSession} frame={<canvas key={`${session?.sessionId ?? "none"}:${session?.sessionEpoch ?? 0n}:${reconnectCount}:${canObserve ? role : "closed"}`} ref={canvasRef} />} onInput={onInput} onTakeControl={takeControl} onHandBack={releaseControl} onReconnect={reconnectAllowed ? reconnect : undefined} onSurfaceFocusChange={(focused) => { dispatchHandoff({ type: "focus", focused }); }} loading={Boolean(session && canObserve && !frameSize)} error={streamError || undefined} />;
}
