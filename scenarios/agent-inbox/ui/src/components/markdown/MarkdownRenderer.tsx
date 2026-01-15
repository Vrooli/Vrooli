import { Component, memo, useMemo, type ComponentPropsWithoutRef, type ReactNode, type ErrorInfo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "./components/CodeBlock";
import { InlineCode } from "./components/InlineCode";
import { LinkWithPreview } from "./components/LinkWithPreview";

/**
 * Lightweight error boundary specifically for markdown rendering.
 * Shows a graceful fallback when markdown parsing fails instead of crashing.
 */
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
      // Fallback: render as plain text with preserved whitespace
      return (
        <pre className="whitespace-pre-wrap text-sm text-slate-300 font-mono">
          {this.props.content}
        </pre>
      );
    }
    return this.props.children;
  }
}

interface MarkdownRendererProps {
  /** The markdown content to render */
  content: string;
  /** Additional CSS classes */
  className?: string;
  /** Whether content is still streaming (affects rendering strategy) */
  isStreaming?: boolean;
}

/**
 * Renders markdown content with syntax highlighting and custom components.
 * Memoized to prevent unnecessary re-parses.
 */
export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
  className,
  isStreaming = false,
}: MarkdownRendererProps) {
  // Memoize the components object to prevent recreation
  const components = useMemo(
    () => ({
      // Code blocks and inline code
      code: ({
        inline,
        className: codeClassName,
        children,
        ...props
      }: ComponentPropsWithoutRef<"code"> & { inline?: boolean }) => {
        // Extract code content as string
        const codeContent = extractTextContent(children);
        const isInline =
          inline ?? (!codeClassName && !codeContent.includes("\n"));

        // Inline code
        if (isInline) {
          return <InlineCode>{children}</InlineCode>;
        }

        // Fenced code block
        return (
          <CodeBlock code={codeContent} className={codeClassName} {...props} />
        );
      },

      // Pre tag - we handle this in CodeBlock, so just pass through
      pre: ({ children }: { children?: ReactNode }) => <>{children}</>,

      // Links - with hover preview
      a: ({ href, children, ...props }: ComponentPropsWithoutRef<"a">) => (
        <LinkWithPreview href={href} {...props}>
          {children}
        </LinkWithPreview>
      ),

      // Headings with proper styling
      h1: ({ children }: { children?: ReactNode }) => (
        <h1 className="text-2xl font-bold mt-6 mb-4 text-slate-100">{children}</h1>
      ),
      h2: ({ children }: { children?: ReactNode }) => (
        <h2 className="text-xl font-bold mt-5 mb-3 text-slate-100">{children}</h2>
      ),
      h3: ({ children }: { children?: ReactNode }) => (
        <h3 className="text-lg font-semibold mt-4 mb-2 text-slate-100">{children}</h3>
      ),
      h4: ({ children }: { children?: ReactNode }) => (
        <h4 className="text-base font-semibold mt-3 mb-2 text-slate-200">{children}</h4>
      ),

      // Paragraphs
      p: ({ children }: { children?: ReactNode }) => (
        <p className="my-2 leading-relaxed">{children}</p>
      ),

      // Lists
      ul: ({ children }: { children?: ReactNode }) => (
        <ul className="list-disc list-inside my-2 space-y-1">{children}</ul>
      ),
      ol: ({ children }: { children?: ReactNode }) => (
        <ol className="list-decimal list-inside my-2 space-y-1">{children}</ol>
      ),
      li: ({ children }: { children?: ReactNode }) => (
        <li className="leading-relaxed">{children}</li>
      ),

      // Blockquotes
      blockquote: ({ children }: { children?: ReactNode }) => (
        <blockquote className="border-l-4 border-indigo-500 pl-4 my-3 italic text-slate-300">
          {children}
        </blockquote>
      ),

      // Horizontal rule
      hr: () => <hr className="my-6 border-slate-600" />,

      // Tables
      table: ({ children }: { children?: ReactNode }) => (
        <div className="overflow-x-auto my-4">
          <table className="min-w-full border-collapse border border-slate-600">
            {children}
          </table>
        </div>
      ),
      thead: ({ children }: { children?: ReactNode }) => (
        <thead className="bg-slate-700">{children}</thead>
      ),
      th: ({ children }: { children?: ReactNode }) => (
        <th className="border border-slate-600 px-4 py-2 text-left font-semibold">
          {children}
        </th>
      ),
      td: ({ children }: { children?: ReactNode }) => (
        <td className="border border-slate-600 px-4 py-2">{children}</td>
      ),

      // Strong and emphasis
      strong: ({ children }: { children?: ReactNode }) => (
        <strong className="font-semibold text-slate-100">{children}</strong>
      ),
      em: ({ children }: { children?: ReactNode }) => (
        <em className="italic">{children}</em>
      ),

      // Strikethrough (GFM)
      del: ({ children }: { children?: ReactNode }) => (
        <del className="line-through text-slate-400">{children}</del>
      ),
    }),
    []
  );

  // Don't render if content is empty or not a string
  if (!content) {
    return null;
  }

  // Defensive: ensure content is a string
  const safeContent = typeof content === "string" ? content : String(content);

  return (
    <MarkdownErrorBoundary content={safeContent}>
      <div className={`markdown-content ${className || ""}`}>
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
          {safeContent}
        </ReactMarkdown>
      </div>
    </MarkdownErrorBoundary>
  );
}, (prevProps, nextProps) => {
  // Custom comparison for memoization
  // For streaming content, compare length + last chars to detect meaningful changes
  if (prevProps.isStreaming || nextProps.isStreaming) {
    const prevLen = prevProps.content.length;
    const nextLen = nextProps.content.length;
    if (prevLen === nextLen && prevProps.content === nextProps.content) {
      return true;
    }
    return false;
  }
  // For non-streaming, simple equality check
  return prevProps.content === nextProps.content && prevProps.className === nextProps.className;
});

/**
 * Extracts text content from React children.
 * Handles both string children and nested elements.
 */
function extractTextContent(children: ReactNode): string {
  if (typeof children === "string") {
    return children;
  }
  if (Array.isArray(children)) {
    return children.map(extractTextContent).join("");
  }
  if (children && typeof children === "object" && "props" in children) {
    return extractTextContent((children as { props: { children?: ReactNode } }).props.children);
  }
  return "";
}
