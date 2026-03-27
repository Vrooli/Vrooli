import { memo, useEffect, useState } from "react";
import { Check, Copy } from "lucide-react";
import { useCodeCopy } from "../hooks/useCodeCopy";
import { detectLanguage, normalizeLanguage } from "../utils/languageDetection";

interface CodeBlockProps {
  code: string;
  language?: string;
  className?: string;
}

// Lazy-loaded shiki highlighter (singleton)
let highlighterPromise: Promise<import("shiki").Highlighter> | null = null;

function getHighlighter() {
  if (!highlighterPromise) {
    highlighterPromise = import("shiki").then((shiki) =>
      shiki.createHighlighter({
        themes: ["github-dark"],
        langs: [
          "typescript", "javascript", "python", "go", "json", "bash",
          "sql", "html", "css", "yaml", "markdown", "jsx", "tsx",
          "rust", "java", "c", "cpp", "ruby", "php", "swift", "kotlin",
        ],
      }),
    );
  }
  return highlighterPromise;
}

/** Syntax-highlighted code block with language label and copy button. */
export const CodeBlock = memo(function CodeBlock({ code, language, className }: CodeBlockProps) {
  const safeCode = typeof code === "string" ? code : (code ? String(code) : "");
  const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null);
  const { copied, copyCode } = useCodeCopy(safeCode);

  const extractedLang = className?.replace(/^language-/, "") || language;
  const normalizedLang = extractedLang ? normalizeLanguage(extractedLang) : detectLanguage(safeCode);
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

        if (!cancelled) setHighlightedHtml(html);
      } catch (err) {
        console.warn("Syntax highlighting failed:", err);
      }
    }

    highlight();
    return () => { cancelled = true; };
  }, [safeCode, normalizedLang]);

  return (
    <div className="relative group rounded-lg overflow-hidden my-3 max-w-full">
      <div className="flex items-center justify-between px-4 py-2 bg-wc-surface-base border-b border-wc-default">
        <span className="text-xs text-wc-text-muted font-mono">{displayLang}</span>
        <button
          onClick={copyCode}
          className="flex items-center gap-1.5 text-xs text-wc-text-muted hover:text-wc-text-primary transition-colors"
          aria-label={copied ? "Copied" : "Copy code"}
          type="button"
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
      <div className="bg-wc-surface overflow-x-auto max-w-full">
        {highlightedHtml ? (
          <div
            className="max-w-full overflow-x-auto p-4 text-sm [&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0 [&>pre]:!max-w-full [&>pre]:!overflow-x-auto"
            dangerouslySetInnerHTML={{ __html: highlightedHtml }}
          />
        ) : (
          <pre className="max-w-full p-4 text-sm text-wc-text-primary font-mono whitespace-pre overflow-x-auto">
            {safeCode}
          </pre>
        )}
      </div>
    </div>
  );
});
