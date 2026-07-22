/**
 * @vrooliComponentSource react-component-library:markdown-renderer
 * @vrooliComponentVersion 0.3.2
 * @vrooliComponentAdoption 612450da-7d3d-4888-85a9-e9ecf63254a6
 * @vrooliComponentAppliedAt 2026-07-21T21:01:34Z
 * @vrooliComponentSourceSha256 115ca575ac122dc181b31a053058fb4ac40a3c885e382fe186b6c63739810177
 * @vrooliComponentDriftHash 115ca575ac122dc181b31a053058fb4ac40a3c885e382fe186b6c63739810177
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
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
        .then(({ svg }) => { if (active) setState({ svg, loading: false }); })
        .catch((reason: unknown) => {
          if (active) setState({
            error: reason instanceof Error ? reason.message : "Unable to render Mermaid diagram",
            loading: false,
          });
        });
    }, debounceMs);
    return () => { active = false; window.clearTimeout(timer); };
  }, [code, debounceMs]);

  return state;
}