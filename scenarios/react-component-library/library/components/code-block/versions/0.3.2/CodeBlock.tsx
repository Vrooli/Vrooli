/**
 * @libraryId react-component-library:code-block
 * @displayName Code Block
 * @description Code block with language label and copy feedback.
 * @version 0.3.2
 * @tags ["markdown","code"]
 * @deps {"react":"^18","shiki":"^4.3.1"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useEffect, useState } from "react";
import { languageLabel, normalizeCodeLanguage } from "./languageDetection";
import { useCodeCopy } from "./useCodeCopy";
import { codeBlockStyles } from "./styles";

const HIGHLIGHT_CACHE_LIMIT = 400;
const highlightCache = new Map<string, string>();
let highlighter: Promise<import("shiki").Highlighter> | undefined;

export interface CodeBlockProps {
  code: string;
  language?: string;
  className?: string;
  copyLabel?: string;
  copiedLabel?: string;
}

function useHighlightedCode(code: string, language: string) {
  const normalizedLanguage = normalizeCodeLanguage(language);
  const key = `${normalizedLanguage}\u0000${code}`;
  const [html, setHtml] = useState(() => highlightCache.get(key));

  useEffect(() => {
    if (highlightCache.has(key)) {
      setHtml(highlightCache.get(key));
      return;
    }
    let cancelled = false;
    highlighter ??= import("shiki").then((shiki) =>
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
          "tsx",
          "jsx",
        ],
      }),
    );
    void highlighter
      .then((instance) =>
        instance.codeToHtml(code, {
          lang: instance.getLoadedLanguages().includes(normalizedLanguage)
            ? normalizedLanguage
            : "text",
          theme: "github-dark",
        }),
      )
      .then((result) => {
        if (highlightCache.size >= HIGHLIGHT_CACHE_LIMIT) {
          const oldest = highlightCache.keys().next().value;
          if (oldest) highlightCache.delete(oldest);
        }
        highlightCache.set(key, result);
        if (!cancelled) setHtml(result);
      })
      .catch(() => {
        if (!cancelled) setHtml(undefined);
      });
    return () => {
      cancelled = true;
    };
  }, [code, key, normalizedLanguage]);

  return html;
}

export function CodeBlock({
  code,
  language,
  className,
  copyLabel = "Copy",
  copiedLabel = "Copied",
}: CodeBlockProps) {
  const html = useHighlightedCode(code, language ?? "text");
  const { copied, copy } = useCodeCopy();
  return (
    <section className={`rcl-code-block ${className ?? ""}`} data-rcl-code-block>
      <style data-rcl-code-block-styles dangerouslySetInnerHTML={{ __html: codeBlockStyles }} />
      <header className="rcl-code-block__header">
        <span>{languageLabel(language)}</span>
        <button
          data-testid="data-display.code-block"
          type="button"
          onClick={() => void copy(code)}
          className="rcl-code-block__copy"
        >
          {copied ? copiedLabel : copyLabel}
        </button>
      </header>
      {html ? (
        <div className="rcl-code-block__body" dangerouslySetInnerHTML={{ __html: html }} />
      ) : (
        <pre className="rcl-code-block__body">
          <code>{code}</code>
        </pre>
      )}
    </section>
  );
}
