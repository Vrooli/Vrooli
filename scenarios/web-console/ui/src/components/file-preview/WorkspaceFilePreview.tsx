import { useCallback, useEffect, useRef, useState } from "react";
import MessagesFileViewer from "../MessagesFileViewer";
import { useFilePreviewController } from "./useFilePreviewController";
import type { PreviewSourceContext } from "../../api/filePreview";
import { ARTIFACT_VIEWER_MIN_WIDTH } from "./artifactViewerLayout";

export interface WorkspaceFilePreviewRequest {
  sessionId: string;
  path: string;
  source: PreviewSourceContext;
}

interface WorkspaceFilePreviewProps {
  request: WorkspaceFilePreviewRequest | null;
  mobile: boolean;
  width: number;
  focusMode: boolean;
  onWidthChange: (width: number) => void;
  onFocus: () => void;
  onRestore: () => void;
  onClose: () => void;
  onHandoff?: (sessionId: string, path: string) => void;
}

const MAX_WIDTH = 900;

export default function WorkspaceFilePreview({
  request,
  mobile,
  width,
  focusMode,
  onWidthChange,
  onFocus,
  onRestore,
  onClose,
  onHandoff,
}: WorkspaceFilePreviewProps) {
  const sessionId = request?.sessionId ?? "";
  const controller = useFilePreviewController(sessionId);
  const resizeStart = useRef<{ x: number; width: number } | null>(null);

  useEffect(() => {
    if (!request) return;
    void controller.openPreview(request.path, request.source);
  }, [controller.openPreview, request]);

  const handleResizeStart = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
    if (focusMode) return;
    event.preventDefault();
    resizeStart.current = { x: event.clientX, width };
    const move = (moveEvent: PointerEvent) => {
      const start = resizeStart.current;
      if (!start) return;
      onWidthChange(Math.max(ARTIFACT_VIEWER_MIN_WIDTH, Math.min(MAX_WIDTH, start.width + start.x - moveEvent.clientX)));
    };
    const end = () => {
      resizeStart.current = null;
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", end);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", end, { once: true });
  }, [focusMode, onWidthChange, width]);

  if (!request) return null;

  const viewerProps = {
    state: controller.state,
    onHandoff: onHandoff ? (path: string) => { onHandoff(request.sessionId, path); } : undefined,
    onClose,
    onReopen: controller.reopen,
    onRendererError: controller.reportError,
    onNavigate: controller.navigateTo,
    onNavigateBack: controller.navigateBack,
    onLoadMore: controller.loadMore,
    onListOptionsChange: controller.setListOptions,
  };

  if (mobile) return <MessagesFileViewer {...viewerProps} />;

  return (
    <div className={focusMode ? "absolute inset-0 z-wc-chrome-raised" : "h-full shrink-0"} style={focusMode ? undefined : { width }}>
      <MessagesFileViewer
        {...viewerProps}
        presentation="dock"
        focusMode={focusMode}
        onFocus={onFocus}
        onRestore={onRestore}
        onResizeStart={handleResizeStart}
      />
    </div>
  );
}
