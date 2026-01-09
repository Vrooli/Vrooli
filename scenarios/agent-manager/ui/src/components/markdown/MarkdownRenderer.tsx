import { memo, useMemo, type ComponentPropsWithoutRef, type ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "./components/CodeBlock";
import { InlineCode } from "./components/InlineCode";

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
  const components = useMemo(
    () => ({
      code: ({
        inline,
        className: codeClassName,
        children,
        ...props
      }: ComponentPropsWithoutRef<"code"> & { inline?: boolean }) => {
        const codeContent = extractTextContent(children);
        const isInline =
          inline ?? (!codeClassName && !codeContent.includes("\n"));

        if (isInline) {
          return <InlineCode>{children}</InlineCode>;
        }

        return (
          <CodeBlock code={codeContent} className={codeClassName} {...props} />
        );
      },

      pre: ({ children }: { children?: ReactNode }) => <>{children}</>,

      a: ({ href, children, ...props }: ComponentPropsWithoutRef<"a">) => (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="text-primary underline underline-offset-2 hover:text-primary/80"
          {...props}
        >
          {children}
        </a>
      ),

      h1: ({ children }: { children?: ReactNode }) => (
        <h1 className="text-2xl font-semibold mt-6 mb-3 text-foreground">{children}</h1>
      ),
      h2: ({ children }: { children?: ReactNode }) => (
        <h2 className="text-xl font-semibold mt-5 mb-3 text-foreground">{children}</h2>
      ),
      h3: ({ children }: { children?: ReactNode }) => (
        <h3 className="text-lg font-semibold mt-4 mb-2 text-foreground">{children}</h3>
      ),
      h4: ({ children }: { children?: ReactNode }) => (
        <h4 className="text-base font-semibold mt-3 mb-2 text-foreground">{children}</h4>
      ),

      p: ({ children }: { children?: ReactNode }) => (
        <p className="my-2 leading-relaxed text-foreground">{children}</p>
      ),

      ul: ({ children }: { children?: ReactNode }) => (
        <ul className="list-disc list-inside my-2 space-y-1 text-foreground">{children}</ul>
      ),
      ol: ({ children }: { children?: ReactNode }) => (
        <ol className="list-decimal list-inside my-2 space-y-1 text-foreground">{children}</ol>
      ),
      li: ({ children }: { children?: ReactNode }) => (
        <li className="leading-relaxed">{children}</li>
      ),

      blockquote: ({ children }: { children?: ReactNode }) => (
        <blockquote className="border-l-4 border-primary/40 pl-4 my-3 italic text-muted-foreground">
          {children}
        </blockquote>
      ),

      hr: () => <hr className="my-6 border-border" />,

      table: ({ children }: { children?: ReactNode }) => (
        <div className="overflow-x-auto my-4">
          <table className="min-w-full border-collapse border border-border text-sm">
            {children}
          </table>
        </div>
      ),
      thead: ({ children }: { children?: ReactNode }) => (
        <thead className="bg-muted/40">{children}</thead>
      ),
      th: ({ children }: { children?: ReactNode }) => (
        <th className="border border-border px-3 py-2 text-left font-semibold">
          {children}
        </th>
      ),
      td: ({ children }: { children?: ReactNode }) => (
        <td className="border border-border px-3 py-2">{children}</td>
      ),

      strong: ({ children }: { children?: ReactNode }) => (
        <strong className="font-semibold text-foreground">{children}</strong>
      ),
      em: ({ children }: { children?: ReactNode }) => (
        <em className="italic">{children}</em>
      ),

      del: ({ children }: { children?: ReactNode }) => (
        <del className="line-through text-muted-foreground">{children}</del>
      ),
    }),
    []
  );

  if (!content) {
    return null;
  }

  const safeContent = typeof content === "string" ? content : String(content);

  return (
    <div className={className ? `markdown-content ${className}` : "markdown-content"}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {safeContent}
      </ReactMarkdown>
    </div>
  );
}, (prevProps, nextProps) => {
  if (prevProps.isStreaming || nextProps.isStreaming) {
    const prevLen = prevProps.content.length;
    const nextLen = nextProps.content.length;
    if (prevLen === nextLen && prevProps.content === nextProps.content) {
      return true;
    }
    return false;
  }
  return prevProps.content === nextProps.content && prevProps.className === nextProps.className;
});

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
