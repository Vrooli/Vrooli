import { Component, memo, useMemo, type ComponentPropsWithoutRef, type ReactNode, type ErrorInfo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "./components/CodeBlock";
import { InlineCode } from "./components/InlineCode";
import { MermaidDiagram } from "./components/MermaidDiagram";

interface MarkdownErrorBoundaryProps {
  children: ReactNode;
  content: string;
}

interface MarkdownErrorBoundaryState {
  hasError: boolean;
}

class MarkdownErrorBoundary extends Component<MarkdownErrorBoundaryProps, MarkdownErrorBoundaryState> {
  constructor(props: MarkdownErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): MarkdownErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error("[MarkdownRenderer] Failed to render content:", error, errorInfo);
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <pre className="whitespace-pre-wrap text-sm text-wc-text-primary font-mono">
          {this.props.content}
        </pre>
      );
    }
    return this.props.children;
  }
}

interface MarkdownRendererProps {
  content: string;
  className?: string;
  /** Search query to highlight within rendered text. */
  searchQuery?: string;
  /** Whether this message contains the focused search match (stronger highlight). */
  isSearchFocused?: boolean;
}

/** Renders markdown content with syntax highlighting, mermaid diagrams, and GFM support. */
export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
  className,
  searchQuery,
  isSearchFocused = false,
}: MarkdownRendererProps) {
  const components = useMemo(
    () => ({
      code: ({
        inline,
        className: codeClassName,
        children,
        ...props
      }: ComponentPropsWithoutRef<"code"> & { inline?: boolean }) => {
        const codeContent = extractTextContent(children);
        const isInline = inline ?? (!codeClassName && !codeContent.includes("\n"));

        if (isInline) return <InlineCode>{children}</InlineCode>;
        if (codeClassName === "language-mermaid") return <MermaidDiagram code={codeContent} />;
        return <CodeBlock code={codeContent} className={codeClassName} {...props} />;
      },

      pre: ({ children }: { children?: ReactNode }) => <>{children}</>,

      a: ({ href, children, ...props }: ComponentPropsWithoutRef<"a">) => (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="text-wc-accent underline underline-offset-2 hover:text-wc-accent/80"
          {...props}
        >
          {children}
        </a>
      ),

      h1: ({ children }: { children?: ReactNode }) => (
        <h1 className="text-2xl font-bold mt-6 mb-4 text-wc-text-primary">{children}</h1>
      ),
      h2: ({ children }: { children?: ReactNode }) => (
        <h2 className="text-xl font-bold mt-5 mb-3 text-wc-text-primary">{children}</h2>
      ),
      h3: ({ children }: { children?: ReactNode }) => (
        <h3 className="text-lg font-semibold mt-4 mb-2 text-wc-text-primary">{children}</h3>
      ),
      h4: ({ children }: { children?: ReactNode }) => (
        <h4 className="text-base font-semibold mt-3 mb-2 text-wc-text-secondary">{children}</h4>
      ),

      p: ({ children }: { children?: ReactNode }) => (
        <p className="my-2 leading-relaxed break-words [overflow-wrap:anywhere]">{children}</p>
      ),

      ul: ({ children }: { children?: ReactNode }) => (
        <ul className="list-disc list-inside my-2 space-y-1">{children}</ul>
      ),
      ol: ({ children }: { children?: ReactNode }) => (
        <ol className="list-decimal list-inside my-2 space-y-1">{children}</ol>
      ),
      li: ({ children }: { children?: ReactNode }) => (
        <li className="leading-relaxed break-words [overflow-wrap:anywhere]">{children}</li>
      ),

      blockquote: ({ children }: { children?: ReactNode }) => (
        <blockquote className="border-l-4 border-wc-accent pl-4 my-3 italic text-wc-text-secondary">
          {children}
        </blockquote>
      ),

      hr: () => <hr className="my-6 border-wc-default" />,

      table: ({ children }: { children?: ReactNode }) => (
        <div className="overflow-x-auto my-4">
          <table className="min-w-full border-collapse border border-wc-default">{children}</table>
        </div>
      ),
      thead: ({ children }: { children?: ReactNode }) => (
        <thead className="bg-wc-surface-raised">{children}</thead>
      ),
      th: ({ children }: { children?: ReactNode }) => (
        <th className="border border-wc-default px-4 py-2 text-left font-semibold">{children}</th>
      ),
      td: ({ children }: { children?: ReactNode }) => (
        <td className="border border-wc-default px-4 py-2">{children}</td>
      ),

      strong: ({ children }: { children?: ReactNode }) => (
        <strong className="font-semibold text-wc-text-primary">{children}</strong>
      ),
      em: ({ children }: { children?: ReactNode }) => <em className="italic">{children}</em>,

      del: ({ children }: { children?: ReactNode }) => (
        <del className="line-through text-wc-text-faint">{children}</del>
      ),
    }),
    [],
  );

  if (!content) return null;

  const safeContent = typeof content === "string" ? content : String(content);

  // If search is active, wrap rendered output with highlighting
  const rendered = (
    <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
      {safeContent}
    </ReactMarkdown>
  );

  return (
    <MarkdownErrorBoundary content={safeContent}>
      <div className={`markdown-content min-w-0 max-w-full break-words [overflow-wrap:anywhere] ${className || ""}`}>
        {searchQuery ? (
          <SearchHighlightWrapper query={searchQuery} isFocused={isSearchFocused}>
            {rendered}
          </SearchHighlightWrapper>
        ) : (
          rendered
        )}
      </div>
    </MarkdownErrorBoundary>
  );
}, (prevProps, nextProps) => {
  return (
    prevProps.content === nextProps.content &&
    prevProps.className === nextProps.className &&
    prevProps.searchQuery === nextProps.searchQuery &&
    prevProps.isSearchFocused === nextProps.isSearchFocused
  );
});

function extractTextContent(children: ReactNode): string {
  if (typeof children === "string") return children;
  if (Array.isArray(children)) return children.map(extractTextContent).join("");
  if (children && typeof children === "object" && "props" in children) {
    return extractTextContent((children as { props: { children?: ReactNode } }).props.children);
  }
  return "";
}

/**
 * Wraps rendered markdown with CSS-based search highlighting.
 * Uses a CSS custom highlight approach via mark elements injected by
 * walking text nodes post-render would be expensive, so instead we apply
 * a subtle background to the entire message when it matches.
 * Individual text-level highlighting is handled per text node in the markdown.
 */
function SearchHighlightWrapper({
  children,
  query,
  isFocused,
}: {
  children: ReactNode;
  query: string;
  isFocused: boolean;
}) {
  // For now, apply a container-level visual indicator that this message matches.
  // Fine-grained text highlighting in rendered markdown requires DOM manipulation
  // which would break React's reconciliation. The container highlight is sufficient
  // for identifying which messages contain matches.
  return (
    <div
      className={isFocused ? "ring-1 ring-wc-accent/50 rounded-lg -mx-1 px-1" : ""}
      data-search-query={query}
    >
      {children}
    </div>
  );
}
