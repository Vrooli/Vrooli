import { memo, useEffect, useState } from "react";
import { Check, Copy } from "lucide-react";
import { cn } from "../../lib/utils";
import {
  detectLanguage,
  getLanguageFromFilename,
  normalizeLanguage,
} from "../../lib/languageDetection";

interface CodeBlockProps {
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
  /** Additional CSS classes */
  className?: string;
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
          "rust",
          "java",
          "c",
          "cpp",
          "ruby",
          "php",
          "swift",
          "kotlin",
          "toml",
          "dockerfile",
          "makefile",
          "xml",
          "graphql",
          "protobuf",
          "lua",
          "ini",
          "nginx",
          "terraform",
          "scss",
          "less",
          "r",
          "vim",
          "fish",
          "powershell",
        ],
      })
    );
  }
  return highlighterPromise;
}

/**
 * Syntax-highlighted code block with language detection, line numbers, and copy functionality.
 * Uses shiki for accurate, theme-aware syntax highlighting.
 */
export const CodeBlock = memo(function CodeBlock({
  code,
  filename,
  language,
  maxHeight = "400px",
  showLineNumbers = true,
  showHeader = true,
  className,
}: CodeBlockProps) {
  const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

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
        // Fall back to plain text display
      }
    }

    highlight();

    return () => {
      cancelled = true;
    };
  }, [code, detectedLang]);

  const lines = code.split("\n");
  const lineNumberWidth = String(lines.length).length;

  return (
    <div
      className={cn(
        "relative group rounded-lg overflow-hidden border border-slate-700",
        className
      )}
    >
      {/* Header with language label and copy button */}
      {showHeader && (
        <div className="flex items-center justify-between px-4 py-2 bg-slate-900 border-b border-slate-700">
          <span className="text-xs text-slate-400 font-mono uppercase">
            {displayLang}
          </span>
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
        className="bg-slate-950 overflow-auto"
        style={{ maxHeight }}
      >
        {highlightedHtml ? (
          <div className="flex">
            {/* Line numbers */}
            {showLineNumbers && (
              <div className="flex-shrink-0 py-4 pl-4 pr-2 select-none border-r border-slate-800">
                <div className="text-sm font-mono text-slate-600 text-right leading-relaxed">
                  {lines.map((_, i) => (
                    <div
                      key={i}
                      style={{ minWidth: `${lineNumberWidth}ch` }}
                    >
                      {i + 1}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Highlighted code */}
            <div
              className={cn(
                "flex-1 p-4 text-sm overflow-x-auto",
                "[&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0",
                "[&>pre>code]:leading-relaxed"
              )}
              dangerouslySetInnerHTML={{ __html: highlightedHtml }}
            />
          </div>
        ) : (
          <div className="flex">
            {/* Line numbers (fallback) */}
            {showLineNumbers && (
              <div className="flex-shrink-0 py-4 pl-4 pr-2 select-none border-r border-slate-800">
                <div className="text-sm font-mono text-slate-600 text-right leading-relaxed">
                  {lines.map((_, i) => (
                    <div
                      key={i}
                      style={{ minWidth: `${lineNumberWidth}ch` }}
                    >
                      {i + 1}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Plain text fallback */}
            <pre className="flex-1 p-4 text-sm text-slate-200 font-mono whitespace-pre overflow-x-auto leading-relaxed">
              {code}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
});
