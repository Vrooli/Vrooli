/**
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @description Inline code token renderer with copy affordance.
 * @version 0.3.3
 * @tags ["markdown","code"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { MouseEvent, ReactNode } from "react";
import { useCodeCopy } from "./useCodeCopy";
import { inlineCodeStyles } from "./styles";

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

export const InlineCode = withClassName(function InlineCode({
  children,
  resolveInlineToken,
  looksLikeFileReference,
  onLinkClick,
  onFileReferenceClick,
  copyLabel = "Copy code",
}: InlineCodeProps) {
  const text =
    typeof children === "string" ? children : typeof children === "number" ? String(children) : "";
  const resolution = resolveInlineToken?.(text);
  const isFile = !resolution && looksLikeFileReference?.(text);
  const { copied, copy } = useCodeCopy();
  const tokenClass = "rcl-inline__token";

  if (resolution)
    return (
      <a
        data-testid="primitives.code"
        href={resolution.href}
        data-rcl-inline
        data-entity-ref={resolution.kind === "entity" ? "true" : undefined}
        onClick={(event) => onLinkClick?.(resolution.href, event)}
        className={`${tokenClass} rcl-inline__link`}
      >
        {text}
      </a>
    );
  if (isFile)
    return (
      <button
        data-testid="primitives.code"
        type="button"
        data-rcl-inline
        onClick={() => onFileReferenceClick?.(text)}
        className={`${tokenClass} rcl-inline__link`}
      >
        {text}
      </button>
    );
  return (
    <span className="rcl-inline" data-rcl-inline>
      <style data-rcl-inline-styles dangerouslySetInnerHTML={{ __html: inlineCodeStyles }} />
      <code className={tokenClass}>{text}</code>
      <button
        data-testid="primitives.code"
        type="button"
        aria-label={copyLabel}
        onClick={() => void copy(text)}
        className="rcl-inline__copy"
      >
        {copied ? "Copied" : "Copy"}
      </button>
    </span>
  );
});
