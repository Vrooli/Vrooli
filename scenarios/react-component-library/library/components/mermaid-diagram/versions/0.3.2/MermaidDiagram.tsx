/**
 * @libraryId react-component-library:mermaid-diagram
 * @displayName Mermaid Diagram
 * @description Mermaid diagram source preview with an expand seam.
 * @version 0.3.2
 * @tags ["markdown","mermaid"]
 * @deps {"react":"^18","mermaid":"^11.4.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import { useState } from "react";
import { useCodeCopy } from "./useCodeCopy";
import { useMermaidSvg } from "./useMermaidSvg";
import { mermaidStyles } from "./styles";

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
