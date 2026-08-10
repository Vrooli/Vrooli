/**
 * @vrooliComponentSource react-component-library:markdown-renderer
 * @vrooliComponentVersion 0.3.2
 * @vrooliComponentAdoption 612450da-7d3d-4888-85a9-e9ecf63254a6
 * @vrooliComponentAppliedAt 2026-07-21T21:01:34Z
 * @vrooliComponentSourceSha256 67fd0cbfc805386b0bb2dfbad4c8d4fefbfc352fe431649045e7a03c8831cda8
 * @vrooliComponentDriftHash 67fd0cbfc805386b0bb2dfbad4c8d4fefbfc352fe431649045e7a03c8831cda8
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { Component, type CSSProperties, type ErrorInfo, type MouseEvent, type ReactNode, useMemo, useRef } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "./CodeBlock";
import { InlineCode, type InlineTokenResolution } from "./InlineCode";
import { MermaidDiagram } from "./MermaidDiagram";
import { remarkProsePaths } from "./languageDetection";

export type { InlineTokenResolution } from "./InlineCode";
export { CodeBlock } from "./CodeBlock";
export { InlineCode } from "./InlineCode";
export { MermaidDiagram } from "./MermaidDiagram";
export { normalizeCodeLanguage, languageLabel, remarkProsePaths } from "./languageDetection";
export { useCodeCopy } from "./useCodeCopy";
export { useMermaidSvg } from "./useMermaidSvg";

export interface MarkdownRendererProps {
  content: string;
  className?: string;
  inline?: boolean;
  resolveInlineToken?: (text: string) => InlineTokenResolution | null;
  looksLikeFileReference?: (text: string) => boolean;
  onLinkClick?: (href: string, event: MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick?: (path: string) => void;
  onMermaidOpen?: (code: string) => void;
  "data-testid"?: string;
}

function reactNodeText(value: ReactNode): string {
  if (typeof value === "string" || typeof value === "number" || typeof value === "bigint") return String(value);
  if (Array.isArray(value)) return value.map(reactNodeText).join("");
  return "";
}

class MarkdownErrorBoundary extends Component<{ content: string; children: ReactNode }, { failed: boolean }> {
  state = { failed: false };
  static getDerivedStateFromError() { return { failed: true }; }
  componentDidCatch(_error: Error, _info: ErrorInfo) {}
  render() { return this.state.failed ? <pre className="whitespace-pre-wrap font-mono text-sm text-[var(--markdown-code-text)]">{this.props.content}</pre> : this.props.children; }
}

const markdownTokens: CSSProperties & Record<`--${string}`, string> = {
  "--markdown-border": "var(--color-border, currentColor)",
  "--markdown-code-surface": "var(--color-surface-muted, transparent)",
  "--markdown-code-text": "var(--color-foreground, currentColor)",
  "--markdown-muted": "var(--color-muted-foreground, currentColor)",
  "--markdown-link": "var(--color-accent, currentColor)",
  "--markdown-error": "var(--color-danger, currentColor)",
};

/**
 * LOCAL EDIT (drift from markdown-renderer 0.3.2): identity-stable component
 * map, and a memoized tree.
 *
 * The library builds `components` with the caller's callbacks in the useMemo
 * dependency list. Every real caller passes at least one inline arrow (a
 * message bubble closing over its own row), so the map was rebuilt on every
 * render — and a rebuilt map means a brand-new `code` *function*, which React
 * reads as a changed element type and responds to by unmounting the subtree and
 * mounting a fresh one.
 *
 * That is invisible for a paragraph and expensive for anything holding state: a
 * mermaid diagram restarted its full layout, a CodeBlock lost its "Copied"
 * flash, a wide table lost its horizontal scroll position. On the session page,
 * which polls every 3s, a diagram visibly flickered placeholder → SVG →
 * placeholder forever.
 *
 * The fix routes the callbacks through a ref so the map can be memoized on []
 * and the element types stay put. Two consequences worth knowing:
 *
 *  - Handlers are read at call time, so a re-render is enough to pick up a new
 *    one; no remount is needed and none happens.
 *  - `resolveInlineToken` / `looksLikeFileReference` run *during* child render,
 *    so they stay in the tree memo's dependencies. A caller that stabilises
 *    them (see ChatMessageBubble) gets the parse skipped entirely; a caller
 *    that does not still gets a cheap re-render instead of a remount.
 *
 * Writing the ref during render is safe for this shape: it is a latest-props
 * cache, rewritten wholesale on every render pass, and its only readers are
 * this component's own children rendering later in the same pass.
 */
export function MarkdownRenderer({ content, className, inline = false, resolveInlineToken, looksLikeFileReference, onLinkClick, onFileReferenceClick, onMermaidOpen, "data-testid": testId }: MarkdownRendererProps) {
  const callbacks = useRef({ resolveInlineToken, looksLikeFileReference, onLinkClick, onFileReferenceClick, onMermaidOpen });
  callbacks.current = { resolveInlineToken, looksLikeFileReference, onLinkClick, onFileReferenceClick, onMermaidOpen };

  // `onMermaidOpen`'s *presence* decides whether the diagram toolbar shows an
  // Open button, so it is read as a boolean here and fed to the tree memo.
  // Its identity still does not matter.
  const canOpenMermaid = Boolean(onMermaidOpen);

  const components = useMemo(() => ({
    code: ({ children, className: codeClass }: { children?: ReactNode; className?: string }) => {
      const text = reactNodeText(children).replace(/\n$/, "");
      const language = codeClass?.replace(/^language-/, "");
      if (language === "mermaid") {
        const open = callbacks.current.onMermaidOpen;
        return <MermaidDiagram code={text} onMermaidOpen={open ? (value) => callbacks.current.onMermaidOpen?.(value) : undefined} />;
      }
      if (language) return <CodeBlock code={text} language={language} />;
      return <InlineCode
        resolveInlineToken={(value) => callbacks.current.resolveInlineToken?.(value) ?? null}
        looksLikeFileReference={(value) => callbacks.current.looksLikeFileReference?.(value) ?? false}
        onLinkClick={(href, event) => callbacks.current.onLinkClick?.(href, event)}
        onFileReferenceClick={(path) => callbacks.current.onFileReferenceClick?.(path)}
      >{text}</InlineCode>;
    },
    a: ({ href = "", children }: { href?: string; children?: ReactNode }) => <a href={href} onClick={(event) => callbacks.current.onLinkClick?.(href, event)} className="text-[var(--markdown-link)] underline underline-offset-2">{children}</a>,
    blockquote: ({ children }: { children?: ReactNode }) => <blockquote className="my-3 border-l-2 border-[var(--markdown-link)] pl-3 italic text-[var(--markdown-muted)]">{children}</blockquote>,
    table: ({ children }: { children?: ReactNode }) => <div className="my-3 overflow-x-auto"><table className="border-collapse text-sm">{children}</table></div>,
    th: ({ children }: { children?: ReactNode }) => <th className="border border-[var(--markdown-border)] px-2 py-1 text-left">{children}</th>,
    td: ({ children }: { children?: ReactNode }) => <td className="border border-[var(--markdown-border)] px-2 py-1">{children}</td>,
  }), []);

  // Reusing the element instance lets React bail out of the whole subtree, so
  // an unchanged message skips both the mdast parse and the React render.
  //
  // The last three dependencies look unused to the exhaustive-deps rule and are
  // not: `components` reaches them through `callbacks.current` at child-render
  // time, which static analysis cannot follow. They are exactly the inputs that
  // affect rendered *output* — which tokens resolve to links, whether the
  // diagram toolbar offers Open — so dropping them would serve a stale tree.
  // The pure event handlers are deliberately absent: their identity changes
  // nothing on screen.
  const tree = useMemo(
    () => <ReactMarkdown remarkPlugins={[remarkGfm, remarkProsePaths]} components={components}>{content}</ReactMarkdown>,
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [content, components, canOpenMermaid, resolveInlineToken, looksLikeFileReference],
  );

  if (!content) return null;
  const Wrapper = inline ? "span" : "div";
  return <MarkdownErrorBoundary content={content}><Wrapper className={className} style={markdownTokens} data-testid={testId}>{tree}</Wrapper></MarkdownErrorBoundary>;
}

export default MarkdownRenderer;
