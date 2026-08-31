import { useEffect, useState } from "react";

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
      void import("mermaid")
        .then(({ default: mermaid }) => {
          mermaid.initialize({ startOnLoad: false, securityLevel: "strict" });
          const id = `rcl-mermaid-${Math.random().toString(36).slice(2)}`;
          return mermaid.render(id, code);
        })
        .then(({ svg }) => {
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
