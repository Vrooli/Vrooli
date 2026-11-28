import { ARTIFACT_VIEWER_MIN_WIDTH, FileViewer, ViewerButton, useFilePreviewController, type PreviewSourceContext, type ViewerActionContext } from "@vrooli/file-viewer";
import { PenLine } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type FocusEvent } from "react";

import { sessionPreviewSource } from "../../api/filePreview";
import { FILE_PREVIEW_MAX_WIDTH, useFilePreviewStore, type RecentFilePreview } from "../../stores/useFilePreviewStore";
import { ViewerActions } from "./ViewerActions";
import { projectFilePreviewState, WorkspacePaneStateStatus, WorkspacePaneToolbar } from "../../features/desktop/WorkspaceSurface";
import { WorkspaceSurfaceHandoff, workspaceSurfacePresentationAttributes } from "../../features/desktop/workspaceSurfaceContract";
import { DesignAuthoringTool } from "../../features/design-authoring/DesignAuthoringTool";

export interface WorkspaceFilePreviewRequest {
  sessionId: string;
  path: string;
  source: PreviewSourceContext;
}

interface WorkspaceFilePreviewProps {
  request: WorkspaceFilePreviewRequest | null;
  mobile: boolean;
  width: number;
  maxWidth?: number;
  focusMode: boolean;
  onWidthChange: (width: number) => void;
  onFocus: () => void;
  onRestore: () => void;
  onClose: () => void;
  onHandoff?: (sessionId: string, path: string) => void;
  onRun?: (sessionId: string, path: string, text: string) => void;
  recentFiles?: RecentFilePreview[];
  onOpenRecent?: (sessionId: string, path: string) => void;
}

const MAX_WIDTH = FILE_PREVIEW_MAX_WIDTH;

export default function WorkspaceFilePreview({
  request,
  mobile,
  width,
  maxWidth = MAX_WIDTH,
  focusMode,
  onWidthChange,
  onFocus,
  onRestore,
  onClose,
  onHandoff,
  onRun,
  recentFiles,
  onOpenRecent,
}: WorkspaceFilePreviewProps) {
  const sessionId = request?.sessionId ?? "";
  const source = useMemo(() => sessionPreviewSource(sessionId), [sessionId]);
  const controller = useFilePreviewController(source);
  const handoff = useMemo(() => new WorkspaceSurfaceHandoff("file-preview", sessionId || "closed"), [sessionId]);
  const paneState = projectFilePreviewState(Boolean(request), controller.state.status);
  const resizeStart = useRef<{ x: number; width: number } | null>(null);
  const surfaceRootRef = useRef<HTMLDivElement>(null);
  const authoringActionRef = useRef<HTMLButtonElement>(null);
  const authoringLayerRef = useRef<HTMLDivElement>(null);
  const [authoringOpen, setAuthoringOpen] = useState(false);
  const [authoringInitialSource, setAuthoringInitialSource] = useState<{ path: string; text: string } | undefined>();
  const [authoringRecoveryMessage, setAuthoringRecoveryMessage] = useState("");
  const rememberRecent = useFilePreviewStore((state) => state.rememberRecent);
  const resolvedModel = controller.state.model;
  // A session change leaves the previous controller state on screen for one
  // render. Only a preview that reached ready after the current request began
  // may be recorded, or the old session's file would be filed under the new one.
  const requestKey = `${request?.sessionId ?? ""}\u0000${request?.path ?? ""}`;
  const canRememberRef = useRef(false);

  useEffect(() => {
    if (!request) {
      if (handoff.state.lifecycle !== "closed") {
        handoff.dispatch({ type: "request", operation: "close" });
        handoff.dispatch({ type: "owner-closed", owner: "file-preview" });
      }
      return;
    }
    handoff.dispatch({ type: "request", operation: "open" });
    void controller.openPreview(request.path, request.source);
    return () => {
      handoff.dispatch({ type: "request", operation: "close" });
      handoff.dispatch({ type: "owner-closed", owner: "file-preview" });
    };
  }, [controller.openPreview, handoff, request]);

  useEffect(() => {
    const status = controller.state.status;
    if (["ready", "choosing", "unsupported"].includes(status)) {
      handoff.dispatch({ type: "owner-opened", owner: "file-preview", canControl: false });
    } else if (status === "error") {
      handoff.dispatch({ type: "owner-failed", owner: "file-preview", message: controller.state.error ?? "File preview failed." });
      if (surfaceRootRef.current) surfaceRootRef.current.dataset.surfaceFocused = "false";
    }
  }, [controller.state.error, controller.state.status, handoff]);

  useEffect(() => { handoff.dispatch({ type: "focus", focused: focusMode }); }, [focusMode, handoff]);

  useEffect(() => { canRememberRef.current = false; }, [requestKey]);

  // Remember a resolved file once it is actually on screen. Directories and
  // archive members are skipped: neither reopens from a path the source can
  // resolve later without extra context.
  useEffect(() => {
    if (controller.state.status !== "ready") {
      canRememberRef.current = true;
      return;
    }
    if (!request || !resolvedModel) return;
    if (resolvedModel.kind === "directory" || resolvedModel.kind === "archive" || resolvedModel.archiveMember) {
      canRememberRef.current = false;
      return;
    }
    if (!canRememberRef.current) return;
    canRememberRef.current = false;
    rememberRecent({
      sessionId: request.sessionId,
      path: resolvedModel.resolvedPath,
      basename: resolvedModel.basename,
      kind: resolvedModel.kind,
    });
  }, [controller.state.status, resolvedModel?.resolvedPath, resolvedModel?.basename, resolvedModel?.kind, resolvedModel?.archiveMember, request?.sessionId, rememberRecent]);

  const closePreview = useCallback(() => {
    handoff.dispatch({ type: "request", operation: "close" });
    onClose();
  }, [handoff, onClose]);

  useEffect(() => {
    if (!authoringOpen) return;
    authoringLayerRef.current?.querySelector<HTMLButtonElement>('[role="dialog"] button[aria-label="Close authoring workflow"]')?.focus();
  }, [authoringOpen]);

  const closeAuthoring = useCallback(() => {
    setAuthoringOpen(false);
    authoringActionRef.current?.focus();
  }, []);

  useEffect(() => {
    if (!authoringOpen) return;
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      event.stopPropagation();
      closeAuthoring();
    };
    window.addEventListener("keydown", handleEscape, true);
    return () => window.removeEventListener("keydown", handleEscape, true);
  }, [authoringOpen, closeAuthoring]);

  const handleResizeStart = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
    if (focusMode) return;
    event.preventDefault();
    resizeStart.current = { x: event.clientX, width };
    const move = (moveEvent: PointerEvent) => {
      const start = resizeStart.current;
      if (!start) return;
      const rtl = document.documentElement.dir === "rtl";
      const delta = rtl ? moveEvent.clientX - start.x : start.x - moveEvent.clientX;
      onWidthChange(Math.max(ARTIFACT_VIEWER_MIN_WIDTH, Math.min(maxWidth, start.width + delta)));
    };
    const end = () => {
      resizeStart.current = null;
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", end);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", end, { once: true });
  }, [focusMode, maxWidth, onWidthChange, width]);

  // A new target is a new authoring context; leaving the dialog open would
  // keep the previous file's source under a different preview.
  useEffect(() => {
    setAuthoringOpen(false);
    setAuthoringInitialSource(undefined);
    setAuthoringRecoveryMessage("");
  }, [request?.sessionId, request?.path]);

  if (!request) return null;

  const parentOf = (value: string) => {
    const index = Math.max(value.lastIndexOf("/"), value.lastIndexOf("\\"));
    return index > 0 ? value.slice(0, index) : "";
  };
  const viewerRecentFiles = (recentFiles ?? [])
    .filter((item) => item.sessionId === request.sessionId)
    .slice(0, 8)
    .map((item) => ({
      id: `${item.sessionId}:${item.path}`,
      path: item.path,
      name: item.basename,
      meta: parentOf(item.path),
      active: item.path === resolvedModel?.resolvedPath,
      onSelect: () => { onOpenRecent?.(item.sessionId, item.path); },
    }));

  const viewerProps = {
    source,
    controller,
    recentFiles: viewerRecentFiles,
    onClose: closePreview,
    onFocus: () => { handoff.dispatch({ type: "focus", focused: true }); onFocus(); },
    onRestore: () => { handoff.dispatch({ type: "focus", focused: false }); onRestore(); },
    renderHeaderActions: (context: ViewerActionContext) => (
      <WorkspacePaneToolbar label="File preview controls" state={paneState} nowrap>
        <WorkspacePaneStateStatus state={paneState} label="File preview" />
        <ViewerButton
          ref={authoringActionRef}
          type="button"
          variant="secondary"
          onClick={() => {
            const sourceText = context.text?.content;
            const validPreviewSource = Boolean(sourceText?.trim()) && /\.html?$/i.test(context.path);
            setAuthoringInitialSource(validPreviewSource && sourceText ? { path: context.path, text: sourceText } : undefined);
            setAuthoringRecoveryMessage(validPreviewSource ? "" : "The current preview is not a readable HTML source. In Step 1, choose the retained brief, source map, selected-image manifest, referenced images and editable HTML source to continue.");
            setAuthoringOpen(true);
          }}
          data-testid="open-design-authoring"
          aria-label="Open design packet authoring"
          title={authoringRecoveryMessage ? "Design packet needs source files" : "Create and revise a source-bound design packet"}
        >
          <PenLine className="h-4 w-4" aria-hidden="true" />
          <span className="hidden sm:inline">Design packet</span>
        </ViewerButton>
        <ViewerActions
          context={context}
          onHandoff={onHandoff ? (path: string) => { onHandoff(request.sessionId, path); } : undefined}
          onRun={onRun ? (path: string, text: string) => { onRun(request.sessionId, path, text); } : undefined}
        />
      </WorkspacePaneToolbar>
    ),
  };

  const authoringTool = authoringOpen && <div ref={authoringLayerRef}>
    <DesignAuthoringTool
      sessionId={request.sessionId}
      onClose={closeAuthoring}
      onOpenArtifact={(path) => { closeAuthoring(); void controller.openPreview(path, request.source); }}
      initialSource={authoringInitialSource}
    />
    {authoringRecoveryMessage && <p role="status" className="sr-only">{authoringRecoveryMessage}</p>}
  </div>;

  const surfaceContractAttributes = {
    ...workspaceSurfacePresentationAttributes("file-preview", paneState, handoff.state.focused && controller.state.status !== "error"),
    "data-state": paneState,
  };
  const surfaceFocusHandlers = {
    onFocusCapture: (event: FocusEvent<HTMLDivElement>) => { handoff.dispatch({ type: "focus", focused: true }); event.currentTarget.dataset.surfaceFocused = "true"; },
    onBlurCapture: (event: FocusEvent<HTMLDivElement>) => { if (!event.currentTarget.contains(event.relatedTarget as Node | null)) { handoff.dispatch({ type: "focus", focused: false }); event.currentTarget.dataset.surfaceFocused = "false"; } },
  };

  if (mobile) return <div ref={surfaceRootRef} {...surfaceContractAttributes} {...surfaceFocusHandlers} className="contents"><FileViewer {...viewerProps} />{authoringTool}</div>;

  return (
    <div ref={surfaceRootRef} {...surfaceContractAttributes} {...surfaceFocusHandlers} className="contents">
      <div className={focusMode ? "absolute inset-0 z-wc-chrome-raised" : "h-full shrink-0"} style={focusMode ? undefined : { width }}>
        <FileViewer
          {...viewerProps}
          presentation="dock"
          focusMode={focusMode}
          onFocus={onFocus}
          onRestore={onRestore}
          onResizeStart={handleResizeStart}
          resizeValue={{ min: ARTIFACT_VIEWER_MIN_WIDTH, max: maxWidth, now: width }}
          onResizeByKeyboard={onWidthChange}
        />
      </div>
      {authoringTool}
    </div>
  );
}
