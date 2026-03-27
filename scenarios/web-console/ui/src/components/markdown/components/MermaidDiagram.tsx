import { memo, useEffect, useRef, useState } from "react";
import { Check, Code, Copy, Eye, Loader2 } from "lucide-react";
import { useCodeCopy } from "../hooks/useCodeCopy";

interface MermaidDiagramProps {
  code: string;
}

// Lazy-loaded mermaid instance (singleton)
let mermaidPromise: Promise<typeof import("mermaid")["default"]> | null = null;

function getMermaid() {
  if (!mermaidPromise) {
    mermaidPromise = import("mermaid")
      .then((mod) => {
        const mermaid = mod.default;
        mermaid.initialize({
          startOnLoad: false,
          theme: "dark",
          securityLevel: "strict",
          fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace",
          themeVariables: {
            primaryColor: "#334155",
            primaryTextColor: "#e2e8f0",
            primaryBorderColor: "#475569",
            lineColor: "#6366f1",
            secondaryColor: "#1e293b",
            tertiaryColor: "#0f172a",
            noteBkgColor: "#1e293b",
            noteTextColor: "#e2e8f0",
            noteBorderColor: "#475569",
          },
        });
        return mermaid;
      })
      .catch((err) => {
        mermaidPromise = null;
        throw err;
      });
  }
  return mermaidPromise;
}

/** Renders a mermaid diagram from source code with source/diagram toggle. */
export const MermaidDiagram = memo(function MermaidDiagram({ code }: MermaidDiagramProps) {
  const [svgHtml, setSvgHtml] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showSource, setShowSource] = useState(false);
  const { copied, copyCode } = useCodeCopy(code);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);

    if (debounceRef.current) clearTimeout(debounceRef.current);

    debounceRef.current = setTimeout(() => {
      async function render() {
        try {
          const mermaid = await getMermaid();
          if (cancelled) return;

          const id = `mermaid-${crypto.randomUUID()}`;
          const { svg } = await mermaid.render(id, code);
          if (cancelled) return;

          setSvgHtml(svg);
          setError(null);
        } catch (err) {
          if (cancelled) return;
          setError(err instanceof Error ? err.message : "Failed to render diagram");
          setSvgHtml(null);
        }
      }
      void render();
    }, 100);

    return () => {
      cancelled = true;
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [code]);

  if (!code.trim()) {
    return (
      <div className="relative group rounded-lg overflow-hidden my-3">
        <div className="flex items-center justify-between px-4 py-2 bg-wc-surface-base border-b border-wc-default">
          <span className="text-xs text-wc-text-muted font-mono">mermaid</span>
        </div>
        <div className="bg-wc-surface p-8 flex items-center justify-center">
          <span className="text-sm text-wc-text-faint italic">Empty diagram</span>
        </div>
      </div>
    );
  }

  return (
    <div className="relative group rounded-lg overflow-hidden my-3">
      <div className="flex items-center justify-between px-4 py-2 bg-wc-surface-base border-b border-wc-default">
        <span className="text-xs text-wc-text-muted font-mono">mermaid</span>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowSource((prev) => !prev)}
            className="flex items-center gap-1.5 text-xs text-wc-text-muted hover:text-wc-text-primary transition-colors"
            aria-label={showSource ? "Show diagram" : "Show source"}
            type="button"
          >
            {showSource ? (
              <><Eye className="h-3.5 w-3.5" /><span>Diagram</span></>
            ) : (
              <><Code className="h-3.5 w-3.5" /><span>Source</span></>
            )}
          </button>
          <button
            onClick={copyCode}
            className="flex items-center gap-1.5 text-xs text-wc-text-muted hover:text-wc-text-primary transition-colors"
            aria-label={copied ? "Copied" : "Copy code"}
            type="button"
          >
            {copied ? (
              <><Check className="h-3.5 w-3.5 text-green-400" /><span className="text-green-400">Copied</span></>
            ) : (
              <><Copy className="h-3.5 w-3.5" /><span>Copy</span></>
            )}
          </button>
        </div>
      </div>
      <div className="bg-wc-surface overflow-x-auto">
        {showSource ? (
          <pre className="p-4 text-sm text-wc-text-primary font-mono whitespace-pre overflow-x-auto">
            {code}
          </pre>
        ) : error ? (
          <div>
            <div className="px-4 py-2 bg-red-900/40 border-b border-red-700/50 text-xs text-red-300">
              {error}
            </div>
            <pre className="p-4 text-sm text-wc-text-primary font-mono whitespace-pre overflow-x-auto">
              {code}
            </pre>
          </div>
        ) : svgHtml ? (
          <div
            className="p-4 flex items-center justify-center [&>svg]:max-w-full"
            dangerouslySetInnerHTML={{ __html: svgHtml }}
          />
        ) : (
          <div className="p-8 flex items-center justify-center">
            <Loader2 className="h-6 w-6 text-wc-text-muted animate-spin" />
          </div>
        )}
      </div>
    </div>
  );
});
