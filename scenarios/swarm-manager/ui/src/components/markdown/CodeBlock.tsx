/**
 * @vrooliComponentSource react-component-library:markdown-renderer
 * @vrooliComponentVersion 0.3.2
 * @vrooliComponentAdoption 612450da-7d3d-4888-85a9-e9ecf63254a6
 * @vrooliComponentAppliedAt 2026-07-21T21:01:34Z
 * @vrooliComponentSourceSha256 82d280ead27410cc48c2d0ab004a1688425ddd83af1f995841b240e1350cdbd2
 * @vrooliComponentDriftHash 82d280ead27410cc48c2d0ab004a1688425ddd83af1f995841b240e1350cdbd2
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useEffect, useState } from "react";
import { languageLabel, normalizeCodeLanguage } from "./languageDetection";
import { useCodeCopy } from "./useCodeCopy";

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
    if (highlightCache.has(key)) { setHtml(highlightCache.get(key)); return; }
    let cancelled = false;
    highlighter ??= import("shiki").then((shiki) => shiki.createHighlighter({
      themes: ["github-dark"],
      langs: ["typescript", "javascript", "python", "go", "json", "bash", "sql", "html", "css", "yaml", "markdown", "tsx", "jsx"],
    }));
    void highlighter.then((instance) => instance.codeToHtml(code, {
      lang: instance.getLoadedLanguages().includes(normalizedLanguage) ? normalizedLanguage : "text",
      theme: "github-dark",
    })).then((result) => {
      if (highlightCache.size >= HIGHLIGHT_CACHE_LIMIT) {
        const oldest = highlightCache.keys().next().value;
        if (oldest) highlightCache.delete(oldest);
      }
      highlightCache.set(key, result);
      if (!cancelled) setHtml(result);
    }).catch(() => { if (!cancelled) setHtml(undefined); });
    return () => { cancelled = true; };
  }, [code, key, normalizedLanguage]);

  return html;
}

export function CodeBlock({ code, language, className, copyLabel = "Copy", copiedLabel = "Copied" }: CodeBlockProps) {
  const html = useHighlightedCode(code, language ?? "text");
  const { copied, copy } = useCodeCopy();
  return <section className={`my-3 overflow-hidden rounded border border-[var(--markdown-border)] bg-[var(--markdown-code-surface)] ${className ?? ""}`}>
    <header className="flex items-center justify-between border-b border-[var(--markdown-border)] px-3 py-2 text-xs text-[var(--markdown-muted)]">
      <span>{languageLabel(language)}</span>
      <button type="button" onClick={() => void copy(code)} className="rounded px-1 text-[var(--markdown-link)] hover:opacity-80">{copied ? copiedLabel : copyLabel}</button>
    </header>
    {html ? <div className="overflow-x-auto p-3 text-sm [&>pre]:m-0 [&>pre]:bg-transparent [&>pre]:p-0" dangerouslySetInnerHTML={{ __html: html }} /> : <pre className="overflow-x-auto p-3 text-sm text-[var(--markdown-code-text)]"><code>{code}</code></pre>}
  </section>;
}