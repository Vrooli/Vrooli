import { ARTIFACT_VIEWER_MIN_WIDTH, FileViewer, useFilePreviewController, type PreviewSourceContext, type ViewerActionContext } from "@vrooli/file-viewer";
import { useCallback, useEffect, useMemo, useRef } from "react";

import { sessionPreviewSource } from "../../api/filePreview";
import { ViewerActions } from "./ViewerActions";

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
  onRun?: (sessionId: string, path: string, text: string) => void;
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
  onRun,
}: WorkspaceFilePreviewProps) {
  const sessionId = request?.sessionId ?? "";
  const source = useMemo(() => sessionPreviewSource(sessionId), [sessionId]);
  const controller = useFilePreviewController(source);
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
    source,
    controller,
    onClose,
    renderHeaderActions: (context: ViewerActionContext) => (
      <ViewerActions
        context={context}
        onHandoff={onHandoff ? (path: string) => { onHandoff(request.sessionId, path); } : undefined}
        onRun={onRun ? (path: string, text: string) => { onRun(request.sessionId, path, text); } : undefined}
      />
    ),
  };

  if (mobile) return <FileViewer {...viewerProps} />;

  return (
    <div className={focusMode ? "absolute inset-0 z-wc-chrome-raised" : "h-full shrink-0"} style={focusMode ? undefined : { width }}>
      <FileViewer
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
