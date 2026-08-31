/**
 * @libraryId react-component-library:mermaid-diagram
 * @displayName Mermaid Diagram
 * @version 0.3.3
 * @tags ["markdown","mermaid"]
 * @deps {"react":"^18","mermaid":"^11.4.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useState } from "react";
import { useCodeCopy } from "../../../../support/mermaid-diagram/0.3.3/useCodeCopy";
import { useMermaidSvg } from "../../../../support/mermaid-diagram/0.3.3/useMermaidSvg";
export const mermaidStyles = `
[data-rcl-mermaid] { min-inline-size: 0; overflow: hidden; margin-block: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-mermaid] .rcl-mermaid__header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-xs) var(--space-sm); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-mermaid] .rcl-mermaid__tabs, [data-rcl-mermaid] .rcl-mermaid__actions { display: flex; align-items: center; gap: var(--space-xs); }
[data-rcl-mermaid] button { border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-accent); padding: var(--space-3xs) var(--space-2xs); font: var(--text-label); cursor: pointer; }
[data-rcl-mermaid] button:hover, [data-rcl-mermaid] button[aria-pressed="true"] { background: color-mix(in srgb, var(--color-accent) 10%, transparent); color: var(--color-foreground); }
[data-rcl-mermaid] .rcl-mermaid__body { overflow-x: auto; padding: var(--space-sm); color: var(--color-foreground); font: var(--text-body); }
[data-rcl-mermaid] .rcl-mermaid__error { padding: var(--space-sm); color: var(--color-danger); font: var(--text-caption); }
[data-rcl-mermaid] .rcl-mermaid__body > svg { display: block; max-inline-size: 100%; block-size: auto; margin-inline: auto; }
`;
export interface MermaidDiagramProps {
  code: string;
  onMermaidOpen?: (code: string) => void;
  sourceLabel?: string;
  diagramLabel?: string;
  openLabel?: string;
  copyLabel?: string;
}

export const MermaidDiagram = withClassName(function MermaidDiagram({
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
    <section className="rcl-mermaid" data-rcl-mermaid>
      <StyleSheet name="mermaid-diagram-0-3-2" css={mermaidStyles} />
      <header className="rcl-mermaid__header">
        <div className="rcl-mermaid__tabs">
          <button type="button" aria-pressed={!showSource} onClick={() => setShowSource(false)}>
            {diagramLabel}
          </button>
          <button type="button" aria-pressed={showSource} onClick={() => setShowSource(true)}>
            {sourceLabel}
          </button>
        </div>
        <div className="rcl-mermaid__actions">
          <button type="button" onClick={() => void copy(code)}>
            {copied ? "Copied" : copyLabel}
          </button>
          {onMermaidOpen && (
            <button type="button" onClick={() => onMermaidOpen(code)} className="rcl-mermaid__open">
              {openLabel}
            </button>
          )}
        </div>
      </header>
      {showSource ? (
        <pre className="rcl-mermaid__body">{code}</pre>
      ) : error ? (
        <>
          <p role="alert" className="rcl-mermaid__error">
            {error}
          </p>
          <pre className="rcl-mermaid__body">{code}</pre>
        </>
      ) : loading ? (
        <p className="rcl-mermaid__body">Rendering diagram…</p>
      ) : (
        <div className="rcl-mermaid__body" dangerouslySetInnerHTML={{ __html: svg ?? "" }} />
      )}
    </section>
  );
});
