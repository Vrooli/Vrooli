import { memo, type ComponentPropsWithoutRef, type ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

interface MarkdownRendererProps {
  content: string;
  className?: string;
}

export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
  className,
}: MarkdownRendererProps) {
  return (
    <div
      className={`prose-doc min-w-0 max-w-full break-words [overflow-wrap:anywhere] ${className ?? ""}`}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {content}
      </ReactMarkdown>
    </div>
  );
});

const components = {
  h1: ({ children }: { children?: ReactNode }) => (
    <h1 className="mt-6 mb-4 text-2xl font-bold text-app-foreground">{children}</h1>
  ),
  h2: ({ children }: { children?: ReactNode }) => (
    <h2 className="mt-5 mb-3 text-xl font-bold text-app-foreground">{children}</h2>
  ),
  h3: ({ children }: { children?: ReactNode }) => (
    <h3 className="mt-4 mb-2 text-lg font-semibold text-app-foreground">{children}</h3>
  ),
  h4: ({ children }: { children?: ReactNode }) => (
    <h4 className="mt-3 mb-2 text-base font-semibold text-app-foreground">{children}</h4>
  ),
  p: ({ children }: { children?: ReactNode }) => (
    <p className="my-2 leading-relaxed text-app-foreground">{children}</p>
  ),
  ul: ({ children }: { children?: ReactNode }) => (
    <ul className="my-2 list-disc space-y-1 ps-6 text-app-foreground">{children}</ul>
  ),
  ol: ({ children }: { children?: ReactNode }) => (
    <ol className="my-2 list-decimal space-y-1 ps-6 text-app-foreground">{children}</ol>
  ),
  li: ({ children }: { children?: ReactNode }) => (
    <li className="leading-relaxed">{children}</li>
  ),
  blockquote: ({ children }: { children?: ReactNode }) => (
    <blockquote className="my-3 border-l-4 border-app-border ps-4 italic text-app-muted-foreground">
      {children}
    </blockquote>
  ),
  hr: () => <hr className="my-6 border-app-border" />,
  a: ({ href, children, ...props }: ComponentPropsWithoutRef<"a">) => {
    const safeHref = href ?? "";
    const external = /^(https?:)?\/\//.test(safeHref) || safeHref.startsWith("mailto:");
    return (
      <a
        href={safeHref}
        target={external ? "_blank" : undefined}
        rel={external ? "noopener noreferrer" : undefined}
        className="text-app-primary underline underline-offset-2 hover:opacity-80"
        {...props}
      >
        {children}
      </a>
    );
  },
  table: ({ children }: { children?: ReactNode }) => (
    <div className="my-4 overflow-x-auto">
      <table className="w-auto border-collapse border border-app-border">{children}</table>
    </div>
  ),
  thead: ({ children }: { children?: ReactNode }) => (
    <thead className="bg-app-surface-muted/60">{children}</thead>
  ),
  th: ({ children }: { children?: ReactNode }) => (
    <th className="border border-app-border px-4 py-2 text-start font-semibold align-top">{children}</th>
  ),
  td: ({ children }: { children?: ReactNode }) => (
    <td className="border border-app-border px-4 py-2 align-top">{children}</td>
  ),
  strong: ({ children }: { children?: ReactNode }) => (
    <strong className="font-semibold text-app-foreground">{children}</strong>
  ),
  em: ({ children }: { children?: ReactNode }) => <em className="italic">{children}</em>,
  del: ({ children }: { children?: ReactNode }) => (
    <del className="text-app-muted-foreground line-through">{children}</del>
  ),
  code: ({
    inline,
    className,
    children,
    ...props
  }: ComponentPropsWithoutRef<"code"> & { inline?: boolean }) => {
    const text = extractText(children);
    const looksInline = inline ?? (!className && !text.includes("\n"));
    if (looksInline) {
      return (
        <code className="rounded bg-app-surface-muted px-1.5 py-0.5 font-mono text-[0.875em] text-app-foreground">
          {children}
        </code>
      );
    }
    return (
      <code
        className={`block overflow-x-auto rounded-control border border-app-border bg-app-surface-muted/60 p-3 font-mono text-xs leading-relaxed text-app-foreground ${className ?? ""}`}
        {...props}
      >
        {children}
      </code>
    );
  },
  pre: ({ children }: { children?: ReactNode }) => <pre className="my-3">{children}</pre>,
};

function extractText(node: ReactNode): string {
  if (typeof node === "string") return node;
  if (typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(extractText).join("");
  if (node && typeof node === "object" && "props" in node) {
    return extractText((node as { props: { children?: ReactNode } }).props.children);
  }
  return "";
}
