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

export interface InlineTokenResolution {
  href: string;
  kind?: string;
}
export interface InlineCodeProps {
  children: ReactNode;
  resolveInlineToken?: (text: string) => InlineTokenResolution | null;
  looksLikeFileReference?: (text: string) => boolean;
  onLinkClick?: (href: string, event: MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick?: (path: string) => void;
  copyLabel?: string;
}

export function InlineCode({
  children,
  resolveInlineToken,
  looksLikeFileReference,
  onLinkClick,
  onFileReferenceClick,
  copyLabel = "Copy code",
}: InlineCodeProps) {
  const text = typeof children === "string" || typeof children === "number" ? String(children) : "";
  const resolution = resolveInlineToken?.(text);
  const isFile = !resolution && looksLikeFileReference?.(text);
  const { copied, copy } = useCodeCopy();
  const tokenClass =
    "rounded bg-[var(--markdown-code-surface)] px-space-3xs py-space-3xs font-mono text-[var(--markdown-code-text)]";

  if (resolution)
    return (
      <a
        href={resolution.href}
        data-entity-ref={resolution.kind === "entity" ? "true" : undefined}
        onClick={(event) => onLinkClick?.(resolution.href, event)}
        className={`${tokenClass} text-[var(--markdown-link)] underline`}
      >
        {text}
      </a>
    );
  if (isFile)
    return (
      <button
        type="button"
        onClick={() => onFileReferenceClick?.(text)}
        className={`${tokenClass} rounded text-[var(--markdown-link)] hover:opacity-80 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-app-focus`}
      >
        {text}
      </button>
    );
  return (
    <span className="group relative inline-flex items-center">
      <code className={tokenClass}>{text}</code>
      <button
        type="button"
        aria-label={copyLabel}
        onClick={() => void copy(text)}
        // The copy affordance used to be `hidden` until group-hover, which made it
        // unreachable by keyboard: display:none is not focusable. Fading it keeps
        // the same quiet visual while leaving it in the tab order.
        className="ml-space-3xs rounded px-space-3xs text-[10px] text-[var(--markdown-muted)] opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100 hover:opacity-80 motion-reduce:transition-none focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-app-focus"
      >
        {copied ? "Copied" : "Copy"}
      </button>
    </span>
  );
}
