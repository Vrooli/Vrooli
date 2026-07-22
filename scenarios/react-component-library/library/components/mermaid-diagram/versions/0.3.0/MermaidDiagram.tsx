/**
 * @libraryId react-component-library:mermaid-diagram
 * @displayName Mermaid Diagram
 * @description Mermaid diagram source preview with an expand seam.
 * @version 0.3.0
 * @tags ["markdown","mermaid"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */


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

export function MermaidDiagram({ code, onMermaidOpen, sourceLabel = "Source", diagramLabel = "Diagram", openLabel = "Open", copyLabel = "Copy" }: MermaidDiagramProps) {
  const [showSource, setShowSource] = useState(false);
  const { svg, error, loading } = useMermaidSvg(code);
  const { copied, copy } = useCodeCopy();
  return <section className="my-3 overflow-x-auto rounded border border-[var(--markdown-border)] bg-[var(--markdown-code-surface)]">
    <header className="flex items-center justify-between gap-2 border-b border-[var(--markdown-border)] px-3 py-2 text-xs text-[var(--markdown-muted)]">
      <div className="flex gap-2"><button type="button" aria-pressed={!showSource} onClick={() => setShowSource(false)}>{diagramLabel}</button><button type="button" aria-pressed={showSource} onClick={() => setShowSource(true)}>{sourceLabel}</button></div>
      <div className="flex gap-2"><button type="button" onClick={() => void copy(code)}>{copied ? "Copied" : copyLabel}</button>{onMermaidOpen && <button type="button" onClick={() => onMermaidOpen(code)} className="text-[var(--markdown-link)]">{openLabel}</button>}</div>
    </header>
    {showSource ? <pre className="p-3 text-xs text-[var(--markdown-code-text)]">{code}</pre> : error ? <><p role="alert" className="p-3 text-xs text-[var(--markdown-error)]">{error}</p><pre className="p-3 text-xs text-[var(--markdown-code-text)]">{code}</pre></> : loading ? <p className="p-3 text-xs text-[var(--markdown-muted)]">Rendering diagram…</p> : <div className="p-3 [&>svg]:max-w-full" dangerouslySetInnerHTML={{ __html: svg ?? "" }} />}
  </section>;
}