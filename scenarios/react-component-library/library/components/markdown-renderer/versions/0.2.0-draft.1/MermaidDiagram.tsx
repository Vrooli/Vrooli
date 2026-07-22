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
  return <section className="my-3 overflow-x-auto rounded border border-slate-700 bg-slate-950">
    <header className="flex items-center justify-between gap-2 border-b border-slate-700 px-3 py-2 text-xs text-slate-400">
      <div className="flex gap-2"><button type="button" aria-pressed={!showSource} onClick={() => setShowSource(false)}>{diagramLabel}</button><button type="button" aria-pressed={showSource} onClick={() => setShowSource(true)}>{sourceLabel}</button></div>
      <div className="flex gap-2"><button type="button" onClick={() => void copy(code)}>{copied ? "Copied" : copyLabel}</button>{onMermaidOpen && <button type="button" onClick={() => onMermaidOpen(code)} className="text-cyan-300">{openLabel}</button>}</div>
    </header>
    {showSource ? <pre className="p-3 text-xs text-slate-200">{code}</pre> : error ? <><p role="alert" className="p-3 text-xs text-red-300">{error}</p><pre className="p-3 text-xs text-slate-200">{code}</pre></> : loading ? <p className="p-3 text-xs text-slate-400">Rendering diagram…</p> : <div className="p-3 [&>svg]:max-w-full" dangerouslySetInnerHTML={{ __html: svg ?? "" }} />}
  </section>;
}