import { useEffect, useState } from "react";
import { useFileViewerHost } from "../../i18n/context.js";
import type { DiagramEngine } from "../../loaders.js";

const DIAGRAM_CONFIG = {
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
} as const;

// One initialised engine per host loader, shared by the inline diagram and the
// full-screen viewer so the theme has a single source of truth. A failed load
// is forgotten so a later render can retry.
const engines = new WeakMap<() => Promise<DiagramEngine>, Promise<DiagramEngine>>();

function loadEngine(load: () => Promise<DiagramEngine>): Promise<DiagramEngine> {
  let engine = engines.get(load);
  if (!engine) {
    engine = load().then((loaded) => { loaded.initialize(DIAGRAM_CONFIG); return loaded; });
    engines.set(load, engine);
    engine.catch(() => { engines.delete(load); });
  }
  return engine;
}

export interface MermaidRenderResult {
  svgHtml: string | null;
  error: string | null;
  loading: boolean;
  /** The host supplied no diagram engine; show the source instead. */
  unavailable: boolean;
}

const RENDER_DEBOUNCE_MS = 100;

/**
 * Renders Mermaid source to SVG markup with the host's diagram engine. It
 * debounces rapid `code` changes and cancels stale renders, so the latest
 * source always wins. `loading` is true while a non-empty diagram is being
 * rendered and no result or error is available yet; `unavailable` is true when
 * the host configured no engine (`loaders.diagram`).
 */
export function useMermaidSvg(code: string): MermaidRenderResult {
  const load = useFileViewerHost().loaders.diagram;
  const [svgHtml, setSvgHtml] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(() => Boolean(load) && code.trim().length > 0);

  useEffect(() => {
    let cancelled = false;
    setError(null);

    if (!load || !code.trim()) {
      setSvgHtml(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    const timer = setTimeout(() => {
      async function render(engineLoader: () => Promise<DiagramEngine>) {
        try {
          const engine = await loadEngine(engineLoader);
          if (cancelled) return;
          const { svg } = await engine.render(`mermaid-${crypto.randomUUID()}`, code);
          if (cancelled) return;
          setSvgHtml(svg);
          setError(null);
        } catch (err) {
          if (cancelled) return;
          setError(err instanceof Error ? err.message : "Failed to render diagram");
          setSvgHtml(null);
        } finally {
          if (!cancelled) setLoading(false);
        }
      }
      void render(load);
    }, RENDER_DEBOUNCE_MS);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [code, load]);

  return { svgHtml, error, loading, unavailable: !load };
}
