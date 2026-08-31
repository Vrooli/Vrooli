/**
 * @libraryId react-component-library:code-block
 * @displayName Code Block
 * @description Syntax-highlighted fenced code block with bounded cache and copy feedback.
 * @version 0.4.0
 * @tags ["markdown","code","shiki"]
 * @deps {"react":"^18","shiki":"^4.3.1"}
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useEffect, useState } from "react";
import { languageLabel, normalizeCodeLanguage } from "./languageDetection";
import { useCodeCopy } from "./useCodeCopy";
import { markdownStyles } from "./markdownStyles";

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

export const CodeBlock = withClassName(function CodeBlock({
  code,
  language,
  className,
  copyLabel = "Copy",
  copiedLabel = "Copied",
}: CodeBlockProps) {
  const html = useHighlightedCode(code, language ?? "text");
  const { copied, copy } = useCodeCopy();
  return (
    <section className={`rcl-md__code ${className ?? ""}`} data-rcl-markdown>
      <StyleSheet name="markdown-renderer-0-4-0" css={markdownStyles} />
      <header className="rcl-md__code-header">
        <span>{languageLabel(language)}</span>
        <button type="button" onClick={() => void copy(code)} className="rcl-md__copy-button">
          {copied ? copiedLabel : copyLabel}
        </button>
      </header>
      {html ? (
        <div className="rcl-md__code-body" dangerouslySetInnerHTML={{ __html: html }} />
      ) : (
        <pre className="rcl-md__code-body">
          <code>{code}</code>
        </pre>
      )}
    </section>
  );
});
