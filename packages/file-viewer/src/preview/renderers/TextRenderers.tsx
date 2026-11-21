import { useEffect, useRef, useState } from "react";
import { Code2, Eye, Minus, Plus } from "lucide-react";
import { useFileViewerText } from "../../i18n/context.js";
import { IconButton } from "@vrooli/react-component-library/IconButton";

import { StructuredPreview } from "./StructuredPreview.js";
import { isStructuredPath } from "../structuredData.js";
import { strings } from "../../i18n/strings.js";
import { cn } from "../../lib/classnames.js";

import { PreviewActions, PreviewMetaLine, PreviewNotice } from "./shared.js";
import { useHtmlPreview } from "../useHtmlPreview.js";
import { attachHtmlPreviewInspection, htmlPreviewPinchScript } from "../htmlPreviewPinch.js";
import { attachHtmlPreviewLinks, htmlPreviewLinksScript } from "../htmlPreviewLinks.js";
import { CodeLinePreview } from "./CodeLinePreview.js";
import { ArtifactFrame, ArtifactFooter } from "./ArtifactChrome.js";
import { formatBytes } from "../format.js";
import type { PreviewRendererProps } from "../types.js";

// CodePreview renders code/text/diff fallback content with line numbers,
// syntax highlighting, font sizing, wrap toggle, and target-line scroll.
export function CodePreview(props: PreviewRendererProps) {
  const { model } = props;
  // Keep API downloads inert; only this isolated renderer interprets markup.
  if (!model.archiveMember && (/\.html?$/i.test(model.resolvedPath) || model.mimeType.split(";")[0]?.trim().toLowerCase() === "text/html")) {
    return <HtmlPreview key={model.previewId} {...props} />;
  }
  if (props.text === null) return <UnavailableText model={model} />;
  if (isStructuredPath(model.resolvedPath)) return <StructuredPreview {...props} />;
  return <ArtifactFrame testId="file-preview-source"><SourcePreview key={model.previewId} {...props} /><ArtifactFooter model={model} metadata={formatBytes(model.sizeBytes)} /></ArtifactFrame>;
}

function SourcePreview({ model, text }: PreviewRendererProps) {
  if (text === null) return <UnavailableText model={model} />;
  const content = text.content;
  if (content === "") {
    return <EmptyText />;
  }
  return (
    <CodeLinePreview content={content} path={model.resolvedPath} highlightLine={model.line ?? null} truncated={text.truncated} plainText={model.kind === "text"} />
  );
}

function escapeHtmlAttribute(value: string): string {
  return value.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// Fetching the HTML through srcDoc avoids a second document navigation through
// an Access/tunnel boundary. The base still points at the session-bound runtime
// route, so relative CSS, JS, images, and child pages resolve normally.
function htmlPreviewDocument(content: string, runtimeHref?: string, anchor?: string): string {
  let baseHref = "about:srcdoc";
  let runtimeOrigin = "";
  if (runtimeHref) {
    try {
      const base = new URL(runtimeHref, window.location.href);
      baseHref = base.href;
      runtimeOrigin = base.origin;
    } catch {
      // Keep the inert srcdoc fallback if a malformed runtime URL is supplied.
    }
  }

  const resourceOrigin = runtimeOrigin || "about:srcdoc";
  const framePolicy = runtimeOrigin ? runtimeOrigin : "'none'";
  const documentPolicy = `default-src 'none'; script-src 'unsafe-inline' ${runtimeOrigin}; style-src 'unsafe-inline' ${runtimeOrigin}; img-src ${resourceOrigin} data: blob:; font-src ${resourceOrigin} data:; media-src ${resourceOrigin} data: blob:; base-uri ${resourceOrigin}; connect-src 'none'; form-action 'none'; frame-src ${framePolicy}; navigate-to ${framePolicy}; object-src 'none'`;
  // Set the opaque frame's own URL before authored scripts read location.hash.
  // The absolute srcdoc URL avoids resolving the fragment against the runtime base.
  const fragmentScript = anchor
    ? `<script data-fragment="${escapeHtmlAttribute(anchor)}">location.replace("about:srcdoc#" + document.currentScript.dataset.fragment)</script>`
    : "";
  return `<!doctype html><meta http-equiv="Content-Security-Policy" content="${documentPolicy}"><base href="${escapeHtmlAttribute(baseHref)}"><meta name="referrer" content="no-referrer">${fragmentScript}${htmlPreviewPinchScript()}${htmlPreviewLinksScript()}${content}`;
}

const HTML_VIEWPORTS = [
  { value: "responsive", width: 0, label: "htmlViewportResponsive" },
  { value: "phone", width: 390, label: "htmlViewportPhone" },
  { value: "tablet", width: 768, label: "htmlViewportTablet" },
  { value: "desktop", width: 1440, label: "htmlViewportDesktop" },
] as const;
const HTML_ZOOM_STEPS = [0.1, 0.25, 0.5, 0.75, 1, 1.25, 1.5, 2];

function HtmlPreview(props: PreviewRendererProps) {
  const { model, text, onNavigate, source: previewSource } = props;
  const { t } = useFileViewerText();
  const { content, error } = useHtmlPreview(model, text);
  const [source, setSource] = useState(!!model.line || !!text?.truncated);
  const [viewport, setViewport] = useState("responsive");
  const [zoom, setZoom] = useState<number | "fit">("fit");
  const stageRef = useRef<HTMLDivElement>(null);
  const inspectionRef = useRef<HTMLDivElement>(null);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const [stageSize, setStageSize] = useState({ width: 0, height: 0 });

  useEffect(() => {
    const frame = frameRef.current;
    if (!frame) return;
    return attachHtmlPreviewLinks(frame, model.resolvedPath, onNavigate);
  }, [content, source, model.resolvedPath, onNavigate]);

  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) return;
    const observer = new ResizeObserver(([entry]) => {
      if (!entry) return;
      const { width, height } = entry.contentRect;
      setStageSize((previous) => previous.width === width && previous.height === height ? previous : { width, height });
    });
    observer.observe(stage);
    return () => { observer.disconnect(); };
  }, [source, content]);

  const viewportWidth = HTML_VIEWPORTS.find((preset) => preset.value === viewport)?.width || stageSize.width;
  const fitScale = viewportWidth > 0 && stageSize.width > 0 ? Math.min(1, stageSize.width / viewportWidth) : 1;
  const scale = zoom === "fit" ? fitScale : zoom;
  useEffect(() => {
    const frame = frameRef.current;
    const surface = inspectionRef.current;
    const stage = stageRef.current;
    if (!frame || !surface || !stage || !viewportWidth || !stageSize.height) return;
    return attachHtmlPreviewInspection(frame, surface, stage, scale, viewportWidth * scale, stageSize.height);
  }, [source, scale, viewportWidth, stageSize.height]);

  const smallerZoom = [...HTML_ZOOM_STEPS].reverse().find((step) => step < scale - 0.001);
  const largerZoom = HTML_ZOOM_STEPS.find((step) => step > scale + 0.001);
  const toggleLabel = t(source ? strings.viewer.htmlRendered : strings.viewer.htmlSource);
  const selectClass = "h-11 min-w-0 rounded-lg border-0 bg-transparent px-1 text-xs text-fv-text-secondary outline-none hover:bg-fv-surface-input focus-visible:ring-2 focus-visible:ring-fv-accent disabled:opacity-40";

  if (error) return <UnavailableText model={model} failed={error === "fetch"} />;
  if (content === null) return <div role="status" className="p-6 text-sm text-fv-text-muted">{t(strings.viewer.loadingPreview)}</div>;
  if (content === "") return <EmptyText />;
  const sourceProps = { ...props, text: text ?? { resolvedPath: model.resolvedPath, kind: model.kind, mimeType: model.mimeType, content, truncated: false } };
  return (
    <div className="flex h-full min-h-0 min-w-0 flex-col" data-testid="file-preview-html">
      <div role="toolbar" aria-label={t(strings.viewer.htmlControls)} className="flex shrink-0 items-center gap-1 border-b border-fv-default px-2 py-1 sm:px-3">
        <IconButton
          type="button"
          swapIdentity={`html-view-toggle-${model.previewId}`}
          surface="ghost"
          shape="rounded"
          className="h-11 w-11 shrink-0"
          aria-label={toggleLabel}
          title={toggleLabel}
          onClick={() => { setSource((previous) => !previous); }}
        >
          {source ? <Eye /> : <Code2 />}
        </IconButton>
        <select
          aria-label={t(strings.viewer.htmlViewport)}
          title={t(strings.viewer.htmlViewport)}
          value={viewport}
          disabled={source}
          onChange={(event) => { setViewport(event.target.value); setZoom("fit"); }}
          className={cn(selectClass, "flex-1")}
        >
          {HTML_VIEWPORTS.map((preset) => (
            <option key={preset.value} value={preset.value} className="bg-fv-surface-raised">
              {t(strings.viewer[preset.label])}{preset.width ? ` · ${String(preset.width)}px` : ""}
            </option>
          ))}
        </select>
        <div className="flex shrink-0 items-center">
          <IconButton
            type="button" surface="ghost" shape="rounded" className="h-11 w-11"
            aria-label={t(strings.viewer.htmlZoomOut)} title={t(strings.viewer.htmlZoomOut)}
            disabled={source || smallerZoom === undefined}
            onClick={() => { if (smallerZoom !== undefined) setZoom(smallerZoom); }}
          ><Minus /></IconButton>
          <select
            aria-label={t(strings.viewer.htmlZoom)}
            title={zoom === "fit" ? `${t(strings.viewer.htmlZoomFit)} · ${String(Math.round(scale * 100))}%` : t(strings.viewer.htmlZoom)}
            value={zoom}
            disabled={source}
            onChange={(event) => { setZoom(event.target.value === "fit" ? "fit" : Number(event.target.value)); }}
            className={cn(selectClass, "w-16 tabular-nums")}
          >
            <option value="fit" className="bg-fv-surface-raised">{t(strings.viewer.htmlZoomFit)}</option>
            {HTML_ZOOM_STEPS.map((step) => <option key={step} value={step} className="bg-fv-surface-raised">{Math.round(step * 100)}%</option>)}
          </select>
          <IconButton
            type="button" surface="ghost" shape="rounded" className="h-11 w-11"
            aria-label={t(strings.viewer.htmlZoomIn)} title={t(strings.viewer.htmlZoomIn)}
            disabled={source || largerZoom === undefined}
            onClick={() => { if (largerZoom !== undefined) setZoom(largerZoom); }}
          ><Plus /></IconButton>
        </div>
      </div>
      {source ? <div className="min-h-0 flex-1"><SourcePreview {...sourceProps} /></div> : <>
        <p className="shrink-0 px-3 py-2 text-xs text-fv-text-muted">{t(strings.viewer.htmlLimitations)}</p>
        {text?.truncated && <PreviewNotice message={t(strings.viewer.truncatedNotice)} tone="info" />}
        <div ref={stageRef} className="min-h-0 min-w-0 flex-1 overflow-auto bg-fv-surface-base">
          <div ref={inspectionRef} className="relative mx-auto overflow-hidden" style={{ contain: "paint", width: viewportWidth * scale || "100%", height: stageSize.height || "100%" }}>
            <iframe
              ref={frameRef}
              title={t(strings.viewer.htmlPreview, { name: model.basename })}
              sandbox="allow-scripts"
              referrerPolicy="no-referrer"
              srcDoc={htmlPreviewDocument(content, previewSource?.capabilities.htmlRuntime ? model.runtimeHref : undefined, props.anchor)}
              className="absolute left-0 top-0 max-w-none origin-top-left border-0 bg-white [color-scheme:light]"
              style={{ willChange: "transform", width: viewportWidth || "100%", height: stageSize.height ? stageSize.height / scale : "100%", transform: `scale(${String(scale)})` }}
            />
          </div>
        </div>
      </>}
    </div>
  );
}

function UnavailableText({ model, failed = false }: { model: PreviewRendererProps["model"]; failed?: boolean }) {
  const { t } = useFileViewerText();
  return <div className="flex h-full flex-col items-center justify-center gap-4 overflow-auto p-6 text-center" data-testid="file-preview-text-unavailable">
    <PreviewNotice message={t(failed ? strings.viewer.mediaLoadError : strings.viewer.textUnavailable)} />
    <PreviewMetaLine model={model} />
    <PreviewActions model={model} className="justify-center" />
  </div>;
}

function EmptyText() {
  const { t } = useFileViewerText();
  return (
    <div className="flex h-full items-center justify-center text-sm text-fv-text-muted" data-testid="file-preview-empty">
      {t(strings.viewer.emptyFile)}
    </div>
  );
}
