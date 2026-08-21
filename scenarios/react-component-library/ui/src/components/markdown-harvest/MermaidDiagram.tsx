/**
 * @libraryId react-component-library:mermaid-diagram
 * @displayName Mermaid Diagram
 * @description Strict Mermaid SVG renderer with source, copy, error, and open seams.
 * @version 0.3.0
 * @tags ["markdown","mermaid","diagram"]
 * @deps {"react":"^18","mermaid":"^11.4.0"}
 */

import { useState } from "react";
import { useCodeCopy } from "./useCodeCopy";
import { useMermaidSvg } from "./useMermaidSvg";

export interface MermaidDiagramProps {
  code: string;
  onMermaidOpen?: (code: string) => void;
  sourceLabel?: string;
  diagramLabel?: string;
  openLabel?: string;
  copyLabel?: string;
}

/**
 * Toolbar affordances carried their own (absent) styling: no hover, no
 * :focus-visible ring, no pressed treatment. This asset is a self-contained
 * harvest — see the @deps header — so it cannot import the library Button; the
 * treatment is instead expressed entirely in shared design tokens.
 */
function toolbarButtonClass(pressed?: boolean) {
  return [
    "touch-target rounded px-space-3xs transition-colors motion-reduce:transition-none",
    "focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-app-focus",
    pressed ? "bg-[var(--markdown-border)] text-[var(--markdown-link)]" : "hover:opacity-80",
  ].join(" ");
}

export function MermaidDiagram({
  code,
  onMermaidOpen,
  sourceLabel = "Source",
  diagramLabel = "Diagram",
  openLabel = "Open",
  copyLabel = "Copy",
}: MermaidDiagramProps) {
  const [showSource, setShowSource] = useState(false);
  const { svg, error, loading } = useMermaidSvg(code);
  const { copied, copy } = useCodeCopy();
  return (
    <section className="my-space-xs overflow-x-auto rounded border border-[var(--markdown-border)] bg-[var(--markdown-code-surface)]">
      <header className="flex items-center justify-between gap-space-2xs border-b border-[var(--markdown-border)] px-space-xs py-space-2xs text-xs text-[var(--markdown-muted)]">
        <div className="flex gap-space-2xs">
          <button
            type="button"
            aria-pressed={!showSource}
            onClick={() => setShowSource(false)}
            className={toolbarButtonClass(!showSource)}
          >
            {diagramLabel}
          </button>
          <button
            type="button"
            aria-pressed={showSource}
            onClick={() => setShowSource(true)}
            className={toolbarButtonClass(showSource)}
          >
            {sourceLabel}
          </button>
        </div>
        <div className="flex gap-space-2xs">
          <button type="button" onClick={() => void copy(code)} className={toolbarButtonClass()}>
            {copied ? "Copied" : copyLabel}
          </button>
          {onMermaidOpen && (
            <button
              type="button"
              onClick={() => onMermaidOpen(code)}
              className={`${toolbarButtonClass()} text-[var(--markdown-link)]`}
            >
              {openLabel}
            </button>
          )}
        </div>
      </header>
      {showSource ? (
        <pre className="p-space-xs text-xs text-[var(--markdown-code-text)]">{code}</pre>
      ) : error ? (
        <>
          <p role="alert" className="p-space-xs text-xs text-[var(--markdown-error)]">
            {error}
          </p>
          <pre className="p-space-xs text-xs text-[var(--markdown-code-text)]">{code}</pre>
        </>
      ) : loading ? (
        <p className="p-space-xs text-xs text-[var(--markdown-muted)]">Rendering diagram…</p>
      ) : (
        <div
          className="p-space-xs [&>svg]:max-w-full"
          dangerouslySetInnerHTML={{ __html: svg ?? "" }}
        />
      )}
    </section>
  );
}
