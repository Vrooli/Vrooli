/**
 * @libraryId react-component-library:inline-code
 * @displayName Inline Code
 * @version 0.3.5
 * @tags ["markdown","code"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { MouseEvent, ReactNode } from "react";
import { useCodeCopy } from "../../../../support/inline-code/versions/0.3.5/useCodeCopy";
export const inlineCodeStyles = `
[data-rcl-inline] { display: inline-flex; position: relative; align-items: center; gap: var(--space-3xs); color: var(--color-foreground); }
[data-rcl-inline].rcl-inline__token { border-radius: var(--radius-control); background: var(--color-surface-muted); padding: var(--space-3xs) var(--space-2xs); color: var(--color-foreground); font-family: var(--font-mono, "JetBrains Mono", "Fira Code", "SF Mono", Consolas, "Liberation Mono", Menlo, monospace); }
[data-rcl-inline] .rcl-inline__token { border-radius: var(--radius-control); background: var(--color-surface-muted); padding: var(--space-3xs) var(--space-2xs); color: var(--color-foreground); font-family: var(--font-mono, "JetBrains Mono", "Fira Code", "SF Mono", Consolas, "Liberation Mono", Menlo, monospace); }
[data-rcl-inline].rcl-inline__link { color: var(--color-accent); text-decoration: underline; text-underline-offset: var(--space-3xs); }
[data-rcl-inline] .rcl-inline__copy { visibility: hidden; border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); padding: var(--space-3xs) var(--space-2xs); font: var(--text-label); cursor: pointer; }
[data-rcl-inline]:hover .rcl-inline__copy, [data-rcl-inline]:focus-within .rcl-inline__copy { visibility: visible; }
[data-rcl-inline] .rcl-inline__copy:hover { background: color-mix(in srgb, var(--color-accent) 10%, transparent); }
`;
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
      <StyleSheet name="inline-code-0-3-3" css={inlineCodeStyles} />
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
