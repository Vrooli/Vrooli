import { useEffect, useId, useState } from "react";

interface MermaidViewerProps {
  code: string | null;
  onClose: () => void;
}

/** Full-screen Mermaid renderer for authored plan diagrams. */
export function MermaidViewer({ code, onClose }: MermaidViewerProps) {
  const instanceID = useId().replace(/:/g, "-");
  const [svg, setSvg] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!code) { setSvg(""); setError(""); return; }
    let active = true;
    setSvg(""); setError("");
    void import("mermaid").then(async ({ default: mermaid }) => {
      mermaid.initialize({ startOnLoad: false, securityLevel: "strict", theme: "dark" });
      const rendered = await mermaid.render(`plan-manager-${instanceID}`, code);
      if (active) setSvg(rendered.svg);
    }).catch((reason: unknown) => {
      if (active) setError(reason instanceof Error ? reason.message : "Unable to render Mermaid diagram.");
    });
    return () => { active = false; };
  }, [code, instanceID]);

  if (!code) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4" role="dialog" aria-modal="true" aria-label="Mermaid diagram">
      <section className="flex max-h-full w-full max-w-6xl flex-col rounded-panel border border-app-border bg-app-surface shadow-xl">
        <div className="flex items-center justify-between border-b border-app-border px-4 py-3">
          <h2 className="text-base font-semibold text-app-foreground">Mermaid diagram</h2>
          <button type="button" onClick={onClose} className="rounded-control border border-app-border px-3 py-1 text-sm text-app-foreground">Close</button>
        </div>
        <div className="min-h-0 overflow-auto p-4">
          {error ? <div className="rounded-control border border-app-warning bg-app-warning/10 p-3 text-sm text-app-warning"><p>{error}</p><pre className="mt-3 overflow-auto whitespace-pre-wrap font-mono text-xs text-app-foreground">{code}</pre></div> : svg ? <div data-testid="mermaid-viewer-svg" className="min-w-max" dangerouslySetInnerHTML={{ __html: svg }} /> : <p className="text-sm text-app-muted-foreground">Rendering diagram…</p>}
        </div>
      </section>
    </div>
  );
}
