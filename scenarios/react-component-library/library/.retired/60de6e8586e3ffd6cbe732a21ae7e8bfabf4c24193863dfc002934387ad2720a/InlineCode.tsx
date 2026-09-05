/**
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @description Routed entity, file-reference, and copyable inline code token.
 * @version 0.4.0
 * @tags ["markdown","code","inline"]
 * @deps {"react":"^18"}
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { MouseEvent, ReactNode } from "react";
import { useCodeCopy } from "./useCodeCopy";
import { markdownStyles } from "./markdownStyles";

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
    typeof children === "string"
      ? children
      : typeof children === "number"
        ? String(children)
        : "";
  const resolution = resolveInlineToken?.(text);
  const isFile = !resolution && looksLikeFileReference?.(text);
  const { copied, copy } = useCodeCopy();
  const tokenClass = "rcl-md__inline-token";

  if (resolution)
    return (
      <a
        href={resolution.href}
        data-rcl-markdown
        data-entity-ref={resolution.kind === "entity" ? "true" : undefined}
        onClick={(event) => onLinkClick?.(resolution.href, event)}
        className={`${tokenClass} rcl-md__link`}
      >
        {text}
      </a>
    );
  if (isFile)
    return (
      <button
        type="button"
        data-rcl-markdown
        onClick={() => onFileReferenceClick?.(text)}
        className={`${tokenClass} rcl-md__link`}
      >
        {text}
      </button>
    );
  return (
    <span className="rcl-md__inline" data-rcl-markdown>
      <StyleSheet name="markdown-renderer-0-4-0" css={markdownStyles} />
      <code className={tokenClass}>{text}</code>
      <button
        type="button"
        aria-label={copyLabel}
        onClick={() => void copy(text)}
        className="rcl-md__inline-copy"
      >
        {copied ? "Copied" : "Copy"}
      </button>
    </span>
  );
});
