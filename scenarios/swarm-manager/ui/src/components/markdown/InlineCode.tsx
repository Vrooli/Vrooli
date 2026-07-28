/**
 * @vrooliComponentSource react-component-library:markdown-renderer
 * @vrooliComponentVersion 0.3.2
 * @vrooliComponentAdoption 612450da-7d3d-4888-85a9-e9ecf63254a6
 * @vrooliComponentAppliedAt 2026-07-21T21:01:34Z
 * @vrooliComponentSourceSha256 5d075b2ae6276fc785bfcf889cd1d86676801bd75e5a3342a5ef4be3af5e05c2
 * @vrooliComponentDriftHash 5d075b2ae6276fc785bfcf889cd1d86676801bd75e5a3342a5ef4be3af5e05c2
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { MouseEvent, ReactNode } from "react";
import { useCodeCopy } from "./useCodeCopy";

export interface InlineTokenResolution { href: string; kind?: string; }
export interface InlineCodeProps {
  children: ReactNode;
  resolveInlineToken?: (text: string) => InlineTokenResolution | null;
  looksLikeFileReference?: (text: string) => boolean;
  onLinkClick?: (href: string, event: MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick?: (path: string) => void;
  copyLabel?: string;
}

function reactNodeText(value: ReactNode): string {
  if (typeof value === "string" || typeof value === "number" || typeof value === "bigint") return String(value);
  if (Array.isArray(value)) return value.map(reactNodeText).join("");
  return "";
}

export function InlineCode({ children, resolveInlineToken, looksLikeFileReference, onLinkClick, onFileReferenceClick, copyLabel = "Copy code" }: InlineCodeProps) {
  const text = reactNodeText(children);
  const resolution = resolveInlineToken?.(text);
  const isFile = !resolution && looksLikeFileReference?.(text);
  const { copied, copy } = useCodeCopy();
  const tokenClass = "rounded bg-[var(--markdown-code-surface)] px-1 py-0.5 font-mono text-[var(--markdown-code-text)]";

  if (resolution) return <a href={resolution.href} data-entity-ref={resolution.kind === "entity" ? "true" : undefined} onClick={(event) => onLinkClick?.(resolution.href, event)} className={`${tokenClass} text-[var(--markdown-link)] underline`}>{text}</a>;
  if (isFile) return <button type="button" onClick={() => onFileReferenceClick?.(text)} className={`${tokenClass} text-[var(--markdown-link)] hover:opacity-80`}>{text}</button>;
  return <span className="group relative inline-flex items-center"><code className={tokenClass}>{text}</code><button type="button" aria-label={copyLabel} onClick={() => void copy(text)} className="ml-1 hidden rounded px-1 text-[10px] text-[var(--markdown-muted)] group-hover:inline hover:opacity-80">{copied ? "Copied" : "Copy"}</button></span>;
}
