/**
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @description Inline code token renderer with copy affordance.
 * @version 0.3.0
 * @tags ["markdown","code"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */


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
  const text = String(children ?? "");
  const resolution = resolveInlineToken?.(text);
  const isFile = !resolution && looksLikeFileReference?.(text);
  const { copied, copy } = useCodeCopy();
  const tokenClass = "rounded bg-[var(--color-surface-muted)] [padding-inline:var(--space-3xs)] [padding-block:var(--space-3xs)] font-mono text-[var(--color-foreground)]";

  if (resolution) return <a href={resolution.href} data-entity-ref={resolution.kind === "entity" ? "true" : undefined} onClick={(event) => onLinkClick?.(resolution.href, event)} className={`${tokenClass} text-[var(--color-accent)] underline`}>{text}</a>;
  if (isFile) return <button type="button" onClick={() => onFileReferenceClick?.(text)} className={`${tokenClass} text-[var(--color-accent)] hover:bg-[var(--color-surface-raised)]`}>{text}</button>;
  return <span className="group relative inline-flex items-center"><code className={tokenClass}>{text}</code><button type="button" aria-label={copyLabel} onClick={() => void copy(text)} className="[margin-inline-start:var(--space-3xs)] hidden rounded [padding-inline:var(--space-3xs)] [font-size:var(--text-caption-size)] text-[var(--color-muted-foreground)] group-hover:inline hover:bg-[var(--color-surface-muted)]">{copied ? "Copied" : "Copy"}</button></span>;
}