/**
 * @libraryId react-component-library:markdown-renderer
 * @displayName Markdown Renderer
 * @description Syntax-highlighted fenced code block with bounded cache and copy feedback.
 * @version 0.4.2
 * @tags ["markdown","rendering","gfm"]
 * @deps {"react":"^18","react-markdown":"^10.1.0","remark-gfm":"^4.0.1","shiki":"^4.3.1","mermaid":"^11.4.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

import {
  Component,
  type CSSProperties,
  type ErrorInfo,
  type MouseEvent,
  type ReactNode,
  useMemo,
} from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "./CodeBlock";
import { InlineCode, type InlineTokenResolution } from "./InlineCode";
import { MermaidDiagram } from "./MermaidDiagram";
import { remarkProsePaths } from "./languageDetection";
import { markdownStyles } from "./markdownStyles";

export type { InlineTokenResolution } from "./InlineCode";
export { CodeBlock } from "./CodeBlock";
export { InlineCode } from "./InlineCode";
export { MermaidDiagram } from "./MermaidDiagram";
export { normalizeCodeLanguage, languageLabel, remarkProsePaths } from "./languageDetection";
export { useCodeCopy } from "./useCodeCopy";
export { resetMermaidRenderCacheForTests, useMermaidSvg } from "./useMermaidSvg";

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

class MarkdownErrorBoundary extends Component<
  { content: string; children: ReactNode },
  { failed: boolean }
> {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  componentDidCatch(_error: Error, _info: ErrorInfo) {}
  render() {
    return this.state.failed ? (
      <pre className="rcl-md__error">{this.props.content}</pre>
    ) : (
      this.props.children
    );
  }
}

const markdownTokens: CSSProperties & Record<`--${string}`, string> = {
  "--markdown-border": "var(--color-border, #cbd5e1)",
  "--markdown-code-surface": "var(--color-surface-muted, #f1f5f9)",
  "--markdown-code-text": "var(--color-foreground, #0f172a)",
  "--markdown-muted": "var(--color-muted-foreground, #64748b)",
  "--markdown-link": "var(--color-accent, #0891b2)",
  "--markdown-error": "var(--color-danger, #dc2626)",
};

export const MarkdownRenderer = withClassName(function MarkdownRenderer({
  content,
  className,
  inline = false,
  resolveInlineToken,
  looksLikeFileReference,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
}: MarkdownRendererProps) {
  const components = useMemo(
    () => ({
      code: ({ children, className: codeClass }: { children?: ReactNode; className?: string }) => {
        const text = (
          typeof children === "string"
            ? children
            : typeof children === "number"
              ? String(children)
              : ""
        ).replace(/\n$/, "");
        const language = codeClass?.replace(/^language-/, "");
        if (language === "mermaid")
          return <MermaidDiagram code={text} onMermaidOpen={onMermaidOpen} />;
        if (language) return <CodeBlock code={text} language={language} />;
        return (
          <InlineCode
            resolveInlineToken={resolveInlineToken}
            looksLikeFileReference={looksLikeFileReference}
            onLinkClick={onLinkClick}
            onFileReferenceClick={onFileReferenceClick}
          >
            {text}
          </InlineCode>
        );
      },
      a: ({ href = "", children }: { href?: string; children?: ReactNode }) => (
        <a href={href} onClick={(event) => onLinkClick?.(href, event)} className="rcl-md__link">
          {children}
        </a>
      ),
      blockquote: ({ children }: { children?: ReactNode }) => (
        <blockquote className="rcl-md__blockquote">{children}</blockquote>
      ),
      table: ({ children }: { children?: ReactNode }) => (
        <div className="rcl-md__table-scroll">
          <table>{children}</table>
        </div>
      ),
      th: ({ children }: { children?: ReactNode }) => <th>{children}</th>,
      td: ({ children }: { children?: ReactNode }) => <td>{children}</td>,
    }),
    [looksLikeFileReference, onFileReferenceClick, onLinkClick, onMermaidOpen, resolveInlineToken],
  );
  if (!content) return null;
  const Wrapper = inline ? "span" : "div";
  return (
    <MarkdownErrorBoundary content={content}>
      <Wrapper
        className={`rcl-md__root ${className ?? ""}`}
        style={markdownTokens}
        data-rcl-markdown
      >
        <StyleSheet name="markdown-renderer-0-4-0" css={markdownStyles} />
        <ReactMarkdown remarkPlugins={[remarkGfm, remarkProsePaths]} components={components}>
          {content}
        </ReactMarkdown>
      </Wrapper>
    </MarkdownErrorBoundary>
  );
});

export default MarkdownRenderer;
