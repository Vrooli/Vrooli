import type { TemplateNode } from "../templates/spec";

export interface DeviceChromeVisibility {
  statusBar: boolean;
  island: boolean;
}

const DEFAULT_CHROME: DeviceChromeVisibility = { statusBar: true, island: true };
const sourceChromeCache = new WeakMap<TemplateNode, DeviceChromeVisibility>();

/** Avoid drawing device chrome over matching chrome already imported from HTML. */
export function deviceChromeVisibility(spec: TemplateNode): DeviceChromeVisibility {
  const cached = sourceChromeCache.get(spec);
  if (cached) return cached;
  const sourceHtml = spec.props.sourceHtml;
  if (typeof sourceHtml !== "string" || typeof DOMParser === "undefined") {
    // HTML imports are materialized as editable layer trees. Their ids retain
    // the importer's `-html-` marker after the source document is expanded.
    // The source status bar is one of those layers; don't draw another phone
    // overlay on top of it.
    const importedHtmlTree = hasImportedHtmlLayer(spec);
    const visibility = importedHtmlTree ? { statusBar: false, island: false } : DEFAULT_CHROME;
    sourceChromeCache.set(spec, visibility);
    return visibility;
  }
  const source = new DOMParser().parseFromString(sourceHtml, "text/html");
  const visibility = {
    statusBar: !source.querySelector(".statusbar, [data-device-status-bar]"),
    island: !source.querySelector(".island, [data-device-island]"),
  };
  sourceChromeCache.set(spec, visibility);
  return visibility;
}

function hasImportedHtmlLayer(node: TemplateNode): boolean {
  if (node.id.includes("-html-")) return true;
  return node.children?.some(hasImportedHtmlLayer) ?? false;
}
