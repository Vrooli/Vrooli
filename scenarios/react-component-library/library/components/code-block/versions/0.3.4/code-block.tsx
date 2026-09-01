/**
 * @libraryId react-component-library:code-block
 * @displayName Code Block
 * @version 0.3.4
 * @tags ["markdown","code"]
 * @deps {"react":"^18","shiki":"^4.3.1"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useEffect, useState } from "react";
import { languageLabel, normalizeCodeLanguage } from "../../../../support/code-block/versions/0.3.4/languageDetection";
import { useCodeCopy } from "../../../../support/code-block/versions/0.3.4/useCodeCopy";
export const codeBlockStyles = `
[data-rcl-code-block] { min-inline-size: 0; overflow: hidden; margin-block: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-code-block] .rcl-code-block__header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-xs) var(--space-sm); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-code-block] button { border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-accent); padding: var(--space-3xs) var(--space-2xs); font: var(--text-label); cursor: pointer; }
[data-rcl-code-block] button:hover { background: color-mix(in srgb, var(--color-accent) 10%, transparent); }
[data-rcl-code-block] .rcl-code-block__body { overflow-x: auto; padding: var(--space-sm); color: var(--color-foreground); font: var(--text-body); }
[data-rcl-code-block] .rcl-code-block__body > pre, [data-rcl-code-block] .rcl-code-block__body pre { margin: 0; background: transparent; padding: 0; }
`;
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
    <section className={`rcl-code-block ${className ?? ""}`} data-rcl-code-block>
      <StyleSheet name="code-block-0-3-3" css={codeBlockStyles} />
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
});
