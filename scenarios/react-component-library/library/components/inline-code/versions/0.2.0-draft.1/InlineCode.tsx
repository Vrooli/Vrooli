/**
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @description Inline code token renderer with copy affordance.
 * @version 0.2.0-draft.1
 * @tags ["markdown","code"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
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

/** @libraryId react-component-library:inline-code @version 0.2.0 */
export function InlineCode({ children, resolveInlineToken, looksLikeFileReference, onLinkClick, onFileReferenceClick, copyLabel = "Copy code" }: InlineCodeProps) {
  const text = String(children ?? "");
  const resolution = resolveInlineToken?.(text);
  const isFile = !resolution && looksLikeFileReference?.(text);
  const { copied, copy } = useCodeCopy();
  const tokenClass = "rounded bg-slate-800 px-1 py-0.5 font-mono text-slate-200";

  if (resolution) return <a href={resolution.href} data-entity-ref={resolution.kind === "entity" ? "true" : undefined} onClick={(event) => onLinkClick?.(resolution.href, event)} className={`${tokenClass} text-cyan-200 underline`}>{text}</a>;
  if (isFile) return <button type="button" onClick={() => onFileReferenceClick?.(text)} className={`${tokenClass} text-cyan-200 hover:bg-slate-700`}>{text}</button>;
  return <span className="group relative inline-flex items-center"><code className={tokenClass}>{text}</code><button type="button" aria-label={copyLabel} onClick={() => void copy(text)} className="ml-1 hidden rounded px-1 text-[10px] text-slate-400 group-hover:inline hover:bg-slate-800">{copied ? "Copied" : "Copy"}</button></span>;
}