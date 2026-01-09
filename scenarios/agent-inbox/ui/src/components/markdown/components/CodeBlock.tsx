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
        ],
      })
    );
  }
  return highlighterPromise;
}

/**
 * Syntax-highlighted code block with language label and copy button.
 */
export const CodeBlock = memo(function CodeBlock({
  code,
  language,
  className,
}: CodeBlockProps) {
  // Defensive: ensure code is a string to prevent crashes
  const safeCode = typeof code === "string" ? code : (code ? String(code) : "");

  const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null);
  const { copied, copyCode } = useCodeCopy(safeCode);

  // Extract language from className if not provided directly
  // react-markdown passes language as className="language-typescript"
  const extractedLang = className?.replace(/^language-/, "") || language;
  const normalizedLang = extractedLang
    ? normalizeLanguage(extractedLang)
    : detectLanguage(safeCode);

  // Display name for the language label
  const displayLang = normalizedLang === "text" ? "" : normalizedLang;

  useEffect(() => {
    let cancelled = false;

    async function highlight() {
      try {
        const highlighter = await getHighlighter();
        if (cancelled) return;

        // Check if the language is supported
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
        // Fall back to plain text
      }
    }

    highlight();

    return () => {
      cancelled = true;
    };
  }, [safeCode, normalizedLang]);

  return (
    <div className="relative group rounded-lg overflow-hidden my-3">
      {/* Header with language label and copy button */}
      <div className="flex items-center justify-between px-4 py-2 bg-slate-900 border-b border-slate-700">
        <span className="text-xs text-slate-400 font-mono">{displayLang}</span>
        <button
          onClick={copyCode}
          className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 transition-colors"
          aria-label={copied ? "Copied" : "Copy code"}
        >
          {copied ? (
            <>
              <Check className="h-3.5 w-3.5 text-green-400" />
              <span className="text-green-400">Copied</span>
            </>
          ) : (
            <>
              <Copy className="h-3.5 w-3.5" />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>

      {/* Code content */}
      <div className="bg-slate-800 overflow-x-auto">
        {highlightedHtml ? (
          <div
            className="p-4 text-sm [&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0"
            dangerouslySetInnerHTML={{ __html: highlightedHtml }}
          />
        ) : (
          <pre className="p-4 text-sm text-slate-200 font-mono whitespace-pre overflow-x-auto">
            {safeCode}
          </pre>
        )}
      </div>
    </div>
  );
});
