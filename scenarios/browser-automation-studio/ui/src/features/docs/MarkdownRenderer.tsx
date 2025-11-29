import ReactMarkdown from "react-markdown";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { oneDark } from "react-syntax-highlighter/dist/esm/styles/prism";
import type { Components } from "react-markdown";

interface MarkdownRendererProps {
  content: string;
  className?: string;
}

const components: Components = {
  code({ className, children, ...props }) {
    const match = /language-(\w+)/.exec(className || "");
    const isInline = !match && !className;

    if (isInline) {
      return (
        <code
          className="px-1.5 py-0.5 bg-gray-800 text-amber-300 rounded text-sm font-mono"
          {...props}
        >
          {children}
        </code>
      );
    }

    return (
      <SyntaxHighlighter
        style={oneDark}
        language={match?.[1] || "text"}
        PreTag="div"
        customStyle={{
          margin: 0,
          borderRadius: "0.5rem",
          fontSize: "0.875rem",
        }}
      >
        {String(children).replace(/\n$/, "")}
      </SyntaxHighlighter>
    );
  },
  table({ children }) {
    return (
      <div className="overflow-x-auto my-4">
        <table className="min-w-full border-collapse text-sm">
          {children}
        </table>
      </div>
    );
  },
  thead({ children }) {
    return <thead className="bg-gray-800/50">{children}</thead>;
  },
  th({ children }) {
    return (
      <th className="px-3 py-2 text-left text-xs font-semibold text-gray-300 uppercase tracking-wide border-b border-gray-700">
        {children}
      </th>
    );
  },
  td({ children }) {
    return (
      <td className="px-3 py-2 text-gray-300 border-b border-gray-800">
        {children}
      </td>
    );
  },
  h1({ children }) {
    return (
      <h1 className="text-2xl font-bold text-white mb-4 mt-6 first:mt-0">
        {children}
      </h1>
    );
  },
  h2({ children }) {
    return (
      <h2 className="text-xl font-semibold text-white mb-3 mt-6 pb-2 border-b border-gray-800">
        {children}
      </h2>
    );
  },
  h3({ children }) {
    return (
      <h3 className="text-lg font-semibold text-gray-200 mb-2 mt-4">
        {children}
      </h3>
    );
  },
  h4({ children }) {
    return (
      <h4 className="text-base font-semibold text-gray-300 mb-2 mt-3">
        {children}
      </h4>
    );
  },
  p({ children }) {
    return <p className="text-gray-300 mb-3 leading-relaxed">{children}</p>;
  },
  ul({ children }) {
    return <ul className="list-disc list-inside mb-3 space-y-1 text-gray-300">{children}</ul>;
  },
  ol({ children }) {
    return <ol className="list-decimal list-inside mb-3 space-y-1 text-gray-300">{children}</ol>;
  },
  li({ children }) {
    return <li className="text-gray-300">{children}</li>;
  },
  blockquote({ children }) {
    return (
      <blockquote className="border-l-4 border-flow-accent/50 pl-4 py-2 my-4 bg-flow-accent/5 rounded-r-lg text-gray-400 italic">
        {children}
      </blockquote>
    );
  },
  a({ href, children }) {
    return (
      <a
        href={href}
        className="text-flow-accent hover:text-blue-400 underline underline-offset-2"
        target={href?.startsWith("http") ? "_blank" : undefined}
        rel={href?.startsWith("http") ? "noopener noreferrer" : undefined}
      >
        {children}
      </a>
    );
  },
  hr() {
    return <hr className="my-6 border-gray-800" />;
  },
  strong({ children }) {
    return <strong className="font-semibold text-white">{children}</strong>;
  },
  em({ children }) {
    return <em className="italic text-gray-400">{children}</em>;
  },
};

export function MarkdownRenderer({ content, className = "" }: MarkdownRendererProps) {
  return (
    <div className={`prose prose-invert max-w-none ${className}`}>
      <ReactMarkdown components={components}>{content}</ReactMarkdown>
    </div>
  );
}
