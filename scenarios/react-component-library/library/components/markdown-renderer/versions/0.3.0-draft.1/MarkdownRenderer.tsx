/**
 * @libraryId react-component-library:markdown-renderer
 * @displayName Markdown Renderer
 * @description GFM markdown renderer with routed inline-token, file, link, and Mermaid seams.
 * @version 0.3.0-draft.1
 * @tags ["markdown","rendering","gfm"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */

/**
 * @libraryId react-component-library:markdown-renderer
 * @displayName Markdown Renderer
 * @description GFM renderer composed from reusable code, inline-token, and Mermaid primitives.
 * @version 0.3.0
 * @tags ["markdown","rendering","gfm"]
 * @deps {"react":"^18","react-markdown":"^10.1.0","remark-gfm":"^4.0.1","shiki":"^4.3.1","mermaid":"^11.4.0"}
 */

import {
  Component,
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

export type { InlineTokenResolution } from "./InlineCode";
export { CodeBlock } from "./CodeBlock";
export { InlineCode } from "./InlineCode";
export { MermaidDiagram } from "./MermaidDiagram";
export {
  normalizeCodeLanguage,
  languageLabel,
  remarkProsePaths,
} from "./languageDetection";
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
      <pre className="whitespace-pre-wrap font-mono text-sm text-slate-200">
        {this.props.content}
      </pre>
    ) : (
      this.props.children
    );
  }
}

export function MarkdownRenderer({
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
      code: ({
        children,
        className: codeClass,
      }: {
        children?: ReactNode;
        className?: string;
      }) => {
        const text = String(children ?? "").replace(/\n$/, "");
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
        <a
          href={href}
          onClick={(event) => onLinkClick?.(href, event)}
          className="text-cyan-300 underline underline-offset-2"
        >
          {children}
        </a>
      ),
      blockquote: ({ children }: { children?: ReactNode }) => (
        <blockquote className="my-3 border-l-2 border-cyan-500 pl-3 italic text-slate-300">
          {children}
        </blockquote>
      ),
      table: ({ children }: { children?: ReactNode }) => (
        <div className="my-3 overflow-x-auto">
          <table className="border-collapse text-sm">{children}</table>
        </div>
      ),
      th: ({ children }: { children?: ReactNode }) => (
        <th className="border border-slate-700 px-2 py-1 text-left">
          {children}
        </th>
      ),
      td: ({ children }: { children?: ReactNode }) => (
        <td className="border border-slate-700 px-2 py-1">{children}</td>
      ),
    }),
    [
      looksLikeFileReference,
      onFileReferenceClick,
      onLinkClick,
      onMermaidOpen,
      resolveInlineToken,
    ],
  );
  if (!content) return null;
  const Wrapper = inline ? "span" : "div";
  return (
    <MarkdownErrorBoundary content={content}>
      <Wrapper className={className}>
        <ReactMarkdown
          remarkPlugins={[remarkGfm, remarkProsePaths]}
          components={components}
        >
          {content}
        </ReactMarkdown>
      </Wrapper>
    </MarkdownErrorBoundary>
  );
}
