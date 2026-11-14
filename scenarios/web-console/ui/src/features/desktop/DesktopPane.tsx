import { useEffect, useRef } from "react";
import type { KeyboardEvent, MouseEvent, WheelEvent } from "react";
import type { DesktopReadiness, DesktopSession } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { canMutateDesktop, desktopViewState } from "./desktopState";

export interface DesktopPaneProps {
  readiness?: DesktopReadiness;
  session?: DesktopSession;
  mediaStream?: MediaStream;
  onOpen?: () => void;
  onOpenViewer?: () => void;
  onRefresh?: () => void;
  onReconnect?: () => void;
  onTakeControl?: () => void;
  onRevoke?: () => void;
  onInput?: (event: MouseEvent<HTMLVideoElement> | KeyboardEvent<HTMLVideoElement> | WheelEvent<HTMLVideoElement>) => void;
  onClipboardToDesktop?: () => void;
  onClipboardFromDesktop?: () => void;
}

export function DesktopPane({ readiness, session, mediaStream, onOpen, onOpenViewer, onRefresh, onReconnect, onTakeControl, onRevoke, onInput, onClipboardToDesktop, onClipboardFromDesktop }: DesktopPaneProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const state = desktopViewState(readiness, session);
  const controller = canMutateDesktop(readiness, session);
  const stats = session?.transportStats;
  const inputHandler = controller ? onInput : undefined;
  useEffect(() => {
    if (videoRef.current) videoRef.current.srcObject = mediaStream ?? null;
  }, [mediaStream]);
  const blocked = state === "blocked" || state === "offline" || state === "failed" || state === "revoked";
  const readyToOpen = state === "ready" && !session;
  return <section aria-label="Remote desktop" data-testid="desktop-pane" data-state={state} className="flex min-h-64 flex-col gap-3 rounded-lg border border-app-border bg-app-surface p-3">
    <header className="flex items-center justify-between gap-3">
      <div><h2 className="text-sm font-semibold">Remote desktop</h2><p role="status" className="text-xs text-app-muted">{readiness?.reasonCode || (session ? `${session.selectedDisplayId} · ${session.codec}` : "Select a trusted node")}</p>{session && <p aria-label="Desktop transport statistics" className="text-xs text-app-muted">Frames {String(stats?.framesSent ?? 0n)} · dropped {String(stats?.framesDropped ?? 0n)} · bytes {String(stats?.bytesSent ?? 0n)} · rejected {String((stats?.controlMessagesRejected ?? 0n) + (stats?.dataMessagesRejected ?? 0n))}</p>}</div>
      <span aria-label={`Desktop state: ${state}`} className="rounded-full border border-app-border px-2 py-1 text-xs">{state}</span>
    </header>
    {blocked ? <div className="flex flex-1 flex-col items-center justify-center gap-2 text-center"><p className="text-sm">{readiness?.recovery || "Desktop control is unavailable."}</p><button type="button" onClick={onRefresh} disabled={!onRefresh || state === "revoked"} className="rounded border border-app-border px-3 py-2 text-sm">Check readiness</button></div> : readyToOpen ? <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center"><p className="text-sm">{readiness?.selectedDisplayId ? `Display ${readiness.selectedDisplayId} is ready.` : "A desktop display is ready."}</p><div className="flex flex-wrap justify-center gap-2"><button type="button" onClick={onOpen} disabled={!onOpen} className="rounded border border-app-border px-3 py-2 text-sm">Open with control</button><button type="button" onClick={onOpenViewer} disabled={!onOpenViewer} className="rounded border border-app-border px-3 py-2 text-sm">View only</button></div></div> : <>
      <div className="relative overflow-hidden rounded border border-app-border bg-black"><video ref={videoRef} autoPlay playsInline muted aria-label="Selected desktop display" onClick={inputHandler} onPointerMove={inputHandler} onPointerDown={inputHandler} onPointerUp={inputHandler} onWheel={inputHandler} onKeyDown={inputHandler} onKeyUp={inputHandler} tabIndex={controller ? 0 : -1} className="aspect-video w-full object-contain" />{state === "connecting" && <p role="status" className="absolute inset-0 flex items-center justify-center text-sm text-white">Connecting…</p>}</div>
      {state === "disconnected" && <p role="status" className="text-sm text-amber-200">Desktop connection lost. Reconnect to start a new session and lease.</p>}
      <div className="flex flex-wrap gap-2" role="toolbar" aria-label="Desktop controls"><button type="button" onClick={onReconnect} disabled={!onReconnect} className="rounded border border-app-border px-2 py-1 text-xs">Reconnect</button>{state === "viewer" && <button type="button" onClick={onTakeControl} disabled={!onTakeControl} className="rounded border border-app-border px-2 py-1 text-xs">Take control</button>}<button type="button" onClick={onClipboardToDesktop} disabled={!controller} className="rounded border border-app-border px-2 py-1 text-xs">Send text</button><button type="button" onClick={onClipboardFromDesktop} disabled={!controller} className="rounded border border-app-border px-2 py-1 text-xs">Read text</button><button type="button" onClick={onRevoke} disabled={!onRevoke} className="rounded border border-app-border px-2 py-1 text-xs">Revoke</button></div>
      {!controller && <p className="text-xs text-app-muted">Viewer mode: input and clipboard actions are disabled until this session holds the controller lease.</p>}
    </>}
  </section>;
}
