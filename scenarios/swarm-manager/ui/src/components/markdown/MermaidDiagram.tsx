/**
 * @vrooliComponentSource react-component-library:markdown-renderer
 * @vrooliComponentVersion 0.3.2
 * @vrooliComponentAdoption 612450da-7d3d-4888-85a9-e9ecf63254a6
 * @vrooliComponentAppliedAt 2026-07-21T21:01:34Z
 * @vrooliComponentSourceSha256 730c67e23fac64f0743f66d5024e869ff6bbd958b641f55b6142c0a26342dfc9
 * @vrooliComponentDriftHash 730c67e23fac64f0743f66d5024e869ff6bbd958b641f55b6142c0a26342dfc9
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
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

// LOCAL EDIT (drift from markdown-renderer 0.3.2): the library ships the
// toolbar controls as unstyled <button> elements, which render as bare
// underlined-looking text with no affordance and no pressed state. These are
// the same tokens the rest of the markdown surface uses.
const TOOLBAR_BUTTON =
  "rounded px-1.5 py-0.5 transition-colors hover:bg-white/5 hover:text-[var(--markdown-text)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--markdown-link)]";
const TOOLBAR_BUTTON_ACTIVE = "bg-white/10 text-[var(--markdown-text)]";

export function MermaidDiagram({ code, onMermaidOpen, sourceLabel = "Source", diagramLabel = "Diagram", openLabel = "Open", copyLabel = "Copy" }: MermaidDiagramProps) {
  const [showSource, setShowSource] = useState(false);
  const { svg, error, loading } = useMermaidSvg(code);
  const { copied, copy } = useCodeCopy();
  return <section className="my-3 overflow-x-auto rounded border border-[var(--markdown-border)] bg-[var(--markdown-code-surface)]">
    <header className="flex items-center justify-between gap-2 border-b border-[var(--markdown-border)] px-3 py-2 text-xs text-[var(--markdown-muted)]">
      <div className="flex gap-1">
        <button type="button" aria-pressed={!showSource} onClick={() => setShowSource(false)} className={`${TOOLBAR_BUTTON} ${!showSource ? TOOLBAR_BUTTON_ACTIVE : ""}`}>{diagramLabel}</button>
        <button type="button" aria-pressed={showSource} onClick={() => setShowSource(true)} className={`${TOOLBAR_BUTTON} ${showSource ? TOOLBAR_BUTTON_ACTIVE : ""}`}>{sourceLabel}</button>
      </div>
      <div className="flex gap-1">
        <button type="button" onClick={() => void copy(code)} className={TOOLBAR_BUTTON}>{copied ? "Copied" : copyLabel}</button>
        {onMermaidOpen && <button type="button" onClick={() => onMermaidOpen(code)} className={`${TOOLBAR_BUTTON} text-[var(--markdown-link)]`}>{openLabel}</button>}
      </div>
    </header>
    {showSource ? <pre className="p-3 text-xs text-[var(--markdown-code-text)]">{code}</pre> : error ? <><p role="alert" className="p-3 text-xs text-[var(--markdown-error)]">{error}</p><pre className="p-3 text-xs text-[var(--markdown-code-text)]">{code}</pre></> : loading ? <p className="p-3 text-xs text-[var(--markdown-muted)]">Rendering diagram…</p> : <div className="p-3 [&>svg]:max-w-full" dangerouslySetInnerHTML={{ __html: svg ?? "" }} />}
  </section>;
}