import { useEffect, useRef, useState } from "react";
import type { KeyboardEvent, MouseEvent, PointerEvent, WheelEvent } from "react";
import type { DesktopReadiness, DesktopSession } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { canControlDesktop, desktopOwnerState, desktopPaneState, isWorkspaceNavigationKey, workspaceSurfaceTouchAction, WorkspacePaneToolbar, WorkspaceSurfacePane } from "./WorkspaceSurface";

export interface DesktopPaneProps {
  readiness?: DesktopReadiness;
  session?: DesktopSession;
  mediaStream?: MediaStream;
  frame?: string;
  onOpenWebRTC?: () => void;
  onTypeText?: (text: string) => Promise<boolean>;
  onOpen?: () => void;
  onOpenViewer?: () => void;
  onRefresh?: () => void;
  onReconnect?: () => void;
  onClose?: () => void;
  onTakeControl?: () => void;
  onReleaseControl?: () => void;
  onRevoke?: () => void;
  onReleaseInput?: () => void;
  onInput?: (event: MouseEvent<HTMLVideoElement | HTMLImageElement> | PointerEvent<HTMLVideoElement | HTMLImageElement> | KeyboardEvent<HTMLVideoElement | HTMLImageElement> | WheelEvent<HTMLVideoElement | HTMLImageElement>) => void;
  onClipboardToDesktop?: () => void;
  onClipboardFromDesktop?: () => void;
  onSurfaceFocusChange?: (focused: boolean) => void;
}

export function DesktopPane({ readiness, session, mediaStream, frame, onOpenWebRTC, onTypeText, onOpen, onOpenViewer, onRefresh, onReconnect, onClose, onTakeControl, onReleaseControl, onRevoke, onReleaseInput, onInput, onClipboardToDesktop, onClipboardFromDesktop, onSurfaceFocusChange }: DesktopPaneProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [typing, setTyping] = useState(false);
  const textRef = useRef<HTMLInputElement>(null);
  const state = desktopOwnerState(readiness, session);
  const paneState = desktopPaneState(readiness, session);
  const controller = canControlDesktop(readiness, session);
  const stats = session?.transportStats;
  const inputHandler = controller ? onInput : undefined;
  const pointerDown = (event: PointerEvent<HTMLVideoElement | HTMLImageElement>) => {
    if (typeof event.currentTarget.setPointerCapture === "function") event.currentTarget.setPointerCapture(event.pointerId);
    inputHandler?.(event);
  };
  const keyboardInput = (event: KeyboardEvent<HTMLVideoElement | HTMLImageElement>) => {
    // Tab stays with the Web Console so keyboard users can reach the pane
    // toolbar and leave the remote input target.
    if (isWorkspaceNavigationKey(event)) return;
    event.preventDefault();
    event.stopPropagation();
    inputHandler?.(event);
  };
  useEffect(() => {
    if (videoRef.current) videoRef.current.srcObject = mediaStream ?? null;
  }, [frame, mediaStream]);
  const surfaceInput = {
    onBlur: onReleaseInput,
    onPointerMove: inputHandler,
    onPointerDown: pointerDown,
    onPointerUp: inputHandler,
    onPointerCancel: inputHandler,
    onWheel: inputHandler,
    onKeyDown: keyboardInput,
    onKeyUp: keyboardInput,
    tabIndex: controller ? 0 : -1,
    style: { touchAction: workspaceSurfaceTouchAction(controller) },
    className: "aspect-video w-full object-contain",
  };
  const blocked = state === "blocked" || state === "offline" || state === "failed" || state === "revoked";
  const readyToOpen = state === "ready" && !session;
  return <WorkspaceSurfacePane compact label="Remote desktop" title="Remote desktop" owner="device-control" canControl={Boolean(readiness && session && canControlDesktop(readiness, session))} onFocusChange={onSurfaceFocusChange} state={paneState} summary={!session && blocked ? readiness?.reasonCode : undefined} statsLabel="Desktop transport statistics" stats={session ? (stats ? `Frames ${String(stats.framesSent)} · dropped ${String(stats.framesDropped)} · bytes ${String(stats.bytesSent)} · rejected ${String(stats.controlMessagesRejected + stats.dataMessagesRejected)}` : session.codec) : undefined}>
    <div data-testid="desktop-pane" className="flex flex-col gap-2">
    {blocked ? <div className="flex flex-1 flex-col items-center justify-center gap-2 text-center"><p className="text-sm">{state === "revoked" ? "The controller lease was revoked. Close this session before reconnecting." : readiness?.recovery || "Desktop control is unavailable."}</p><div className="flex flex-wrap justify-center gap-2">{state !== "revoked" && <Button type="button" variant="secondary" onClick={onRefresh} disabled={!onRefresh}>Check readiness</Button>}{session && <Button type="button" variant="secondary" onClick={onClose ?? onRevoke} disabled={!onClose && !onRevoke}>Close session</Button>}</div></div> : readyToOpen ? <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center"><p className="text-sm">{readiness?.selectedDisplayId ? `Display ${readiness.selectedDisplayId} is ready.` : "A desktop display is ready."}</p><div className="flex flex-wrap justify-center gap-2"><Button type="button" onClick={onOpen} disabled={!onOpen}>Open with control</Button><Button type="button" variant="secondary" onClick={onOpenViewer} disabled={!onOpenViewer}>View only</Button>{onOpenWebRTC && <Button type="button" variant="secondary" onClick={onOpenWebRTC}>Open with WebRTC</Button>}</div></div> : <>
      <div className="relative overflow-hidden rounded-lg bg-black ring-1 ring-inset ring-app-border">{frame ? <img src={frame} alt="Selected desktop display" draggable={false} {...surfaceInput} />
        : <video ref={videoRef} autoPlay playsInline muted aria-label="Selected desktop display" {...surfaceInput} />}{state === "connecting" && <p role="status" className="absolute inset-0 flex items-center justify-center text-sm text-white">Connecting…</p>}</div>
      {state === "disconnected" && <p role="status" className="text-sm text-amber-200">Desktop connection lost. Reconnect to start a new session and lease.</p>}
      {controller && onTypeText && <form className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2" onSubmit={(event) => {
        event.preventDefault();
        const input = textRef.current;
        if (!input || !input.value || typing) return;
        const text = input.value;
        input.value = "";
        setTyping(true);
        void onTypeText(text).finally(() => setTyping(false));
      }}>
        <label className="min-w-0">
          <span className="sr-only">Type into the focused desktop field</span>
          <input ref={textRef} type="password" autoComplete="off" autoCapitalize="none" spellCheck={false} maxLength={128} disabled={typing} placeholder="Type into the desktop" className="min-h-11 w-full rounded-lg border border-app-border bg-app-surface px-3 text-base" />
        </label>
        <Button type="submit" size="sm" disabled={typing}>{typing ? "Typing…" : "Type on desktop"}</Button>
      </form>}
      <WorkspacePaneToolbar label="Desktop controls" state={paneState}><Button type="button" variant="secondary" size="sm" onClick={onReconnect} disabled={!onReconnect}>Reconnect</Button>{state === "viewer" && <Button type="button" variant="secondary" size="sm" onClick={onTakeControl} disabled={!onTakeControl}>Take control</Button>}{controller && <Button type="button" variant="secondary" size="sm" onClick={onReleaseControl} disabled={!onReleaseControl}>Release control</Button>}{readiness?.clipboardAllowed !== false && <><Button type="button" variant="secondary" size="sm" onClick={onClipboardToDesktop} disabled={!controller}>Send text</Button><Button type="button" variant="secondary" size="sm" onClick={onClipboardFromDesktop} disabled={!controller}>Read text</Button></>}<Button type="button" variant="secondary" size="sm" onClick={onRevoke} disabled={!onRevoke}>Revoke</Button></WorkspacePaneToolbar>
    </>}
    </div>
  </WorkspaceSurfacePane>;
}
