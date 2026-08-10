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

/**
 * LOCAL EDIT (drift from markdown-renderer 0.3.2): mermaid theming.
 *
 * The library default renders the light "default" theme, which lands as a
 * white diagram on Swarm Manager's slate surface — unreadable next to the
 * message text around it. Swarm Manager is dark-only, so the diagram is
 * pinned to the dark theme with the app's own token values rather than
 * mermaid's stock dark palette, which clashes with the cyan accent.
 *
 * web-console carries an equivalent local edit on the same component; if the
 * library ever adopts a theme parameter, both should collapse onto it.
 */
const MERMAID_THEME_VARIABLES = {
  // slate-700 / slate-200 / slate-600: the same surface, text, and border
  // ramp the surrounding markdown uses.
  primaryColor: "#334155",
  primaryTextColor: "#e2e8f0",
  primaryBorderColor: "#475569",
  // cyan-500 — the app accent, so edges read as part of this UI.
  lineColor: "#06b6d4",
  secondaryColor: "#1e293b",
  tertiaryColor: "#0f172a",
  noteBkgColor: "#1e293b",
  noteTextColor: "#e2e8f0",
  noteBorderColor: "#475569",
} as const;

export function useMermaidSvg(code: string, debounceMs = 120): MermaidSvgState {
  const [state, setState] = useState<MermaidSvgState>({ loading: true });

  useEffect(() => {
    let active = true;
    setState({ loading: true });
    const timer = window.setTimeout(() => {
      void import("mermaid")
        .then(({ default: mermaid }) => {
          mermaid.initialize({
            startOnLoad: false,
            securityLevel: "strict",
            theme: "dark",
            themeVariables: MERMAID_THEME_VARIABLES,
          });
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