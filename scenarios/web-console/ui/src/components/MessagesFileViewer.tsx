import { useEffect, useMemo, useRef, useState } from "react";
import { AlertTriangle, Check, Copy, Loader2, Minus, Plus, WrapText, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { MarkdownRenderer } from "./markdown";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import {
  getLanguageFromPath,
  highlightCode,
  type HighlightToken,
  type HighlightedLine,
} from "../lib/codeHighlighter";
import type { FileReferenceContentResponse, FileReferenceResolveResponse } from "../api/conversation";

interface MessagesFileViewerProps {
  open: boolean;
  loading: boolean;
  error: string | null;
  requestedPath: string | null;
  resolved: FileReferenceResolveResponse | null;
  content: FileReferenceContentResponse | null;
  onClose: () => void;
}

export default function MessagesFileViewer({
  open,
  loading,
  error,
  requestedPath,
  resolved,
  content,
  onClose,
}: MessagesFileViewerProps) {
  const { t } = useTranslation();
  const targetLine = content?.line ?? resolved?.line ?? null;
  const displayPath = content?.path ?? resolved?.resolved_path ?? requestedPath ?? "";
  const [copied, setCopied] = useState(false);
  const copyPath = () => {
    if (!displayPath) return;
    void navigator.clipboard.writeText(displayPath);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  const basename = useMemo(() => {
    const fullPath = content?.path ?? resolved?.resolved_path ?? requestedPath ?? "";
    if (!fullPath) return t(strings.messagesFileViewer.filePreviewFallback);
    const parts = fullPath.split(/[\\/]/);
    return parts[parts.length - 1] || fullPath;
  }, [content?.path, requestedPath, resolved?.resolved_path, t]);

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose, open]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[80]">
      <div className="absolute inset-0 bg-wc-backdrop" onClick={onClose} />
      <div className="absolute inset-x-0 bottom-0 top-4 flex flex-col overflow-hidden rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised shadow-2xl md:inset-x-8 md:bottom-8 md:top-8 md:rounded-2xl md:border">
        <div className="shrink-0 border-b border-wc-default px-4 py-3">
          <div className="flex items-center gap-3">
            <h2 className="min-w-0 flex-1 truncate text-sm font-semibold text-wc-text-primary">{basename}</h2>
            <button
              type="button"
              onClick={onClose}
              className="shrink-0 rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
              aria-label={t(strings.messagesFileViewer.closeAriaLabel)}
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="mt-1 flex items-center gap-1.5">
            <p className="min-w-0 flex-1 truncate text-xs text-wc-text-muted">
              {displayPath || t(strings.messagesFileViewer.loadingFile)}
            </p>
            {displayPath && (
              <button
                type="button"
                onClick={copyPath}
                className="shrink-0 rounded p-1 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
                aria-label={copied ? t(strings.messagesFileViewer.copied) : t(strings.messagesFileViewer.copyPath)}
                title={copied ? t(strings.messagesFileViewer.copied) : t(strings.messagesFileViewer.copyPath)}
              >
                {copied ? (
                  <Check className="h-3.5 w-3.5 text-green-400" />
                ) : (
                  <Copy className="h-3.5 w-3.5" />
                )}
              </button>
            )}
          </div>
          {(resolved?.resolution_basis || content?.category || targetLine) && (
            <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-wc-text-faint">
              {resolved?.resolution_basis && (
                <span className="rounded-full border border-wc-default px-2 py-0.5">
                  {resolved.resolution_basis}
                </span>
              )}
              {content?.category && (
                <span className="rounded-full border border-wc-default px-2 py-0.5">
                  {content.category}
                </span>
              )}
              {targetLine && (
                <span className="rounded-full border border-wc-default px-2 py-0.5">
                  {t(strings.messagesFileViewer.linePrefix, { line: targetLine })}
                </span>
              )}
            </div>
          )}
        </div>

        <div className="min-h-0 flex-1 overflow-hidden">
          {loading && (
            <div className="flex h-full items-center justify-center gap-2 text-wc-text-muted">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>{t(strings.messagesFileViewer.loadingPreview)}</span>
            </div>
          )}

          {!loading && error && (
            <div className="h-full overflow-auto px-4 py-4">
              <div className="mx-auto max-w-2xl rounded-2xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-300">
                <div className="mb-2 flex items-center gap-2 font-medium">
                  <AlertTriangle className="h-4 w-4" />
                  <span>{t(strings.messagesFileViewer.unavailable)}</span>
                </div>
                <p>{error}</p>
                {requestedPath && (
                  <p className="mt-2 break-all text-xs text-red-200/80">{t(strings.messagesFileViewer.requestedPrefix, { path: requestedPath })}</p>
                )}
              </div>
            </div>
          )}

          {!loading && !error && content && (
            content.category === "markdown" ? (
              <div className="h-full overflow-auto px-4 py-4">
                <MarkdownRenderer content={content.content} className="text-sm" />
              </div>
            ) : (
              <CodeLinePreview
                content={content.content}
                path={displayPath}
                highlightLine={targetLine}
              />
            )
          )}
        </div>
      </div>
    </div>
  );
}

const FONT_SIZES: readonly number[] = [11, 12, 13, 14, 15, 16, 18];
const MIN_FONT_SIZE = 11;
const MAX_FONT_SIZE = 18;
const DEFAULT_FONT_SIZE = 13;

function CodeLinePreview({
  content,
  path,
  highlightLine,
}: {
  content: string;
  path: string;
  highlightLine: number | null;
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
    return () => cancelAnimationFrame(raf);
  }, [highlightLine, highlighted, plainLines.length, wrap, fontSize]);

  const lineCount = highlighted?.length ?? plainLines.length;
  const gutterWidth = `${String(lineCount).length}ch`;
  const lines = highlighted ?? plainLines.map((line, i) => ({
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
    <div className="flex h-full flex-col bg-[#0d1117]">
      <div className="flex shrink-0 items-center justify-between gap-2 border-b border-wc-default/60 bg-wc-surface-base px-3 py-1.5 text-xs text-wc-text-muted">
        <span className="truncate font-mono text-[11px] uppercase tracking-wide">
          {language ?? t(strings.messagesFileViewer.plaintext)} · {t(strings.messagesFileViewer.linesSuffix, { count: lineCount })}
        </span>
        <div className="flex shrink-0 items-center gap-1">
          <button
            type="button"
            onClick={() => adjustFont(-1)}
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
            onClick={() => adjustFont(1)}
            className="rounded p-1 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-40"
            aria-label={t(strings.messagesFileViewer.increaseFontSize)}
            title={t(strings.messagesFileViewer.increaseFontSize)}
            disabled={fontSize >= MAX_FONT_SIZE}
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            onClick={() => setWrap((prev) => !prev)}
            className={cn(
              "ml-1 flex items-center gap-1 rounded px-1.5 py-1 text-[11px] transition hover:bg-wc-surface-input hover:text-wc-text-primary",
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
        style={{ fontSize: `${fontSize}px` }}
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
              className={cn(
                "flex items-start gap-3 px-3 py-px",
                isHighlighted && "bg-wc-accent/15",
              )}
            >
              <span
                className={cn(
                  "shrink-0 select-none text-right text-wc-text-faint/70 tabular-nums",
                  isHighlighted && "text-wc-accent",
                )}
                style={{ minWidth: gutterWidth, fontSize: `${Math.max(10, fontSize - 1)}px` }}
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
