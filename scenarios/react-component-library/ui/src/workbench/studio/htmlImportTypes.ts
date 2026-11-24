import type { TemplateNode } from "../templates/spec";

export const HTML_LIMITS = {
  /** Input bytes accepted before the file is refused. */
  bytes: 2 * 1024 * 1024,
  /** Source elements visited before the scan stops. */
  elements: 2000,
  /** Content layers emitted before the import stops (the page root is extra). */
  layers: 1000,
  /** Maximum produced layer depth: page is 0, its first children are 1. */
  depth: 12,
  /** Sanitized vector payload accepted per artwork layer. */
  svgBytes: 48 * 1024,
  /** Total embedded vector payload, including expanded local sprite symbols. */
  svgTotalBytes: 256 * 1024,
  /** Aggregate static CSS serialized into editable layer props. */
  styleBytes: 256 * 1024,
} as const;

export type HtmlImportReason = "size" | "empty" | "parse";

/** A typed refusal so the dialog can explain *why* a file was not imported. */
export class HtmlImportError extends Error {
  readonly reason: HtmlImportReason;

  constructor(reason: HtmlImportReason, message: string) {
    super(message);
    this.name = "HtmlImportError";
    this.reason = reason;
  }
}

/**
 * One selectable region of the source document. Explicitly named regions sort
 * before anonymous regions; order within each group follows the source document.
 * `depth` doubles as the indentation for a flat tree/list control.
 */
export interface HtmlRegion {
  id: string;
  /** Lower-case source tag (`section`, `p`, `table`…). */
  tag: string;
  /** Collapsed text snippet, capped at 80 characters; may be empty. */
  label: string;
  /** Safe source hint captured for discovery, never copied to imported props. */
  sourceHint?: string;
  /** Source nesting relative to the body: top-level children are 0. */
  depth: number;
  /** Produced layers in this element's subtree, including the element itself. */
  layers: number;
  /** Direct source element children, after sanitisation. */
  children: number;
  /** Element-child path from the original body, used to preserve source identity in preview. */
  sourcePath?: number[];
}

export interface ParsedHtml {
  /** `<title>`, the file name, or "Imported HTML". */
  title: string;
  /** Content layers across the whole document (page root excluded). */
  layers: number;
  /** Importable regions, document order. Elements that produced nothing are omitted. */
  regions: HtmlRegion[];
  warnings: string[];
  /** Build an inert, network-free visual reference for a selected source region. */
  sourcePreview(regionId: string | null, stylesheets?: string[]): string;
  /** Build a fresh spec for one region; `null` means the whole body. */
  regionSpec(id: string | null): TemplateNode;
}
