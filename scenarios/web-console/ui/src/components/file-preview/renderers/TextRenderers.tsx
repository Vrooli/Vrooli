import { useEffect, useMemo, useRef, useState } from "react";
import { Minus, Plus, WrapText } from "lucide-react";
import { useTranslation } from "react-i18next";

import { MarkdownRenderer } from "../../markdown";
import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import {
  getLanguageFromPath,
  highlightCode,
  type HighlightToken,
  type HighlightedLine,
} from "../../../lib/codeHighlighter";
import { PreviewNotice } from "./shared";
import type { PreviewRendererProps } from "../types";

// MarkdownPreview renders text-kind markdown via the shared MarkdownRenderer.
export function MarkdownPreview({ text }: PreviewRendererProps) {
  const { t } = useTranslation();
  const content = text?.content ?? "";
  if (content.trim() === "") {
    return <EmptyText />;
  }
  return (
    <div className="h-full overflow-auto px-4 py-4" data-testid="file-preview-markdown">
      {text?.truncated && (
        <div className="mb-3">
          <PreviewNotice message={t(strings.messagesFileViewer.truncatedNotice)} tone="info" />
        </div>
      )}
      <MarkdownRenderer content={content} className="text-sm" />
    </div>
  );
}

// CodePreview renders code/text/diff fallback content with line numbers,
// syntax highlighting, font sizing, wrap toggle, and target-line scroll.
export function CodePreview({ model, text }: PreviewRendererProps) {
  const content = text?.content ?? "";
  if (content === "") {
    return <EmptyText />;
  }
  return (
    <CodeLinePreview content={content} path={model.resolvedPath} highlightLine={model.line ?? null} truncated={!!text?.truncated} />
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
    return () => cancelAnimationFrame(raf);
  }, [highlightLine, highlighted, plainLines.length, wrap, fontSize]);

  const lineCount = highlighted?.length ?? plainLines.length;
  const gutterWidth = `${String(lineCount).length}ch`;
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
              className={cn("flex items-start gap-3 px-3 py-px", isHighlighted && "bg-wc-accent/15")}
            >
              <span
                className={cn(
                  "shrink-0 select-none text-end text-wc-text-faint/70 tabular-nums",
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
