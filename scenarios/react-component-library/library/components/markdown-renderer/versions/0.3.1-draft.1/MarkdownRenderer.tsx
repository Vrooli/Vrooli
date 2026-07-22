/**
 * @libraryId react-component-library:markdown-renderer
 * @displayName Markdown Renderer
 * @description GFM markdown renderer with routed inline-token, file, link, and Mermaid seams.
 * @version 0.3.1-draft.1
 * @tags ["markdown","rendering","gfm"]
 * @deps {"react":"^18","react-markdown":"^10.1.0","remark-gfm":"^4.0.1","shiki":"^4.3.1","mermaid":"^11.4.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { Component, type CSSProperties, type ErrorInfo, type MouseEvent, type ReactNode, useMemo } from "react";
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

export function MarkdownRenderer({ content, className, inline = false, resolveInlineToken, looksLikeFileReference, onLinkClick, onFileReferenceClick, onMermaidOpen }: MarkdownRendererProps) {
  const components = useMemo(() => ({
    code: ({ children, className: codeClass }: { children?: ReactNode; className?: string }) => {
      const text = String(children ?? "").replace(/\n$/, "");
      const language = codeClass?.replace(/^language-/, "");
      if (language === "mermaid") return <MermaidDiagram code={text} onMermaidOpen={onMermaidOpen} />;
      if (language) return <CodeBlock code={text} language={language} />;
      return <InlineCode resolveInlineToken={resolveInlineToken} looksLikeFileReference={looksLikeFileReference} onLinkClick={onLinkClick} onFileReferenceClick={onFileReferenceClick}>{text}</InlineCode>;
    },
    a: ({ href = "", children }: { href?: string; children?: ReactNode }) => <a href={href} onClick={(event) => onLinkClick?.(href, event)} className="text-[var(--markdown-link)] underline underline-offset-2">{children}</a>,
    blockquote: ({ children }: { children?: ReactNode }) => <blockquote className="my-3 border-l-2 border-[var(--markdown-link)] pl-3 italic text-[var(--markdown-muted)]">{children}</blockquote>,
    table: ({ children }: { children?: ReactNode }) => <div className="my-3 overflow-x-auto"><table className="border-collapse text-sm">{children}</table></div>,
    th: ({ children }: { children?: ReactNode }) => <th className="border border-[var(--markdown-border)] px-2 py-1 text-left">{children}</th>,
    td: ({ children }: { children?: ReactNode }) => <td className="border border-[var(--markdown-border)] px-2 py-1">{children}</td>,
  }), [looksLikeFileReference, onFileReferenceClick, onLinkClick, onMermaidOpen, resolveInlineToken]);
  if (!content) return null;
  const Wrapper = inline ? "span" : "div";
  return <MarkdownErrorBoundary content={content}><Wrapper className={className} style={markdownTokens}><ReactMarkdown remarkPlugins={[remarkGfm, remarkProsePaths]} components={components}>{content}</ReactMarkdown></Wrapper></MarkdownErrorBoundary>;
}