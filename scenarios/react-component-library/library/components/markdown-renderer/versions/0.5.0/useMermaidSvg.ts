import { useEffect, useState } from "react";

const renderCache = new Map<string, Promise<string>>();

export function resetMermaidRenderCacheForTests(): void {
  renderCache.clear();
}

function renderMermaid(code: string): Promise<string> {
  const pending = renderCache.get(code);
  if (pending) return pending;

  const render = import("mermaid")
    .then(({ default: mermaid }) => {
      mermaid.initialize({ startOnLoad: false, securityLevel: "strict" });
      const id = `rcl-mermaid-${Math.random().toString(36).slice(2)}`;
      return mermaid.render(id, code);
    })
    .then(({ svg }) => svg);
  renderCache.set(code, render);
  return render;
}

export interface MermaidSvgState {
  svg?: string;
  error?: string;
  loading: boolean;
}

export function useMermaidSvg(code: string, debounceMs = 120): MermaidSvgState {
  const [state, setState] = useState<MermaidSvgState>({ loading: true });

  useEffect(() => {
    let active = true;
    setState({ loading: true });
    const timer = window.setTimeout(() => {
      void renderMermaid(code)
        .then((svg) => {
          if (active) setState({ svg, loading: false });
        })
        .catch((reason: unknown) => {
          if (active)
            setState({
              error: reason instanceof Error ? reason.message : "Unable to render Mermaid diagram",
              loading: false,
            });
        });
    }, debounceMs);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [code, debounceMs]);

  return state;
}
