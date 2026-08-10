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

/**
 * LOCAL EDIT (drift from markdown-renderer 0.3.2): cross-mount render cache.
 *
 * The library hook renders from scratch on every mount and opens with
 * `loading: true`. That is fine for a static document and wrong for a live
 * transcript: Swarm Manager polls the session every 3s, and any re-render that
 * changes the markdown component map remounts the diagram, so a stable diagram
 * flickered placeholder → SVG → placeholder on a 3s cadence, paying a full
 * dagre layout each time.
 *
 * MarkdownRenderer no longer remounts on poll (see its own LOCAL EDIT), which
 * removes the cause. This cache is the second line of defence: it makes a
 * repeat mount of *already-rendered source* free and synchronous, so any
 * remount we have not anticipated — a route change, a virtualised list
 * recycling a row, the same diagram quoted in two messages — is silent instead
 * of a visible reflow.
 *
 * Keyed on the source text alone, which is sound only because the theme above
 * is a module constant. If theming ever becomes per-call, it must join the key.
 */
type CacheEntry = { svg: string } | { error: string };

const RENDER_CACHE = new Map<string, CacheEntry>();
const IN_FLIGHT = new Map<string, Promise<CacheEntry>>();

// Bounds chosen against what a transcript realistically holds: a few dozen
// diagrams, each SVG typically 10–80 KB. Whichever bound trips first evicts
// oldest-first, so a single pathological diagram cannot pin the cache.
const MAX_CACHE_ENTRIES = 24;
const MAX_CACHE_CHARS = 4_000_000;

function cacheSizeChars(): number {
  let total = 0;
  for (const entry of RENDER_CACHE.values()) total += "svg" in entry ? entry.svg.length : entry.error.length;
  return total;
}

function storeInCache(code: string, entry: CacheEntry): void {
  RENDER_CACHE.set(code, entry);
  // Map preserves insertion order, so the first key is always the oldest.
  while (RENDER_CACHE.size > MAX_CACHE_ENTRIES || (RENDER_CACHE.size > 1 && cacheSizeChars() > MAX_CACHE_CHARS)) {
    const oldest = RENDER_CACHE.keys().next();
    if (oldest.done) break;
    RENDER_CACHE.delete(oldest.value);
  }
}

function stateFromEntry(entry: CacheEntry): MermaidSvgState {
  return "svg" in entry ? { svg: entry.svg, loading: false } : { error: entry.error, loading: false };
}

// mermaid.initialize mutates module-global config, so it only needs to run
// once. Doing it per render was redundant work on every diagram.
let initialized = false;
// Monotonic instead of random: mermaid only needs the id to be unique within
// the document, and a counter keeps rendered SVG markup reproducible.
let renderSequence = 0;

async function renderMermaid(code: string): Promise<CacheEntry> {
  try {
    const { default: mermaid } = await import("mermaid");
    if (!initialized) {
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        theme: "dark",
        themeVariables: MERMAID_THEME_VARIABLES,
      });
      initialized = true;
    }
    renderSequence += 1;
    const { svg } = await mermaid.render(`rcl-mermaid-${renderSequence}`, code);
    return { svg };
  } catch (reason: unknown) {
    return { error: reason instanceof Error ? reason.message : "Unable to render Mermaid diagram" };
  }
}

/**
 * Renders `code`, sharing one mermaid pass across every concurrent caller with
 * the same source. Failures are cached alongside successes: invalid syntax
 * fails deterministically, so retrying it on each mount only burns layout time
 * to reproduce the same message.
 */
function renderShared(code: string): Promise<CacheEntry> {
  // Re-check the cache rather than trusting the caller's earlier look: between
  // a debounce being armed and it firing, a sibling mount of the same source
  // can have completed the render outright.
  const cached = RENDER_CACHE.get(code);
  if (cached) return Promise.resolve(cached);
  const pending = IN_FLIGHT.get(code);
  if (pending) return pending;
  const promise = renderMermaid(code).then((entry) => {
    IN_FLIGHT.delete(code);
    storeInCache(code, entry);
    return entry;
  });
  IN_FLIGHT.set(code, promise);
  return promise;
}

/** Test seam: the cache is module state and would otherwise leak across specs. */
export function __resetMermaidCacheForTests(): void {
  RENDER_CACHE.clear();
  IN_FLIGHT.clear();
  initialized = false;
  renderSequence = 0;
}

export function useMermaidSvg(code: string, debounceMs = 120): MermaidSvgState {
  // Seeded from the cache during the first render rather than in an effect, so
  // an already-rendered diagram paints its SVG on the mounting frame and never
  // shows the placeholder at all.
  const [state, setState] = useState<MermaidSvgState>(() => {
    const cached = RENDER_CACHE.get(code);
    return cached ? stateFromEntry(cached) : { loading: true };
  });

  useEffect(() => {
    const cached = RENDER_CACHE.get(code);
    if (cached) {
      // Covers the `code` prop changing on an already-mounted component, where
      // the lazy initialiser above no longer applies.
      setState(stateFromEntry(cached));
      return undefined;
    }

    let active = true;
    setState({ loading: true });
    // The debounce exists for the streaming case, where `code` changes on every
    // token; it is pure latency for a diagram that is already complete. An
    // in-flight render for this exact source proves the source is settled, so
    // joining it directly is both faster and cheaper.
    if (IN_FLIGHT.has(code)) {
      void renderShared(code).then((entry) => { if (active) setState(stateFromEntry(entry)); });
      return () => { active = false; };
    }

    const timer = window.setTimeout(() => {
      void renderShared(code).then((entry) => { if (active) setState(stateFromEntry(entry)); });
    }, debounceMs);
    return () => { active = false; window.clearTimeout(timer); };
  }, [code, debounceMs]);

  return state;
}
