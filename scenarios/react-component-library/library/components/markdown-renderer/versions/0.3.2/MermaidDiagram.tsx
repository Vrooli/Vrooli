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
import { markdownStyles } from "./markdownStyles";

export interface MermaidDiagramProps {
  code: string;
  onMermaidOpen?: (code: string) => void;
  sourceLabel?: string;
  diagramLabel?: string;
  openLabel?: string;
  copyLabel?: string;
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
    <section className="rcl-md__diagram" data-rcl-markdown>
      <style
        data-rcl-markdown-styles
        dangerouslySetInnerHTML={{ __html: markdownStyles }}
      />
      <header className="rcl-md__diagram-header">
        <div className="rcl-md__diagram-tabs">
          <button
            type="button"
            aria-pressed={!showSource}
            onClick={() => setShowSource(false)}
          >
            {diagramLabel}
          </button>
          <button
            type="button"
            aria-pressed={showSource}
            onClick={() => setShowSource(true)}
          >
            {sourceLabel}
          </button>
        </div>
        <div className="rcl-md__diagram-actions">
          <button type="button" onClick={() => void copy(code)}>
            {copied ? "Copied" : copyLabel}
          </button>
          {onMermaidOpen && (
            <button
              type="button"
              onClick={() => onMermaidOpen(code)}
              className="rcl-md__link"
            >
              {openLabel}
            </button>
          )}
        </div>
      </header>
      {showSource ? (
        <pre className="rcl-md__diagram-body">{code}</pre>
      ) : error ? (
        <>
          <p role="alert" className="rcl-md__error">
            {error}
          </p>
          <pre className="rcl-md__diagram-body">{code}</pre>
        </>
      ) : loading ? (
        <p className="rcl-md__diagram-body">Rendering diagram…</p>
      ) : (
        <div
          className="rcl-md__diagram-body"
          dangerouslySetInnerHTML={{ __html: svg ?? "" }}
        />
      )}
    </section>
  );
}
