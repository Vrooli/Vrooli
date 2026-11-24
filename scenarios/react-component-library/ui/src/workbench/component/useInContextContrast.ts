import { useCallback, useEffect, useState, type RefObject } from "react";

import { contrastRatioFromCss, effectiveBackground, pickTextProbe } from "./instruments";

export interface ContrastReading {
  nodeId: string;
  label: string;
  foreground: string;
  background: string;
  ratio: number | null;
  rect: { x: number; y: number; w: number; h: number } | null;
}

interface InContextContrastOptions {
  pageRef: RefObject<HTMLDivElement | null>;
  scale: number;
  enabled: boolean;
  renderSpec: unknown;
  templateId: string;
  themeSignature: string;
  status: string;
  storyId: string | null;
}

/** Owns text selection and contrast readings for the context stage. */
export function useInContextContrast({
  pageRef,
  scale,
  enabled,
  renderSpec,
  templateId,
  themeSignature,
  status,
  storyId,
}: InContextContrastOptions) {
  const [contrastNodeId, setContrastNodeId] = useState<string | null>(null);
  const [contrast, setContrast] = useState<ContrastReading | null>(null);

  const readContrast = useCallback(
    (node: HTMLElement | null) => {
      const host = pageRef.current;
      const root = host?.querySelector<HTMLElement>(".tpl");
      if (!host || !root || !node) {
        setContrast(null);
        return;
      }
      const style = getComputedStyle(node);
      const background =
        effectiveBackground((element) => getComputedStyle(element).backgroundColor, node) ?? "";
      const ratio = contrastRatioFromCss(style.color, background);
      const rect = node.getBoundingClientRect();
      const rootRect = root.getBoundingClientRect();
      const factor = scale || 1;
      const label = node.textContent.trim().replace(/\s+/g, " ").slice(0, 32);
      setContrast({
        nodeId: node.dataset.nodeId ?? "",
        label: label || node.dataset.nodeId || "text",
        foreground: style.color,
        background,
        ratio,
        rect: {
          x: (rect.left - rootRect.left) / factor,
          y: (rect.top - rootRect.top) / factor,
          w: rect.width / factor,
          h: rect.height / factor,
        },
      });
    },
    [pageRef, scale],
  );

  useEffect(() => {
    if (!enabled) {
      setContrast(null);
      return;
    }
    const host = pageRef.current;
    if (!host) return;
    const explicit = contrastNodeId
      ? host.querySelector<HTMLElement>(`[data-node-id="${contrastNodeId}"]`)
      : null;
    // Only a node with text can produce a meaningful ratio. The old fallback
    // took the first `.tpl-text` even when it was empty.
    const fallback = explicit ?? pickTextProbe(host);
    if (!explicit && fallback?.dataset.nodeId) setContrastNodeId(fallback.dataset.nodeId);
    readContrast(fallback);
  }, [
    enabled,
    contrastNodeId,
    pageRef,
    readContrast,
    renderSpec,
    status,
    storyId,
    templateId,
    themeSignature,
  ]);

  // A click on the page picks the text node under the pointer for the
  // contrast probe. A native listener keeps the page a document, not a
  // focusable control: the instruments are the interactive surface.
  useEffect(() => {
    const host = pageRef.current;
    if (!host || !enabled) return;
    const onPick = (event: Event) => {
      const target = event.target as HTMLElement | null;
      const node = target?.closest<HTMLElement>("[data-node-id]");
      if (!node) return;
      setContrastNodeId(node.dataset.nodeId ?? null);
      readContrast(node);
    };
    host.addEventListener("click", onPick);
    return () => host.removeEventListener("click", onPick);
  }, [enabled, pageRef, readContrast, renderSpec]);

  return contrast;
}
