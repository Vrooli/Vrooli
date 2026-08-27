import { useEffect, useRef, useState } from "react";
import { AlertTriangle, Check, Code, Copy, Eye, Loader2, Maximize, Minus, Plus, RotateCcw } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../consts/strings";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { useCodeCopy } from "./markdown/hooks/useCodeCopy";
import { useMermaidSvg } from "./markdown/hooks/useMermaidSvg";
import { MermaidZoomSurface, type MermaidZoomSurfaceHandle } from "./mermaid-viewer/MermaidZoomSurface";
import { formatScalePercent } from "./mermaid-viewer/zoomTransform";

interface MessagesMermaidViewerProps {
  open: boolean;
  /** Mermaid source for the active diagram. */
  code: string;
  onClose: () => void;
}

const toolbarButton =
  "inline-flex h-8 w-8 items-center justify-center rounded-lg border border-wc-default bg-wc-surface-input text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary";

/**
 * MessagesMermaidViewer is the full-screen, zoomable Mermaid diagram drawer. It
 * reuses the shared DrawerShell and the Mermaid render hook (so initialization
 * stays a singleton), then layers zoom/pan, source toggle, and copy on top.
 * Diagrams are message-local UI content, so this never touches the file-preview
 * controller or its model.
 */
export default function MessagesMermaidViewer({ open, code, onClose }: MessagesMermaidViewerProps) {
  const { t } = useTranslation();
  const { svgHtml, error, loading } = useMermaidSvg(open ? code : "");
  const [showSource, setShowSource] = useState(false);
  const [scale, setScale] = useState(1);
  const { copied, copyCode } = useCodeCopy(code);
  const surfaceRef = useRef<MermaidZoomSurfaceHandle | null>(null);

  // Each newly opened diagram starts on the diagram view.
  useEffect(() => {
    if (open) setShowSource(false);
  }, [open, code]);

  const showDiagram = !showSource && !error;

  const headerActions = (
    <div className="flex items-center gap-1.5">
      {showDiagram && (
        <>
          <button
            type="button"
            onClick={() => surfaceRef.current?.zoomOut()}
            className={toolbarButton}
            aria-label={t(strings.mermaid.zoomOut)}
            title={t(strings.mermaid.zoomOut)}
          >
            <Minus className="h-4 w-4" />
          </button>
          <span className="w-12 text-center font-mono text-xs text-wc-text-muted" data-testid="mermaid-zoom-level">
            {formatScalePercent(scale)}
          </span>
          <button
            type="button"
            onClick={() => surfaceRef.current?.zoomIn()}
            className={toolbarButton}
            aria-label={t(strings.mermaid.zoomIn)}
            title={t(strings.mermaid.zoomIn)}
          >
            <Plus className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={() => surfaceRef.current?.fit()}
            className={toolbarButton}
            aria-label={t(strings.mermaid.fitToScreen)}
            title={t(strings.mermaid.fitToScreen)}
          >
            <Maximize className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={() => surfaceRef.current?.reset()}
            className={toolbarButton}
            aria-label={t(strings.mermaid.resetZoom)}
            title={t(strings.mermaid.resetZoom)}
          >
            <RotateCcw className="h-4 w-4" />
          </button>
          <span className="mx-1 h-5 w-px bg-wc-default" aria-hidden="true" />
        </>
      )}
      {!error && (
        <button
          type="button"
          onClick={() => setShowSource((prev) => !prev)}
          className={toolbarButton}
          aria-label={showSource ? t(strings.mermaid.showDiagram) : t(strings.mermaid.showSource)}
          title={showSource ? t(strings.mermaid.showDiagram) : t(strings.mermaid.showSource)}
        >
          {showSource ? <Eye className="h-4 w-4" /> : <Code className="h-4 w-4" />}
        </button>
      )}
      <button
        type="button"
        onClick={copyCode}
        className={toolbarButton}
        aria-label={copied ? t(strings.mermaid.copied) : t(strings.mermaid.copySource)}
        title={copied ? t(strings.mermaid.copied) : t(strings.mermaid.copySource)}
      >
        {copied ? <Check className="h-4 w-4 text-green-400" /> : <Copy className="h-4 w-4" />}
      </button>
    </div>
  );

  const badges = (
    <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-wc-text-faint">
      <span className="rounded-full border border-wc-default px-2 py-0.5">{t(strings.mermaid.badgeMessageDiagram)}</span>
      <span className="rounded-full border border-wc-default px-2 py-0.5">mermaid</span>
    </div>
  );

  return (
    <FullPageDrawer
      open={open}
      onClose={onClose}
      closeLabel={t(strings.mermaid.closeViewer)}
      title={t(strings.mermaid.viewerTitle)}
      headerActions={headerActions}
      headerExtra={badges}
      testId="messages-mermaid-viewer-panel"
    >
      {showSource ? (
        <pre
          data-testid="mermaid-viewer-source"
          className="h-full overflow-auto whitespace-pre p-4 font-mono text-sm text-wc-text-primary"
        >
          {code}
        </pre>
      ) : error ? (
        <div className="h-full overflow-auto px-4 py-4">
          <div className="mx-auto max-w-2xl rounded-2xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-300">
            <div className="mb-2 flex items-center gap-2 font-medium">
              <AlertTriangle className="h-4 w-4" />
              <span>{t(strings.mermaid.renderError)}</span>
            </div>
            <p className="break-words">{error}</p>
            <pre className="mt-3 overflow-auto whitespace-pre rounded-lg bg-wc-surface-base p-3 font-mono text-xs text-wc-text-primary">
              {code}
            </pre>
          </div>
        </div>
      ) : svgHtml ? (
        <MermaidZoomSurface
          ref={surfaceRef}
          svgHtml={svgHtml}
          onScaleChange={setScale}
          ariaLabel={t(strings.mermaid.viewerTitle)}
        />
      ) : (
        <div className="flex h-full items-center justify-center gap-2 text-wc-text-muted">
          <Loader2 className="h-5 w-5 animate-spin" />
          {loading && <span>{t(strings.mermaid.rendering)}</span>}
        </div>
      )}
    </FullPageDrawer>
  );
}
