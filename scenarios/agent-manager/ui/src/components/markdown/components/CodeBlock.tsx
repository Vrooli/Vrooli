import { memo, useEffect, useState } from "react";
import { Check, Copy } from "lucide-react";
import { useCodeCopy } from "../hooks/useCodeCopy";
import { detectLanguage, normalizeLanguage } from "../utils/languageDetection";

interface CodeBlockProps {
  /** The code content */
  code: string;
  /** Language hint from markdown fence (e.g., "typescript") */
  language?: string;
  /** Class name from react-markdown */
  className?: string;
}

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
        ],
      })
    );
  }
  return highlighterPromise;
}

export const CodeBlock = memo(function CodeBlock({
  code,
  language,
  className,
}: CodeBlockProps) {
  const safeCode = typeof code === "string" ? code : (code ? String(code) : "");

  const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null);
  const { status, copyCode } = useCodeCopy(safeCode);

  const extractedLang = className?.replace(/^language-/, "") || language;
  const normalizedLang = extractedLang
    ? normalizeLanguage(extractedLang)
    : detectLanguage(safeCode);

  const displayLang = normalizedLang === "text" ? "" : normalizedLang;

  useEffect(() => {
    let cancelled = false;

    async function highlight() {
      try {
        const highlighter = await getHighlighter();
        if (cancelled) return;

        const langs = highlighter.getLoadedLanguages();
        const langToUse = langs.includes(normalizedLang) ? normalizedLang : "text";

        const html = highlighter.codeToHtml(safeCode, {
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
  }, [safeCode, normalizedLang]);

  return (
    <div className="relative rounded-lg border border-border bg-card/50 overflow-hidden my-3">
      <div className="flex items-center justify-between px-4 py-2 bg-muted/60 border-b border-border">
        <span className="text-xs text-muted-foreground font-mono">
          {displayLang}
        </span>
        <button
          onClick={copyCode}
          className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
          aria-label={status === "copied" ? "Copied" : "Copy code"}
        >
          {status === "copied" ? (
            <>
              <Check className="h-3.5 w-3.5 text-success" />
              <span className="text-success">Copied</span>
            </>
          ) : status === "error" ? (
            <>
              <Copy className="h-3.5 w-3.5" />
              <span className="text-destructive">Copy failed</span>
            </>
          ) : (
            <>
              <Copy className="h-3.5 w-3.5" />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>

      <div className="bg-muted/40 overflow-x-auto">
        {highlightedHtml ? (
          <div
            className="p-4 text-sm [&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0"
            dangerouslySetInnerHTML={{ __html: highlightedHtml }}
          />
        ) : (
          <pre className="p-4 text-sm text-foreground font-mono whitespace-pre overflow-x-auto">
            {safeCode}
          </pre>
        )}
      </div>
    </div>
  );
});
