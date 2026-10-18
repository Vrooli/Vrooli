import { useEffect, useMemo, useRef, useState } from "react";
import { Code2, Eye, Minus, Plus, WrapText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { IconButton } from "@vrooli/react-component-library/IconButton";

import { StructuredPreview } from "./StructuredPreview";
import { isStructuredPath } from "../structuredData";
import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import {
  getLanguageFromPath,
  highlightCode,
  type HighlightToken,
  type HighlightedLine,
} from "../../../lib/codeHighlighter";
import { PreviewNotice } from "./shared";
import { attachHtmlPreviewInspection, htmlPreviewPinchScript } from "../htmlPreviewPinch";
import type { PreviewRendererProps } from "../types";

export { MarkdownPreview } from "./MarkdownPreview";

// CodePreview renders code/text/diff fallback content with line numbers,
// syntax highlighting, font sizing, wrap toggle, and target-line scroll.
export function CodePreview(props: PreviewRendererProps) {
  const { model } = props;
  // HTML uses the existing bounded text transport. Keep API downloads inert;
  // only this isolated renderer interprets the markup.
  if (/\.html?$/i.test(model.resolvedPath) || model.mimeType.split(";")[0]?.trim().toLowerCase() === "text/html") {
    return <HtmlPreview key={model.previewId} {...props} />;
  }
  if (isStructuredPath(model.resolvedPath)) return <StructuredPreview {...props} />;
  return <SourcePreview {...props} />;
}

function SourcePreview({ model, text }: PreviewRendererProps) {
  const content = text?.content ?? "";
  if (content === "") {
    return <EmptyText />;
  }
  return (
    <CodeLinePreview content={content} path={model.resolvedPath} highlightLine={model.line ?? null} truncated={!!text?.truncated} />
  );
}

function escapeHtmlAttribute(value: string): string {
  return value.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// Fetching the HTML through srcDoc avoids a second document navigation through
// an Access/tunnel boundary. The base still points at the session-bound runtime
// route, so relative CSS, JS, images, and child pages resolve normally.
function htmlPreviewDocument(content: string, runtimeHref?: string): string {
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
  return `<!doctype html><meta http-equiv="Content-Security-Policy" content="${documentPolicy}"><base href="${escapeHtmlAttribute(baseHref)}"><meta name="referrer" content="no-referrer">${htmlPreviewPinchScript()}${content}`;
}

const HTML_VIEWPORTS = [
  { value: "responsive", width: 0, label: "htmlViewportResponsive" },
  { value: "phone", width: 390, label: "htmlViewportPhone" },
  { value: "tablet", width: 768, label: "htmlViewportTablet" },
  { value: "desktop", width: 1440, label: "htmlViewportDesktop" },
] as const;
const HTML_ZOOM_STEPS = [0.1, 0.25, 0.5, 0.75, 1, 1.25, 1.5, 2];

function HtmlPreview(props: PreviewRendererProps) {
  const { model, text } = props;
  const { t } = useTranslation();
  const [source, setSource] = useState(!!model.line || !!text?.truncated);
  const [viewport, setViewport] = useState("responsive");
  const [zoom, setZoom] = useState<number | "fit">("fit");
  const stageRef = useRef<HTMLDivElement>(null);
  const inspectionRef = useRef<HTMLDivElement>(null);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const [stageSize, setStageSize] = useState({ width: 0, height: 0 });

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
  }, [source]);

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
  const toggleLabel = t(source ? strings.messagesFileViewer.htmlRendered : strings.messagesFileViewer.htmlSource);
  const selectClass = "h-11 min-w-0 rounded-lg border-0 bg-transparent px-1 text-xs text-wc-text-secondary outline-none hover:bg-wc-surface-input focus-visible:ring-2 focus-visible:ring-wc-accent disabled:opacity-40";

  if (!text?.content) return <EmptyText />;
  return (
    <div className="flex h-full min-h-0 min-w-0 flex-col" data-testid="file-preview-html">
      <div role="toolbar" aria-label={t(strings.messagesFileViewer.htmlControls)} className="flex shrink-0 items-center gap-1 border-b border-wc-default px-2 py-1 sm:px-3">
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
          aria-label={t(strings.messagesFileViewer.htmlViewport)}
          title={t(strings.messagesFileViewer.htmlViewport)}
          value={viewport}
          disabled={source}
          onChange={(event) => { setViewport(event.target.value); setZoom("fit"); }}
          className={cn(selectClass, "flex-1")}
        >
          {HTML_VIEWPORTS.map((preset) => (
            <option key={preset.value} value={preset.value} className="bg-wc-surface-raised">
              {t(strings.messagesFileViewer[preset.label])}{preset.width ? ` · ${String(preset.width)}px` : ""}
            </option>
          ))}
        </select>
        <div className="flex shrink-0 items-center">
          <IconButton
            type="button" surface="ghost" shape="rounded" className="h-11 w-11"
            aria-label={t(strings.messagesFileViewer.htmlZoomOut)} title={t(strings.messagesFileViewer.htmlZoomOut)}
            disabled={source || smallerZoom === undefined}
            onClick={() => { if (smallerZoom !== undefined) setZoom(smallerZoom); }}
          ><Minus /></IconButton>
          <select
            aria-label={t(strings.messagesFileViewer.htmlZoom)}
            title={zoom === "fit" ? `${t(strings.messagesFileViewer.htmlZoomFit)} · ${String(Math.round(scale * 100))}%` : t(strings.messagesFileViewer.htmlZoom)}
            value={zoom}
            disabled={source}
            onChange={(event) => { setZoom(event.target.value === "fit" ? "fit" : Number(event.target.value)); }}
            className={cn(selectClass, "w-16 tabular-nums")}
          >
            <option value="fit" className="bg-wc-surface-raised">{t(strings.messagesFileViewer.htmlZoomFit)}</option>
            {HTML_ZOOM_STEPS.map((step) => <option key={step} value={step} className="bg-wc-surface-raised">{Math.round(step * 100)}%</option>)}
          </select>
          <IconButton
            type="button" surface="ghost" shape="rounded" className="h-11 w-11"
            aria-label={t(strings.messagesFileViewer.htmlZoomIn)} title={t(strings.messagesFileViewer.htmlZoomIn)}
            disabled={source || largerZoom === undefined}
            onClick={() => { if (largerZoom !== undefined) setZoom(largerZoom); }}
          ><Plus /></IconButton>
        </div>
      </div>
      {source ? <div className="min-h-0 flex-1"><SourcePreview {...props} /></div> : <>
        <p className="shrink-0 px-3 py-2 text-xs text-wc-text-muted">{t(strings.messagesFileViewer.htmlLimitations)}</p>
        {text.truncated && <PreviewNotice message={t(strings.messagesFileViewer.truncatedNotice)} tone="info" />}
        <div ref={stageRef} className="min-h-0 min-w-0 flex-1 overflow-auto bg-wc-surface-base">
          <div ref={inspectionRef} className="relative mx-auto overflow-hidden" style={{ contain: "paint", width: viewportWidth * scale || "100%", height: stageSize.height || "100%" }}>
            <iframe
              ref={frameRef}
              title={t(strings.messagesFileViewer.htmlPreview, { name: model.basename })}
              sandbox="allow-scripts"
              referrerPolicy="no-referrer"
              srcDoc={htmlPreviewDocument(text.content, model.runtimeHref)}
              className="absolute left-0 top-0 max-w-none origin-top-left border-0 bg-white [color-scheme:light]"
              style={{ willChange: "transform", width: viewportWidth || "100%", height: stageSize.height ? stageSize.height / scale : "100%", transform: `scale(${String(scale)})` }}
            />
          </div>
        </div>
      </>}
    </div>
  );
}

function EmptyText() {
  const { t } = useTranslation();
  return (
    <div className="flex h-full items-center justify-center text-sm text-wc-text-muted" data-testid="file-preview-empty">
      {t(strings.messagesFileViewer.emptyFile)}
    </div>
  );
}

const FONT_SIZES: readonly number[] = [11, 12, 13, 14, 15, 16, 18];
const MIN_FONT_SIZE = 11;
const MAX_FONT_SIZE = 18;
const DEFAULT_FONT_SIZE = 13;

export function CodeLinePreview({
  content,
  path,
  highlightLine,
  truncated = false,
}: {
  content: string;
  path: string;
  highlightLine: number | null;
  truncated?: boolean;
}) {
  const { t } = useTranslation();
  const scrollerRef = useRef<HTMLDivElement | null>(null);
  const lineRefs = useRef<Record<number, HTMLDivElement | null>>({});
  const plainLines = useMemo(() => content.split("\n"), [content]);
  const [highlighted, setHighlighted] = useState<HighlightedLine[] | null>(null);
  const [wrap, setWrap] = useState(false);
  const [fontSize, setFontSize] = useState<number>(DEFAULT_FONT_SIZE);
  const language = useMemo(() => getLanguageFromPath(path), [path]);

  useEffect(() => {
    let cancelled = false;
    setHighlighted(null);
    highlightCode(content, language)
      .then((lines) => {
        if (!cancelled) setHighlighted(lines);
      })
      .catch(() => {
        if (!cancelled) setHighlighted(null);
      });
    return () => {
      cancelled = true;
    };
  }, [content, language]);

  useEffect(() => {
    if (!highlightLine) return;
    const node = lineRefs.current[highlightLine];
    if (!node) return;
    const raf = requestAnimationFrame(() => {
      node.scrollIntoView({ block: "center", behavior: "smooth" });
    });
    return () => { cancelAnimationFrame(raf); };
  }, [highlightLine, highlighted, plainLines.length, wrap, fontSize]);

  const lineCount = highlighted?.length ?? plainLines.length;
  const gutterWidth = `${String(String(lineCount).length)}ch`;
  const lines =
    highlighted ??
    plainLines.map((line, i) => ({
      lineNumber: i + 1,
      tokens: [{ content: line }] as HighlightToken[],
    }));

  const adjustFont = (direction: 1 | -1) => {
    setFontSize((prev) => {
      const idx = FONT_SIZES.indexOf(prev);
      const fallback = FONT_SIZES.indexOf(DEFAULT_FONT_SIZE);
      const current = idx === -1 ? fallback : idx;
      const next = Math.max(0, Math.min(FONT_SIZES.length - 1, current + direction));
      return FONT_SIZES[next] ?? prev;
    });
  };

  return (
    <div className="flex h-full flex-col bg-[#0d1117]" data-testid="file-preview-code">
      <div className="flex shrink-0 items-center justify-between gap-2 border-b border-wc-default/60 bg-wc-surface-base px-3 py-1.5 text-xs text-wc-text-muted">
        <span className="truncate font-mono text-[11px] uppercase tracking-wide">
          {language ?? t(strings.messagesFileViewer.plaintext)} · {t(strings.messagesFileViewer.linesSuffix, { count: lineCount })}
          {truncated ? ` · ${t(strings.messagesFileViewer.truncatedNotice)}` : ""}
        </span>
        <div className="flex shrink-0 items-center gap-1">
          <button
            type="button"
            onClick={() => { adjustFont(-1); }}
            className="rounded p-1 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-40"
            aria-label={t(strings.messagesFileViewer.decreaseFontSize)}
            title={t(strings.messagesFileViewer.decreaseFontSize)}
            disabled={fontSize <= MIN_FONT_SIZE}
          >
            <Minus className="h-3.5 w-3.5" />
          </button>
          <span className="tabular-nums text-[11px]">{fontSize}px</span>
          <button
            type="button"
            onClick={() => { adjustFont(1); }}
            className="rounded p-1 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-40"
            aria-label={t(strings.messagesFileViewer.increaseFontSize)}
            title={t(strings.messagesFileViewer.increaseFontSize)}
            disabled={fontSize >= MAX_FONT_SIZE}
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            onClick={() => { setWrap((prev) => !prev); }}
            className={cn(
              "ms-1 flex items-center gap-1 rounded px-1.5 py-1 text-[11px] transition hover:bg-wc-surface-input hover:text-wc-text-primary",
              wrap && "bg-wc-accent/15 text-wc-accent hover:bg-wc-accent/20 hover:text-wc-accent",
            )}
            aria-label={wrap ? t(strings.messagesFileViewer.disableWordWrap) : t(strings.messagesFileViewer.enableWordWrap)}
            aria-pressed={wrap}
            title={wrap ? t(strings.messagesFileViewer.disableWordWrap) : t(strings.messagesFileViewer.enableWordWrap)}
          >
            <WrapText className="h-3.5 w-3.5" />
            <span>{t(strings.messagesFileViewer.wrap)}</span>
          </button>
        </div>
      </div>
      <div
        ref={scrollerRef}
        className="min-h-0 flex-1 overflow-auto font-mono leading-[1.55]"
        style={{ fontSize: `${String(fontSize)}px` }}
      >
        {lines.map((line) => {
          const lineNumber = line.lineNumber;
          const isHighlighted = lineNumber === highlightLine;
          return (
            <div
              key={lineNumber}
              ref={(node) => {
                lineRefs.current[lineNumber] = node;
              }}
              className={cn("flex items-start gap-3 px-3 py-px", isHighlighted && "bg-wc-accent/15")}
            >
              <span
                className={cn(
                  "shrink-0 select-none text-end text-wc-text-faint/70 tabular-nums",
                  isHighlighted && "text-wc-accent",
                )}
                style={{ minWidth: gutterWidth, fontSize: `${String(Math.max(10, fontSize - 1))}px` }}
              >
                {lineNumber}
              </span>
              <pre
                className={cn(
                  "m-0 flex-1 bg-transparent p-0 text-[#c9d1d9]",
                  wrap ? "whitespace-pre-wrap break-words" : "whitespace-pre",
                )}
              >
                {line.tokens.length === 0 || (line.tokens.length === 1 && line.tokens[0]?.content === "")
                  ? " "
                  : line.tokens.map((token, i) => (
                      <span
                        key={i}
                        style={{ color: token.color }}
                        className={token.fontStyle === "italic" ? "italic" : token.fontStyle === "bold" ? "font-bold" : ""}
                      >
                        {token.content}
                      </span>
                    ))}
              </pre>
            </div>
          );
        })}
      </div>
    </div>
  );
}
