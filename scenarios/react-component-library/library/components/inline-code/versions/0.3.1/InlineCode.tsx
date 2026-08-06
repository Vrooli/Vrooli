/**
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @description Inline code token renderer with copy affordance.
 * @version 0.3.1
 * @tags ["markdown","code"]
 * @deps {"react":"^18"}
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

export function InlineCode({ children, resolveInlineToken, looksLikeFileReference, onLinkClick, onFileReferenceClick, copyLabel = "Copy code" }: InlineCodeProps) {
  const text = typeof children === "string" ? children : typeof children === "number" ? String(children) : "";
  const resolution = resolveInlineToken?.(text);
  const isFile = !resolution && looksLikeFileReference?.(text);
  const { copied, copy } = useCodeCopy();
  const tokenClass = "rounded bg-[var(--markdown-code-surface)] [padding-inline:var(--space-3xs)] [padding-block:var(--space-3xs)] font-mono text-[var(--markdown-code-text)]";

  if (resolution) return <a href={resolution.href} data-entity-ref={resolution.kind === "entity" ? "true" : undefined} onClick={(event) => onLinkClick?.(resolution.href, event)} className={`${tokenClass} text-[var(--markdown-link)] underline`}>{text}</a>;
  if (isFile) return <button type="button" onClick={() => onFileReferenceClick?.(text)} className={`${tokenClass} text-[var(--markdown-link)] hover:opacity-80`}>{text}</button>;
  return <span className="group relative inline-flex items-center"><code className={tokenClass}>{text}</code><button type="button" aria-label={copyLabel} onClick={() => void copy(text)} className="[margin-inline-start:var(--space-3xs)] hidden rounded [padding-inline:var(--space-3xs)] [font-size:var(--text-caption-size)] text-[var(--markdown-muted)] group-hover:inline hover:opacity-80">{copied ? "Copied" : "Copy"}</button></span>;
}
