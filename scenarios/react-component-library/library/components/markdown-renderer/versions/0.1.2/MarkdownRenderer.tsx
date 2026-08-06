import {
  Component,
  type ErrorInfo,
  type MouseEvent,
  type ReactNode,
  useMemo,
} from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

export interface InlineTokenResolution {
  href: string;
  kind?: string;
}
export interface MarkdownRendererProps {
  content: string;
  className?: string;
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

/**
 * @libraryId react-component-library:markdown-renderer
 * @displayName Markdown Renderer
 * @description GFM markdown renderer with routed inline-token, file, link, and Mermaid seams.
 * @version 0.1.2
 * @tags ["markdown","rendering","gfm"]
 * @deps {"react":"^18","react-markdown":"^10.1.0","remark-gfm":"^4.0.1"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */

export function MarkdownRenderer({
  content,
  className,
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
        if (codeClass === "language-mermaid")
          return (
            <button
              type="button"
              className="block w-full rounded border border-slate-700 bg-slate-950 p-3 text-left font-mono text-xs text-slate-200"
              onClick={() => onMermaidOpen?.(text)}
            >
              {text}
            </button>
          );
        if (codeClass)
          return (
            <pre className="my-3 overflow-x-auto rounded border border-slate-700 bg-slate-950 p-3 text-sm text-slate-100">
              <code>{text}</code>
            </pre>
          );
        const resolution = resolveInlineToken?.(text);
        if (resolution)
          return (
            <a
              href={resolution.href}
              data-entity-ref={
                resolution.kind === "entity" ? "true" : undefined
              }
              onClick={(event) => onLinkClick?.(resolution.href, event)}
              className="rounded bg-cyan-500/15 px-1 py-0.5 font-mono text-cyan-200 underline"
            >
              {text}
            </a>
          );
        const file = looksLikeFileReference?.(text);
        if (file)
          return (
            <button
              type="button"
              onClick={() => onFileReferenceClick?.(text)}
              className="rounded bg-slate-800 px-1 py-0.5 font-mono text-cyan-200"
            >
              {text}
            </button>
          );
        return (
          <code className="rounded bg-slate-800 px-1 py-0.5 font-mono text-slate-200">
            {text}
          </code>
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
  return (
    <MarkdownErrorBoundary content={content}>
      <div className={className}>
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
          {content}
        </ReactMarkdown>
      </div>
    </MarkdownErrorBoundary>
  );
}
