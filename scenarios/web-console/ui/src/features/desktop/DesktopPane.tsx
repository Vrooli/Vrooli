import { useEffect, useRef } from "react";
import type { KeyboardEvent, MouseEvent, PointerEvent, WheelEvent } from "react";
import type { DesktopReadiness, DesktopSession } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { canControlDesktop, desktopOwnerState, desktopPaneState, isWorkspaceNavigationKey, workspaceSurfaceTouchAction, WorkspacePaneToolbar, WorkspaceSurfacePane } from "./WorkspaceSurface";

export interface DesktopPaneProps {
  readiness?: DesktopReadiness;
  session?: DesktopSession;
  mediaStream?: MediaStream;
  onOpen?: () => void;
  onOpenViewer?: () => void;
  onRefresh?: () => void;
  onReconnect?: () => void;
  onClose?: () => void;
  onTakeControl?: () => void;
  onReleaseControl?: () => void;
  onRevoke?: () => void;
  onReleaseInput?: () => void;
  onInput?: (event: MouseEvent<HTMLVideoElement> | PointerEvent<HTMLVideoElement> | KeyboardEvent<HTMLVideoElement> | WheelEvent<HTMLVideoElement>) => void;
  onClipboardToDesktop?: () => void;
  onClipboardFromDesktop?: () => void;
  onSurfaceFocusChange?: (focused: boolean) => void;
}

export function DesktopPane({ readiness, session, mediaStream, onOpen, onOpenViewer, onRefresh, onReconnect, onClose, onTakeControl, onReleaseControl, onRevoke, onReleaseInput, onInput, onClipboardToDesktop, onClipboardFromDesktop, onSurfaceFocusChange }: DesktopPaneProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const state = desktopOwnerState(readiness, session);
  const paneState = desktopPaneState(readiness, session);
  const controller = canControlDesktop(readiness, session);
  const stats = session?.transportStats;
  const inputHandler = controller ? onInput : undefined;
  const pointerDown = (event: PointerEvent<HTMLVideoElement>) => {
    if (typeof event.currentTarget.setPointerCapture === "function") event.currentTarget.setPointerCapture(event.pointerId);
    inputHandler?.(event);
  };
  const keyboardInput = (event: KeyboardEvent<HTMLVideoElement>) => {
    // Tab stays with the Web Console so keyboard users can reach the pane
    // toolbar and leave the remote input target.
    if (isWorkspaceNavigationKey(event)) return;
    event.preventDefault();
    event.stopPropagation();
    inputHandler?.(event);
  };
  useEffect(() => {
    if (videoRef.current) videoRef.current.srcObject = mediaStream ?? null;
  }, [mediaStream]);
  const blocked = state === "blocked" || state === "offline" || state === "failed" || state === "revoked";
  const readyToOpen = state === "ready" && !session;
  return <WorkspaceSurfacePane label="Remote desktop" title="Remote desktop" owner="device-control" canControl={Boolean(readiness && session && canControlDesktop(readiness, session))} onFocusChange={onSurfaceFocusChange} state={paneState} summary={readiness?.reasonCode || (session ? `${session.selectedDisplayId} · ${session.codec}` : "Select a trusted node")} statsLabel="Desktop transport statistics" stats={session ? `Frames ${String(stats?.framesSent ?? 0n)} · dropped ${String(stats?.framesDropped ?? 0n)} · bytes ${String(stats?.bytesSent ?? 0n)} · rejected ${String((stats?.controlMessagesRejected ?? 0n) + (stats?.dataMessagesRejected ?? 0n))}` : undefined}>
    <div data-testid="desktop-pane" className="flex min-h-64 flex-col gap-3">
    {blocked ? <div className="flex flex-1 flex-col items-center justify-center gap-2 text-center"><p className="text-sm">{state === "revoked" ? "The controller lease was revoked. Close this session before reconnecting." : readiness?.recovery || "Desktop control is unavailable."}</p><div className="flex flex-wrap justify-center gap-2">{state !== "revoked" && <Button type="button" variant="secondary" onClick={onRefresh} disabled={!onRefresh}>Check readiness</Button>}{session && <Button type="button" variant="secondary" onClick={onClose ?? onRevoke} disabled={!onClose && !onRevoke}>Close session</Button>}</div></div> : readyToOpen ? <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center"><p className="text-sm">{readiness?.selectedDisplayId ? `Display ${readiness.selectedDisplayId} is ready.` : "A desktop display is ready."}</p><div className="flex flex-wrap justify-center gap-2"><Button type="button" onClick={onOpen} disabled={!onOpen}>Open with control</Button><Button type="button" variant="secondary" onClick={onOpenViewer} disabled={!onOpenViewer}>View only</Button></div></div> : <>
      <div className="relative overflow-hidden rounded border border-app-border bg-black"><video ref={videoRef} autoPlay playsInline muted aria-label="Selected desktop display" onBlur={onReleaseInput} onClick={inputHandler} onPointerMove={inputHandler} onPointerDown={pointerDown} onPointerUp={inputHandler} onPointerCancel={inputHandler} onWheel={inputHandler} onKeyDown={keyboardInput} onKeyUp={keyboardInput} tabIndex={controller ? 0 : -1} style={{ touchAction: workspaceSurfaceTouchAction(controller) }} className="aspect-video w-full object-contain" />{state === "connecting" && <p role="status" className="absolute inset-0 flex items-center justify-center text-sm text-white">Connecting…</p>}</div>
      {state === "disconnected" && <p role="status" className="text-sm text-amber-200">Desktop connection lost. Reconnect to start a new session and lease.</p>}
      <WorkspacePaneToolbar label="Desktop controls" state={paneState}><Button type="button" variant="secondary" size="md" onClick={onReconnect} disabled={!onReconnect}>Reconnect</Button>{state === "viewer" && <Button type="button" variant="secondary" size="md" onClick={onTakeControl} disabled={!onTakeControl}>Take control</Button>}{controller && <Button type="button" variant="secondary" size="md" onClick={onReleaseControl} disabled={!onReleaseControl}>Release control</Button>}<Button type="button" variant="secondary" size="md" onClick={onClipboardToDesktop} disabled={!controller}>Send text</Button><Button type="button" variant="secondary" size="md" onClick={onClipboardFromDesktop} disabled={!controller}>Read text</Button><Button type="button" variant="secondary" size="md" onClick={onRevoke} disabled={!onRevoke}>Revoke</Button></WorkspacePaneToolbar>
      {!controller && <p className="text-xs text-app-muted">Viewer mode: input and clipboard actions are disabled until this session holds the controller lease.</p>}
    </>}
    </div>
  </WorkspaceSurfacePane>;
}
