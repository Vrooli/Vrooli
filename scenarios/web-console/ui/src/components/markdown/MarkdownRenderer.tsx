import { Component, memo, useMemo, type ComponentPropsWithoutRef, type MouseEvent, type ReactNode, type ErrorInfo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "./components/CodeBlock";
import { InlineCode } from "./components/InlineCode";
import { MermaidDiagram } from "./components/MermaidDiagram";
import { isExternalHref } from "../../lib/fileReferences";

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
  onLinkClick?: (href: string, event: MouseEvent<HTMLAnchorElement>) => void;
  /** Open a file in the preview dialog when an inline-code chip looks like a path. */
  onFileReferenceClick?: (path: string) => void;
  /** Open a Mermaid diagram in the full-screen zoomable viewer. */
  onMermaidOpen?: (code: string) => void;
}

/** Renders markdown content with syntax highlighting, mermaid diagrams, and GFM support. */
export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
  className,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
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

        if (isInline) return <InlineCode onFileReferenceClick={onFileReferenceClick}>{children}</InlineCode>;
        if (codeClassName === "language-mermaid") return <MermaidDiagram code={codeContent} onOpenFullscreen={onMermaidOpen} />;
        return <CodeBlock code={codeContent} className={codeClassName} {...props} />;
      },

      pre: ({ children }: { children?: ReactNode }) => <>{children}</>,

      a: ({ href, children, ...props }: ComponentPropsWithoutRef<"a">) => {
        const safeHref = href ?? "";
        const external = isExternalHref(safeHref);
        return (
          <a
            href={safeHref}
            target={external ? "_blank" : undefined}
            rel={external ? "noopener noreferrer" : undefined}
            className="text-wc-accent underline underline-offset-2 hover:text-wc-accent/80"
            onClick={(event) => onLinkClick?.(safeHref, event)}
            {...props}
          >
            {children}
          </a>
        );
      },

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
        <blockquote className="border-l-4 border-wc-accent ps-4 my-3 italic text-wc-text-secondary">
          {children}
        </blockquote>
      ),

      hr: () => <hr className="my-6 border-wc-default" />,

      table: ({ children }: { children?: ReactNode }) => (
        <div className="overflow-x-auto my-4">
          <table className="w-auto border-collapse border border-wc-default">{children}</table>
        </div>
      ),
      thead: ({ children }: { children?: ReactNode }) => (
        <thead className="bg-wc-surface-raised">{children}</thead>
      ),
      th: ({ children }: { children?: ReactNode }) => (
        <th className="border border-wc-default px-4 py-2 text-start font-semibold align-top min-w-[8rem]">{children}</th>
      ),
      td: ({ children }: { children?: ReactNode }) => (
        <td className="border border-wc-default px-4 py-2 align-top min-w-[8rem] [overflow-wrap:anywhere]">{children}</td>
      ),

      strong: ({ children }: { children?: ReactNode }) => (
        <strong className="font-semibold text-wc-text-primary">{children}</strong>
      ),
      em: ({ children }: { children?: ReactNode }) => <em className="italic">{children}</em>,

      del: ({ children }: { children?: ReactNode }) => (
        <del className="line-through text-wc-text-faint">{children}</del>
      ),
    }),
    [onLinkClick, onFileReferenceClick, onMermaidOpen],
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
        {rendered}
      </div>
    </MarkdownErrorBoundary>
  );
}, (prevProps, nextProps) => {
  return (
    prevProps.content === nextProps.content &&
    prevProps.className === nextProps.className &&
    prevProps.onLinkClick === nextProps.onLinkClick &&
    prevProps.onFileReferenceClick === nextProps.onFileReferenceClick &&
    prevProps.onMermaidOpen === nextProps.onMermaidOpen
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
