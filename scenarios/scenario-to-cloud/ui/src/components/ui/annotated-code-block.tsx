import { memo, useEffect, useState, useRef } from "react";
import { Check, Copy, AlertCircle, AlertTriangle } from "lucide-react";
import { cn } from "../../lib/utils";
import {
  detectLanguage,
  getLanguageFromFilename,
  normalizeLanguage,
} from "../../lib/languageDetection";

export interface LineAnnotation {
  /** 1-indexed line number */
  line: number;
  /** Severity of the issue */
  severity: "error" | "warn";
  /** The issue message */
  message: string;
  /** Optional hint for fixing */
  hint?: string;
  /** The JSON path this annotation refers to */
  path: string;
  /** Whether this issue can be auto-fixed */
  fixable?: boolean;
}

interface AnnotatedCodeBlockProps {
  /** The code content to display */
  code: string;
  /** Filename (used to detect language from extension) */
  filename?: string;
  /** Language hint (overrides auto-detection) */
  language?: string;
  /** Maximum height before scrolling */
  maxHeight?: string;
  /** Whether to show line numbers */
  showLineNumbers?: boolean;
  /** Whether to show the header with language label and copy button */
  showHeader?: boolean;
  /** Line annotations to display */
  annotations?: LineAnnotation[];
  /** Callback when an annotation is clicked */
  onAnnotationClick?: (annotation: LineAnnotation) => void;
  /** Additional CSS classes */
  className?: string;
  /** Whether the code is editable (enables textarea overlay) */
  editable?: boolean;
  /** Callback when code changes (only called if editable=true) */
  onChange?: (code: string) => void;
  /** Placeholder text for editable mode */
  placeholder?: string;
  /** Error message to display below the editor */
  error?: string;
  /** Test ID for e2e testing (applied to textarea when editable, otherwise to container) */
  testId?: string;
}

// Lazy-loaded shiki highlighter
let highlighterPromise: Promise<import("shiki").Highlighter> | null = null;

async function getHighlighter() {
  if (!highlighterPromise) {
    highlighterPromise = import("shiki").then((shiki) =>
      shiki.createHighlighter({
        themes: ["github-dark"],
        langs: [
          "typescript",
          "javascript",
          "python",
          "go",
          "json",
          "bash",
          "sql",
          "html",
          "css",
          "yaml",
          "markdown",
          "jsx",
          "tsx",
        ],
      })
    );
  }
  return highlighterPromise;
}

interface TooltipState {
  visible: boolean;
  x: number;
  y: number;
  annotation: LineAnnotation | null;
}

/**
 * Syntax-highlighted code block with line annotations for validation errors/warnings.
 * Extends the base CodeBlock with gutter icons, line highlighting, and hover tooltips.
 */
export const AnnotatedCodeBlock = memo(function AnnotatedCodeBlock({
  code,
  filename,
  language,
  maxHeight = "400px",
  showLineNumbers = true,
  showHeader = true,
  annotations = [],
  onAnnotationClick,
  className,
  editable = false,
  onChange,
  placeholder,
  error,
  testId,
}: AnnotatedCodeBlockProps) {
  const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [tooltip, setTooltip] = useState<TooltipState>({
    visible: false,
    x: 0,
    y: 0,
    annotation: null,
  });
  const containerRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const codeContentRef = useRef<HTMLDivElement>(null);

  // Build a map of line number to annotations for quick lookup
  const annotationsByLine = new Map<number, LineAnnotation[]>();
  for (const annotation of annotations) {
    const existing = annotationsByLine.get(annotation.line) || [];
    existing.push(annotation);
    annotationsByLine.set(annotation.line, existing);
  }

  // Determine language from props, filename, or auto-detection
  const detectedLang = language
    ? normalizeLanguage(language)
    : filename
      ? getLanguageFromFilename(filename) || detectLanguage(code)
      : detectLanguage(code);

  // Display name for the language label
  const displayLang = detectedLang === "text" ? "" : detectedLang;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  useEffect(() => {
    let cancelled = false;

    async function highlight() {
      try {
        const highlighter = await getHighlighter();
        if (cancelled) return;

        // Check if the language is supported
        const langs = highlighter.getLoadedLanguages();
        const langToUse = langs.includes(detectedLang) ? detectedLang : "text";

        const html = highlighter.codeToHtml(code, {
          lang: langToUse,
          theme: "github-dark",
        });

        if (!cancelled) {
          setHighlightedHtml(html);
        }
      } catch (err) {
        console.warn("Syntax highlighting failed:", err);
      }
    }

    highlight();

    return () => {
      cancelled = true;
    };
  }, [code, detectedLang]);

  const lines = code.split("\n");
  const lineNumberWidth = String(lines.length).length;

  const handleLineMouseEnter = (
    e: React.MouseEvent<HTMLDivElement>,
    lineAnnotations: LineAnnotation[]
  ) => {
    if (lineAnnotations.length === 0) return;

    const rect = e.currentTarget.getBoundingClientRect();
    const containerRect = containerRef.current?.getBoundingClientRect();

    if (!containerRect) return;

    setTooltip({
      visible: true,
      x: rect.right - containerRect.left + 8,
      y: rect.top - containerRect.top,
      annotation: lineAnnotations[0], // Show first annotation (usually there's only one per line)
    });
  };

  const handleLineMouseLeave = () => {
    setTooltip((prev) => ({ ...prev, visible: false }));
  };

  const handleAnnotationClick = (annotation: LineAnnotation) => {
    onAnnotationClick?.(annotation);
  };

  // Sync scroll between textarea and code content
  const handleScroll = (e: React.UIEvent<HTMLTextAreaElement | HTMLDivElement>) => {
    const source = e.currentTarget;
    const targets = [textareaRef.current, codeContentRef.current].filter(
      (el) => el && el !== source
    ) as HTMLElement[];

    for (const target of targets) {
      target.scrollTop = source.scrollTop;
      target.scrollLeft = source.scrollLeft;
    }
  };

  const handleTextareaChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    onChange?.(e.target.value);
  };

  // Get the most severe annotation for a line (error > warn)
  const getMostSevereAnnotation = (lineAnnotations: LineAnnotation[]) => {
    return lineAnnotations.find((a) => a.severity === "error") || lineAnnotations[0];
  };

  // Count errors and warnings
  const errorCount = annotations.filter((a) => a.severity === "error").length;
  const warnCount = annotations.filter((a) => a.severity === "warn").length;

  return (
    <div
      ref={containerRef}
      className={cn(
        "relative group rounded-lg overflow-hidden border border-slate-700",
        className
      )}
      data-testid={!editable ? testId : undefined}
    >
      {/* Header with language label, annotation summary, and copy button */}
      {showHeader && (
        <div className="flex items-center justify-between px-4 py-2 bg-slate-900 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <span className="text-xs text-slate-400 font-mono uppercase">
              {displayLang}
            </span>
            {/* Annotation summary */}
            {(errorCount > 0 || warnCount > 0) && (
              <div className="flex items-center gap-2">
                {errorCount > 0 && (
                  <span className="flex items-center gap-1 text-xs text-red-400">
                    <AlertCircle className="h-3 w-3" />
                    {errorCount} error{errorCount !== 1 ? "s" : ""}
                  </span>
                )}
                {warnCount > 0 && (
                  <span className="flex items-center gap-1 text-xs text-amber-400">
                    <AlertTriangle className="h-3 w-3" />
                    {warnCount} warning{warnCount !== 1 ? "s" : ""}
                  </span>
                )}
              </div>
            )}
          </div>
          <button
            onClick={handleCopy}
            className={cn(
              "flex items-center gap-1.5 text-xs transition-colors",
              copied
                ? "text-emerald-400"
                : "text-slate-400 hover:text-slate-200"
            )}
            aria-label={copied ? "Copied" : "Copy code"}
          >
            {copied ? (
              <>
                <Check className="h-3.5 w-3.5" />
                <span>Copied</span>
              </>
            ) : (
              <>
                <Copy className="h-3.5 w-3.5" />
                <span>Copy</span>
              </>
            )}
          </button>
        </div>
      )}

      {/* Code content */}
      <div
        ref={codeContentRef}
        className="bg-slate-950 overflow-auto relative"
        style={{ maxHeight }}
        onScroll={editable ? handleScroll : undefined}
      >
        {/* Editable textarea overlay */}
        {editable && (
          <textarea
            ref={textareaRef}
            data-testid={testId}
            value={code}
            onChange={handleTextareaChange}
            onScroll={handleScroll}
            placeholder={placeholder}
            spellCheck={false}
            className={cn(
              "absolute inset-0 w-full h-full resize-none",
              "bg-transparent text-transparent caret-white",
              "font-mono text-sm leading-relaxed",
              "p-4 pl-[calc(theme(spacing.2)+theme(spacing.5)+theme(spacing.2)+3ch+theme(spacing.2)+1px)]",
              "focus:outline-none z-10",
              showLineNumbers && "pl-[calc(theme(spacing.2)+theme(spacing.5)+theme(spacing.2)+3ch+theme(spacing.2)+1px)]",
              !showLineNumbers && "pl-[calc(theme(spacing.2)+theme(spacing.5)+theme(spacing.4))]"
            )}
            style={{
              // Match the line height of the highlighted code
              lineHeight: "1.625rem",
            }}
          />
        )}
        <div className="flex">
          {/* Annotation gutter */}
          <div className="flex-shrink-0 py-4 pl-2 pr-1 select-none">
            <div className="text-sm font-mono leading-relaxed">
              {lines.map((_, i) => {
                const lineNum = i + 1;
                const lineAnnotations = annotationsByLine.get(lineNum) || [];
                const mostSevere = getMostSevereAnnotation(lineAnnotations);

                return (
                  <div
                    key={i}
                    className={cn(
                      "h-[1.625rem] flex items-center justify-center w-5",
                      lineAnnotations.length > 0 && "cursor-pointer"
                    )}
                    onMouseEnter={(e) => handleLineMouseEnter(e, lineAnnotations)}
                    onMouseLeave={handleLineMouseLeave}
                    onClick={() => mostSevere && handleAnnotationClick(mostSevere)}
                  >
                    {mostSevere?.severity === "error" && (
                      <AlertCircle className="h-4 w-4 text-red-500" />
                    )}
                    {mostSevere?.severity === "warn" && !lineAnnotations.some(a => a.severity === "error") && (
                      <AlertTriangle className="h-4 w-4 text-amber-500" />
                    )}
                  </div>
                );
              })}
            </div>
          </div>

          {/* Line numbers */}
          {showLineNumbers && (
            <div className="flex-shrink-0 py-4 pr-2 select-none border-r border-slate-800">
              <div className="text-sm font-mono text-slate-600 text-right leading-relaxed">
                {lines.map((_, i) => (
                  <div
                    key={i}
                    style={{ minWidth: `${lineNumberWidth}ch` }}
                    className="h-[1.625rem] flex items-center justify-end"
                  >
                    {i + 1}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Highlighted code with line backgrounds */}
          {highlightedHtml ? (
            <div className="flex-1 relative">
              {/* Line backgrounds for annotations */}
              <div className="absolute inset-0 py-4">
                {lines.map((_, i) => {
                  const lineNum = i + 1;
                  const lineAnnotations = annotationsByLine.get(lineNum) || [];
                  const hasError = lineAnnotations.some((a) => a.severity === "error");
                  const hasWarn = lineAnnotations.some((a) => a.severity === "warn");

                  return (
                    <div
                      key={i}
                      className={cn(
                        "h-[1.625rem]",
                        hasError && "bg-red-500/10",
                        !hasError && hasWarn && "bg-amber-500/10"
                      )}
                    />
                  );
                })}
              </div>

              {/* Highlighted code */}
              <div
                className={cn(
                  "relative p-4 text-sm overflow-x-auto",
                  "[&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0",
                  "[&>pre>code]:leading-relaxed"
                )}
                dangerouslySetInnerHTML={{ __html: highlightedHtml }}
              />
            </div>
          ) : (
            /* Plain text fallback */
            <div className="flex-1 relative">
              {/* Line backgrounds for annotations */}
              <div className="absolute inset-0 py-4">
                {lines.map((_, i) => {
                  const lineNum = i + 1;
                  const lineAnnotations = annotationsByLine.get(lineNum) || [];
                  const hasError = lineAnnotations.some((a) => a.severity === "error");
                  const hasWarn = lineAnnotations.some((a) => a.severity === "warn");

                  return (
                    <div
                      key={i}
                      className={cn(
                        "h-[1.625rem]",
                        hasError && "bg-red-500/10",
                        !hasError && hasWarn && "bg-amber-500/10"
                      )}
                    />
                  );
                })}
              </div>

              <pre className="relative p-4 text-sm text-slate-200 font-mono whitespace-pre overflow-x-auto leading-relaxed">
                {code}
              </pre>
            </div>
          )}
        </div>
      </div>

      {/* Tooltip */}
      {tooltip.visible && tooltip.annotation && (
        <div
          className="absolute z-50 max-w-xs p-3 rounded-lg bg-slate-800 border border-slate-600 shadow-lg"
          style={{
            left: tooltip.x,
            top: tooltip.y,
          }}
        >
          <div className="flex items-start gap-2">
            {tooltip.annotation.severity === "error" ? (
              <AlertCircle className="h-4 w-4 text-red-400 flex-shrink-0 mt-0.5" />
            ) : (
              <AlertTriangle className="h-4 w-4 text-amber-400 flex-shrink-0 mt-0.5" />
            )}
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span
                  className={cn(
                    "text-xs font-medium px-1.5 py-0.5 rounded",
                    tooltip.annotation.severity === "error"
                      ? "bg-red-500/20 text-red-300"
                      : "bg-amber-500/20 text-amber-300"
                  )}
                >
                  {tooltip.annotation.severity.toUpperCase()}
                </span>
                <code className="text-xs text-slate-400 font-mono truncate">
                  {tooltip.annotation.path}
                </code>
              </div>
              <p className="mt-1.5 text-sm text-slate-200">
                {tooltip.annotation.message}
              </p>
              {tooltip.annotation.hint && (
                <p className="mt-1 text-xs text-slate-400">
                  {tooltip.annotation.hint}
                </p>
              )}
              {tooltip.annotation.fixable && (
                <p className="mt-1.5 text-xs text-blue-400">
                  Click to auto-fix
                </p>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Error message */}
      {error && (
        <div className="px-4 py-2 bg-red-950/50 border-t border-red-500/30">
          <p className="text-xs text-red-400 font-mono">{error}</p>
        </div>
      )}
    </div>
  );
});
