/**
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @description Routed entity, file-reference, and copyable inline code token.
 * @version 0.3.0
 * @tags ["markdown","code","inline"]
 * @deps {"react":"^18"}
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

export function InlineCode({ children, resolveInlineToken, looksLikeFileReference, onLinkClick, onFileReferenceClick, copyLabel = "Copy code" }: InlineCodeProps) {
  const text = typeof children === "string" ? children : typeof children === "number" ? String(children) : "";
  const resolution = resolveInlineToken?.(text);
  const isFile = !resolution && looksLikeFileReference?.(text);
  const { copied, copy } = useCodeCopy();
  const tokenClass = "rounded bg-[var(--markdown-code-surface)] px-1 py-0.5 font-mono text-[var(--markdown-code-text)]";

  if (resolution) return <a href={resolution.href} data-entity-ref={resolution.kind === "entity" ? "true" : undefined} onClick={(event) => onLinkClick?.(resolution.href, event)} className={`${tokenClass} text-[var(--markdown-link)] underline`}>{text}</a>;
  if (isFile) return <button type="button" onClick={() => onFileReferenceClick?.(text)} className={`${tokenClass} text-[var(--markdown-link)] hover:opacity-80`}>{text}</button>;
  return <span className="group relative inline-flex items-center"><code className={tokenClass}>{text}</code><button type="button" aria-label={copyLabel} onClick={() => void copy(text)} className="ml-1 hidden rounded px-1 text-[10px] text-[var(--markdown-muted)] group-hover:inline hover:opacity-80">{copied ? "Copied" : "Copy"}</button></span>;
}
