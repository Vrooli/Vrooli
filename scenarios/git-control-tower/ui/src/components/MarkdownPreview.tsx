import { memo, useMemo, type ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { MermaidDiagram } from "./MermaidDiagram";

interface MarkdownPreviewProps {
  content: string;
}

function extractTextContent(children: ReactNode): string {
  if (typeof children === "string") {
    return children;
  }
  if (Array.isArray(children)) {
    return children.map(extractTextContent).join("");
  }
  if (children && typeof children === "object" && "props" in children) {
    return extractTextContent(
      (children as { props: { children?: ReactNode } }).props.children
    );
  }
  return "";
}

export const MarkdownPreview = memo(function MarkdownPreview({
  content,
}: MarkdownPreviewProps) {
  const components = useMemo(
    () => ({
      // Headers
      h1: ({ children }: { children?: ReactNode }) => (
        <h1 className="text-2xl font-bold text-white mb-4 mt-6 first:mt-0 border-b border-slate-700 pb-2">
          {children}
        </h1>
      ),
      h2: ({ children }: { children?: ReactNode }) => (
        <h2 className="text-xl font-semibold text-white mb-3 mt-5 first:mt-0 border-b border-slate-800 pb-2">
          {children}
        </h2>
      ),
      h3: ({ children }: { children?: ReactNode }) => (
        <h3 className="text-lg font-semibold text-white mb-2 mt-4 first:mt-0">
          {children}
        </h3>
      ),
      h4: ({ children }: { children?: ReactNode }) => (
        <h4 className="text-base font-semibold text-white mb-2 mt-3 first:mt-0">
          {children}
        </h4>
      ),
      // Paragraphs and text
      p: ({ children }: { children?: ReactNode }) => (
        <p className="mb-4 leading-relaxed">{children}</p>
      ),
      // Links
      a: ({ href, children }: { href?: string; children?: ReactNode }) => (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="text-blue-400 hover:text-blue-300 underline"
        >
          {children}
        </a>
      ),
      // Lists
      ul: ({ children }: { children?: ReactNode }) => (
        <ul className="list-disc list-inside mb-4 space-y-1">{children}</ul>
      ),
      ol: ({ children }: { children?: ReactNode }) => (
        <ol className="list-decimal list-inside mb-4 space-y-1">{children}</ol>
      ),
      li: ({ children }: { children?: ReactNode }) => (
        <li className="text-slate-300">{children}</li>
      ),
      // Code
      code: ({
        className,
        children,
      }: {
        className?: string;
        children?: ReactNode;
      }) => {
        if (className === "language-mermaid") {
          return <MermaidDiagram code={extractTextContent(children)} />;
        }
        const isBlock = className?.includes("language-");
        if (isBlock) {
          return (
            <code className="block bg-slate-900 rounded-md p-4 text-sm font-mono text-slate-300 overflow-x-auto mb-4">
              {children}
            </code>
          );
        }
        return (
          <code className="bg-slate-800 text-emerald-300 px-1.5 py-0.5 rounded text-sm font-mono">
            {children}
          </code>
        );
      },
      pre: ({ children }: { children?: ReactNode }) => (
        <pre className="bg-slate-900 rounded-md overflow-x-auto mb-4">
          {children}
        </pre>
      ),
      // Blockquotes
      blockquote: ({ children }: { children?: ReactNode }) => (
        <blockquote className="border-l-4 border-slate-600 pl-4 italic text-slate-400 my-4">
          {children}
        </blockquote>
      ),
      // Tables
      table: ({ children }: { children?: ReactNode }) => (
        <div className="overflow-x-auto mb-4">
          <table className="min-w-full border border-slate-700 rounded">
            {children}
          </table>
        </div>
      ),
      thead: ({ children }: { children?: ReactNode }) => (
        <thead className="bg-slate-800">{children}</thead>
      ),
      tbody: ({ children }: { children?: ReactNode }) => (
        <tbody className="divide-y divide-slate-700">{children}</tbody>
      ),
      tr: ({ children }: { children?: ReactNode }) => <tr>{children}</tr>,
      th: ({ children }: { children?: ReactNode }) => (
        <th className="px-4 py-2 text-left text-sm font-semibold text-slate-200">
          {children}
        </th>
      ),
      td: ({ children }: { children?: ReactNode }) => (
        <td className="px-4 py-2 text-sm text-slate-300">{children}</td>
      ),
      // Horizontal rule
      hr: () => <hr className="border-slate-700 my-6" />,
      // Images
      img: ({ src, alt }: { src?: string; alt?: string }) => (
        <img
          src={src}
          alt={alt || ""}
          className="max-w-full h-auto rounded my-4"
        />
      ),
      // Strong and emphasis
      strong: ({ children }: { children?: ReactNode }) => (
        <strong className="font-semibold text-white">{children}</strong>
      ),
      em: ({ children }: { children?: ReactNode }) => (
        <em className="italic text-slate-300">{children}</em>
      ),
    }),
    []
  );

  return (
    <div
      className="p-6 max-w-none text-slate-200 markdown-preview"
      data-testid="markdown-preview"
    >
      <ReactMarkdown components={components} remarkPlugins={[remarkGfm]}>
        {content}
      </ReactMarkdown>
    </div>
  );
});
