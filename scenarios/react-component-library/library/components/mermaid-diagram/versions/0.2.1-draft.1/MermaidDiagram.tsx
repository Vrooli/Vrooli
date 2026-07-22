/**
 * @libraryId react-component-library:mermaid-diagram
 * @displayName Mermaid Diagram
 * @description Mermaid diagram source preview with an expand seam.
 * @version 0.2.1-draft.1
 * @tags ["markdown","mermaid"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
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

/** @libraryId react-component-library:mermaid-diagram @version 0.2.0 */
export function MermaidDiagram({ code, onMermaidOpen, sourceLabel = "Source", diagramLabel = "Diagram", openLabel = "Open", copyLabel = "Copy" }: MermaidDiagramProps) {
  const [showSource, setShowSource] = useState(false);
  const { svg, error, loading } = useMermaidSvg(code);
  const { copied, copy } = useCodeCopy();
  return <section className="my-3 overflow-x-auto rounded border border-[var(--markdown-border,#334155)] bg-[var(--markdown-code-surface,#020617)]">
    <header className="flex items-center justify-between gap-2 border-b border-[var(--markdown-border,#334155)] px-3 py-2 text-xs text-[var(--markdown-muted,#94a3b8)]">
      <div className="flex gap-2"><button type="button" aria-pressed={!showSource} onClick={() => setShowSource(false)}>{diagramLabel}</button><button type="button" aria-pressed={showSource} onClick={() => setShowSource(true)}>{sourceLabel}</button></div>
      <div className="flex gap-2"><button type="button" onClick={() => void copy(code)}>{copied ? "Copied" : copyLabel}</button>{onMermaidOpen && <button type="button" onClick={() => onMermaidOpen(code)} className="text-[var(--markdown-link,#67e8f9)]">{openLabel}</button>}</div>
    </header>
    {showSource ? <pre className="p-3 text-xs text-[var(--markdown-code-text,#e2e8f0)]">{code}</pre> : error ? <><p role="alert" className="p-3 text-xs text-[var(--markdown-error,#fca5a5)]">{error}</p><pre className="p-3 text-xs text-[var(--markdown-code-text,#e2e8f0)]">{code}</pre></> : loading ? <p className="p-3 text-xs text-[var(--markdown-muted,#94a3b8)]">Rendering diagram…</p> : <div className="p-3 [&>svg]:max-w-full" dangerouslySetInnerHTML={{ __html: svg ?? "" }} />}
  </section>;
}