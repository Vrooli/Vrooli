import { cloneElement, useEffect, useRef, useState, type CanvasHTMLAttributes, type KeyboardEvent as ReactKeyboardEvent, type PointerEvent as ReactPointerEvent, type ReactElement, type WheelEvent as ReactWheelEvent } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { projectRemoteSurfaceState, isWorkspaceNavigationKey, workspaceSurfaceTouchAction, WorkspacePaneToolbar, WorkspaceSurfacePane } from "./WorkspaceSurface";
import { boundedClipboardText, boundedRemoteTextInput, mapSurfacePoint, OrderedLeaseInput, REMOTE_SURFACE_CLIPBOARD_LIMIT, REMOTE_SURFACE_TEXT_INPUT_LIMIT, type BrowserSurfaceInput, type RemoteSurfaceState, type SurfaceSize } from "./remoteSurface";

export interface BrowserPaneSession {
  sessionId: string;
  permissionProfile?: string;
  mode: "managed" | "installed" | "user-attached" | "remote";
  state: RemoteSurfaceState;
  controller: boolean;
  /** In-memory input-queue lease tag. Actual stream authority stays in the HttpOnly owner cookie. */
  controlReceipt?: string;
  sessionEpoch?: bigint;
  association: { source: "agent-run" | "conversation"; runId?: string; threadId?: string; verified: boolean };
  capabilities: { observe: boolean; control?: boolean; tabs?: boolean; frames?: boolean; downloads?: boolean; console?: boolean; clipboard?: boolean };
  page?: { id: string; title?: string; url?: string };
  pages?: Array<{ id: string; title?: string; url?: string; active: boolean }>;
  viewport?: SurfaceSize;
  frameSize?: SurfaceSize;
  frameAgeMs?: number;
  stats?: { frames: number; bytes: number; inputApplied?: number; attachToFirstFrameMs?: number };
}

export interface BrowserPaneProps {
  session?: BrowserPaneSession;
  /** Required before the owner is asked to open a session; no ad hoc session link. */
  launchAssociation?: { source: "agent-run" | "conversation"; runId?: string; threadId?: string; verified: boolean };
  /** Owner-decoded pixels render to canvas; never accept a live page/iframe as a browser surface. */
  frame?: ReactElement<CanvasHTMLAttributes<HTMLCanvasElement>, "canvas">;
  loading?: boolean;
  error?: string;
  onInput?: (input: BrowserSurfaceInput, order: number, lease: { receipt: string; epoch: bigint }) => Promise<void>;
  onPasteToBrowser?: (text: string, order: number, lease: { receipt: string; epoch: bigint }) => Promise<void>;
  onCopyFromBrowser?: (lease: { receipt: string; epoch: bigint }) => Promise<string>;
  onOpen?: () => void;
  /** Open an already-authorized session through BAS; human tab consent stays with that owner. */
  onAttach?: () => void;
  onTakeControl?: () => void;
  onHandBack?: () => void;
  onReconnect?: () => void;
  onClose?: () => void;
  closeLabel?: string;
  onSelectPage?: (pageId: string) => void;
  onShowDownloads?: () => void;
  onShowConsole?: () => void;
  onSurfaceFocusChange?: (focused: boolean) => void;
}

/** Web Console presentation adapter. Session creation, frames and input stay with BAS. */
export function BrowserPane({ session, launchAssociation, frame, loading = false, error, onInput, onPasteToBrowser, onCopyFromBrowser, onOpen, onAttach, onTakeControl, onHandBack, onReconnect, onClose, closeLabel = "Close session", onSelectPage, onShowDownloads, onShowConsole, onSurfaceFocusChange }: BrowserPaneProps) {
  const surfaceRef = useRef<HTMLDivElement>(null);
  const inputQueueRef = useRef(new OrderedLeaseInput());
  const heldPointerRef = useRef<{ id: number; point: { x: number; y: number }; button: "left" | "middle" | "right" }>();
  const pressedKeysRef = useRef(new Set<string>());
  const sessionRef = useRef<BrowserPaneSession>();
  const lastMoveAtRef = useRef(0);
  const [inputError, setInputError] = useState("");
  const [clipboardStatus, setClipboardStatus] = useState("");
  const [textComposerOpen, setTextComposerOpen] = useState(false);
  const [typedText, setTypedText] = useState("");
  sessionRef.current = session;
  const context = session?.association ?? launchAssociation;
  const associationValid = Boolean(context?.verified && (context.source === "conversation" ? context.threadId : true));
  const authorizationValid = associationValid && Boolean(session?.capabilities.observe);
  const state: RemoteSurfaceState = session && !authorizationValid || !session && !associationValid ? "blocked" : session?.state ?? (loading ? "connecting" : "ready");
  const paneState = projectRemoteSurfaceState(state, authorizationValid || !session && associationValid);
  const controller = authorizationValid && session?.capabilities.control === true && session.controller && Boolean(session.controlReceipt) && (session.sessionEpoch ?? 0n) > 0n && state === "controller";
  const canSendInput = controller && Boolean(onInput && session?.viewport && session.frameSize && session.capabilities.frames);
  const associationLabel = context?.source === "agent-run" ? `Verified agent run ${context.runId || "unavailable"}` : `Conversation ${context?.threadId || "unverified"}`;
  const summary = session
    ? `Browser session · ${session.mode} · ${associationLabel}${context?.threadId ? ` · Thread ${context.threadId}` : ""}${session.page?.url ? ` · ${session.page.url}` : ""}`
    : associationValid ? `${associationLabel} · Open through the owning browser session broker.` : "Select a verified agent run or Web Console conversation before opening a browser session.";
  const stats = session?.stats
    ? `Frames ${session.stats.frames} · bytes ${session.stats.bytes}${session.stats.inputApplied === undefined ? "" : ` · inputs ${session.stats.inputApplied}`}${session.stats.attachToFirstFrameMs === undefined ? "" : ` · attach to first frame ${session.stats.attachToFirstFrameMs} ms`}${session.frameAgeMs === undefined ? "" : ` · frame age ${session.frameAgeMs} ms`}`
    : undefined;
  const frameSize = session?.frameSize ?? session?.viewport;
  const frameAspectRatio = frameSize && frameSize.width > 0 && frameSize.height > 0
    ? `${frameSize.width} / ${frameSize.height}`
    : "16 / 9";

  useEffect(() => {
    inputQueueRef.current.updateLease(session?.controlReceipt && session.sessionEpoch
      ? { id: session.controlReceipt, epoch: session.sessionEpoch, controller }
      : undefined);
    if (!controller) {
      pressedKeysRef.current.clear();
      heldPointerRef.current = undefined;
      setTypedText("");
      setTextComposerOpen(false);
    }
  }, [controller, session?.controlReceipt, session?.sessionEpoch]);

  const sendBrowserInput = (input: BrowserSurfaceInput) => {
    const receipt = session?.controlReceipt;
    const epoch = session?.sessionEpoch;
    if (!(input.type === "text" ? controller && onInput : canSendInput) || !receipt || !epoch || !onInput) return;
    const release = input.type === "key" ? input.phase === "up" : input.type === "pointer" && input.phase === "up";
    void inputQueueRef.current.enqueue({ id: receipt, epoch, controller }, (order) => onInput(input, order, { receipt, epoch }), release)
      .then((accepted) => { if (!accepted) setInputError("Browser input was discarded because the control grant changed or the bounded queue is full."); })
      .catch((cause: unknown) => setInputError(cause instanceof Error ? cause.message : "Browser input was rejected."));
  };

  const enterText = () => {
    if (!typedText || !controller) return;
    try {
      const text = boundedRemoteTextInput(typedText);
      sendBrowserInput({ type: "text", text });
      setTypedText("");
    } catch (cause) { setInputError(cause instanceof Error ? cause.message : "Text input is too large."); }
  };

  const updateTypedText = (value: string) => {
    try {
      boundedRemoteTextInput(value);
      setInputError("");
      setTypedText(value);
    } catch (cause) { setInputError(cause instanceof Error ? cause.message : "Text input is too large."); }
  };

  const clipboardLease = () => session?.controlReceipt && session.sessionEpoch && controller
    ? { id: session.controlReceipt, epoch: session.sessionEpoch, controller, receipt: session.controlReceipt }
    : undefined;

  const pasteToBrowser = async () => {
    const lease = clipboardLease();
    if (!lease || !session?.capabilities.clipboard || !onPasteToBrowser || !navigator.clipboard?.readText) return;
    try {
      const text = boundedClipboardText(await navigator.clipboard.readText());
      const accepted = await inputQueueRef.current.enqueue(lease, (order) => onPasteToBrowser(text, order, { receipt: lease.receipt, epoch: lease.epoch }));
      if (!accepted) throw new Error("Clipboard transfer was discarded because the controller grant changed.");
      setClipboardStatus("Text sent to browser.");
    } catch (cause) { setClipboardStatus(cause instanceof Error ? cause.message : "Browser clipboard transfer failed."); }
  };

  const copyFromBrowser = async () => {
    const lease = clipboardLease();
    if (!lease || !session?.capabilities.clipboard || !onCopyFromBrowser || !navigator.clipboard?.writeText) return;
    try {
      const text = boundedClipboardText(await onCopyFromBrowser({ receipt: lease.receipt, epoch: lease.epoch }));
      const current = sessionRef.current;
      if (!current || current.controlReceipt !== lease.receipt || current.sessionEpoch !== lease.epoch || !current.controller) {
        throw new Error("Browser clipboard transfer was discarded because the controller grant changed.");
      }
      await navigator.clipboard.writeText(text);
      setClipboardStatus("Text copied from browser.");
    } catch (cause) { setClipboardStatus(cause instanceof Error ? cause.message : "Browser clipboard transfer failed."); }
  };

  const pointerAction = (phase: "move" | "down" | "up", event: ReactPointerEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!canSendInput || !session?.viewport || !session.frameSize) return;
    const heldPointer = heldPointerRef.current;
    if (phase === "move" && heldPointer?.id !== event.pointerId) return;
    if (phase === "move" && Date.now() - lastMoveAtRef.current < 33) return;
    if (phase === "up" && heldPointer?.id !== event.pointerId) return;
    const rect = surfaceRef.current?.getBoundingClientRect();
    const point = rect && mapSurfacePoint(event.clientX, event.clientY, rect, session.frameSize, session.viewport, phase === "up");
    if (!point) return;
    const button = phase === "up" && heldPointer ? heldPointer.button : event.button === 2 ? "right" : event.button === 1 ? "middle" : "left";
    if (phase === "down") {
      heldPointerRef.current = { id: event.pointerId, point, button };
      if (typeof event.currentTarget.setPointerCapture === "function") event.currentTarget.setPointerCapture(event.pointerId);
    }
    if (phase === "move" && heldPointer) heldPointerRef.current = { ...heldPointer, point };
    if (phase === "up") heldPointerRef.current = undefined;
    if (phase === "move") lastMoveAtRef.current = Date.now();
    const modifiers = [event.shiftKey && "Shift", event.altKey && "Alt", event.ctrlKey && "Control", event.metaKey && "Meta"].filter((value): value is string => Boolean(value));
    sendBrowserInput({ type: "pointer", phase, button, ...point, modifiers });
  };

  const keyboardAction = (phase: "down" | "up", event: ReactKeyboardEvent<HTMLDivElement>) => {
    // Keep an escape path from the remote keyboard target so keyboard users
    // can continue to the pane controls and leave the surface.
    if (isWorkspaceNavigationKey(event)) return;
    event.preventDefault();
    event.stopPropagation();
    if (!canSendInput || phase === "down" && (event.repeat || pressedKeysRef.current.has(event.key))) return;
    const modifiers = [event.shiftKey && "Shift", event.altKey && "Alt", event.ctrlKey && "Control", event.metaKey && "Meta"].filter((value): value is string => Boolean(value));
    if (phase === "down") pressedKeysRef.current.add(event.key);
    else pressedKeysRef.current.delete(event.key);
    sendBrowserInput({ type: "key", phase, key: event.key, modifiers });
  };

  const releaseHeldInput = () => {
    if (canSendInput) {
      for (const key of pressedKeysRef.current) sendBrowserInput({ type: "key", phase: "up", key, modifiers: [] });
      const heldPointer = heldPointerRef.current;
      if (heldPointer) sendBrowserInput({ type: "pointer", phase: "up", button: heldPointer.button, ...heldPointer.point });
    }
    pressedKeysRef.current.clear();
    heldPointerRef.current = undefined;
  };

  const wheelAction = (event: ReactWheelEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!canSendInput || !session?.viewport || !session.frameSize) return;
    const rect = surfaceRef.current?.getBoundingClientRect();
    const point = rect && mapSurfacePoint(event.clientX, event.clientY, rect, session.frameSize, session.viewport);
    if (point) sendBrowserInput({ type: "wheel", ...point, deltaX: event.deltaX, deltaY: event.deltaY });
  };

  return <WorkspaceSurfacePane label="Browser session" title="Browser" owner="bas" canControl={Boolean(session?.capabilities.control)} onFocusChange={onSurfaceFocusChange} state={paneState} summary={summary} stats={stats}>
    {(error || inputError) && <p role="alert" className="rounded border border-rose-400/30 bg-rose-400/10 p-2 text-xs text-rose-100">{error || inputError}</p>}
    {state === "revoked" && <p role="status" className="text-sm">The browser controller lease was revoked. Close this session before reconnecting.</p>}
    {state === "blocked" && <p role="status" className="text-sm">{!associationValid ? "This browser session is not associated with a verified agent run or conversation thread." : "This browser session does not grant observation."} Control is disabled.</p>}
    {state === "ready" && !session && associationValid && <div className="flex flex-1 flex-wrap items-center justify-center gap-2"><Button type="button" onClick={onOpen} disabled={!onOpen}>Open browser session</Button><Button type="button" variant="secondary" onClick={onAttach} disabled={!onAttach}>Attach an approved browser session</Button></div>}
    {session && <>
      {session.capabilities.tabs && session.pages && session.pages.length > 0 && <nav aria-label="Browser tabs" className="flex gap-1 overflow-x-auto">{session.pages.map((page) => <Button key={page.id} type="button" variant={page.active ? "primary" : "secondary"} size="md" shape="square" aria-current={page.active ? "page" : undefined} onClick={() => onSelectPage?.(page.id)} className="min-h-11 max-w-56 truncate">{page.title || page.url || page.id}</Button>)}</nav>}
      <p className="text-xs text-app-muted">Allowed: {[
        session.capabilities.observe && "observe",
        session.capabilities.control && "control",
        session.capabilities.tabs && "tabs",
        session.capabilities.frames && "frames",
        session.capabilities.downloads && "downloads",
        session.capabilities.console && "console",
        session.capabilities.clipboard && "clipboard",
      ].filter(Boolean).join(" · ") || "none"}{session.permissionProfile ? ` · permission profile ${session.permissionProfile}` : ""}</p>
      <div ref={surfaceRef} role="group" aria-label="Live browser surface" aria-disabled={!canSendInput} tabIndex={canSendInput ? 0 : -1} onBlur={releaseHeldInput} onClickCapture={(event) => { event.preventDefault(); event.stopPropagation(); }} onPointerMoveCapture={(event) => pointerAction("move", event)} onPointerDownCapture={(event) => pointerAction("down", event)} onPointerUpCapture={(event) => pointerAction("up", event)} onPointerCancelCapture={(event) => pointerAction("up", event)} onKeyDownCapture={(event) => keyboardAction("down", event)} onKeyUpCapture={(event) => keyboardAction("up", event)} onWheelCapture={wheelAction} style={{ touchAction: workspaceSurfaceTouchAction(canSendInput), aspectRatio: frameAspectRatio, maxHeight: "70vh" }} className={`relative min-h-48 w-full flex-1 overflow-hidden rounded border border-app-border bg-black${canSendInput ? "" : " pointer-events-none"}`}>
      {session.capabilities.frames ? frame ? cloneElement(frame, {
        role: frame.props.role ?? "img",
        "aria-label": frame.props["aria-label"] ?? "Live browser frame",
        className: ["block h-full w-full", frame.props.className].filter(Boolean).join(" "),
        style: { display: "block", width: "100%", height: "100%", objectFit: "contain", ...frame.props.style },
      }) : <div className="flex min-h-48 items-center justify-center p-4 text-center text-sm text-white">{state === "connecting" ? "Waiting for the browser owner’s first frame…" : "The browser owner has not provided a live frame."}</div> : <div className="flex min-h-48 items-center justify-center p-4 text-center text-sm text-white">This permission profile does not allow frame viewing.</div>}
      </div>
      <WorkspacePaneToolbar label="Browser session controls" state={paneState}>
        {controller && onInput && <Button type="button" variant="secondary" size="md" onClick={() => { setTextComposerOpen((open) => !open); setTypedText(""); }}>{textComposerOpen ? "Hide typing controls" : "Type text"}</Button>}
        {session.capabilities.clipboard && <>
          <Button type="button" variant="secondary" size="md" onClick={() => { void pasteToBrowser(); }} disabled={!controller || !onPasteToBrowser || !navigator.clipboard?.readText}>Send text</Button>
          <Button type="button" variant="secondary" size="md" onClick={() => { void copyFromBrowser(); }} disabled={!controller || !onCopyFromBrowser || !navigator.clipboard?.writeText}>Copy text</Button>
        </>}
        {authorizationValid && session.capabilities.control && !controller && session.state === "viewer" && <Button type="button" variant="secondary" size="md" onClick={onTakeControl} disabled={!onTakeControl}>Take control</Button>}
        {controller && <Button type="button" variant="secondary" size="md" onClick={onHandBack} disabled={!onHandBack}>Release control</Button>}
        {session.capabilities.downloads && <Button type="button" variant="secondary" size="md" onClick={onShowDownloads} disabled={!onShowDownloads}>Downloads</Button>}
        {session.capabilities.console && <Button type="button" variant="secondary" size="md" onClick={onShowConsole} disabled={!onShowConsole}>Console</Button>}
        {(session.state === "disconnected" || session.state === "failed") && <Button type="button" variant="secondary" size="md" onClick={onReconnect} disabled={!onReconnect}>Reconnect</Button>}
        <Button type="button" variant="secondary" size="md" onClick={onClose} disabled={!onClose}>{closeLabel}</Button>
      </WorkspacePaneToolbar>
      {textComposerOpen && controller && <form onSubmit={(event) => { event.preventDefault(); enterText(); }} className="flex flex-col gap-2">
        <label htmlFor="browser-text-entry" className="text-xs text-app-muted">Type into the focused browser page</label>
        <textarea id="browser-text-entry" aria-label="Text to enter in browser" value={typedText} onChange={(event) => updateTypedText(event.currentTarget.value)} rows={2} maxLength={REMOTE_SURFACE_TEXT_INPUT_LIMIT} autoFocus className="min-h-11 w-full resize-y rounded border border-app-border bg-app-surface px-3 py-2 text-sm text-app-text" />
        <Button type="submit" variant="secondary" size="md" disabled={!typedText}>Send to browser</Button>
      </form>}
      {clipboardStatus && <p role="status" aria-live="polite" className="text-xs text-app-muted">{clipboardStatus}</p>}
      {!controller ? <p className="text-xs text-app-muted">Observer mode: browser input is unavailable until this verified session holds the controller lease.</p> : !canSendInput && <p className="text-xs text-app-muted">The session has control, but its frame dimensions or ordered input adapter are unavailable.</p>}
    </>}
  </WorkspaceSurfacePane>;
}
