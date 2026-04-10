import {
  Component,
  memo,
  useMemo,
  type ComponentPropsWithoutRef,
  type ErrorInfo,
  type ReactNode,
} from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { CodeBlock } from './components/CodeBlock';
import { MermaidDiagram } from './components/MermaidDiagram';
import { InlineCode } from './components/InlineCode';

interface MarkdownErrorBoundaryProps {
  children: ReactNode;
  content: string;
}

interface MarkdownErrorBoundaryState {
  hasError: boolean;
}

class MarkdownErrorBoundary extends Component<
  MarkdownErrorBoundaryProps,
  MarkdownErrorBoundaryState
> {
  constructor(props: MarkdownErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): MarkdownErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error('[MarkdownRenderer] Failed to render content:', error, errorInfo);
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <pre className="whitespace-pre-wrap font-mono text-sm text-slate-300">
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
}

export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
  className,
}: MarkdownRendererProps) {
  const components = useMemo(
    () => ({
      code: ({
        inline,
        className: codeClassName,
        children,
        ...props
      }: ComponentPropsWithoutRef<'code'> & { inline?: boolean }) => {
        const codeContent = extractTextContent(children);
        const isInline = inline ?? (!codeClassName && !codeContent.includes('\n'));

        if (isInline) {
          return <InlineCode>{children}</InlineCode>;
        }

        if (codeClassName === 'language-mermaid') {
          return <MermaidDiagram code={codeContent} />;
        }

        return <CodeBlock code={codeContent} className={codeClassName} {...props} />;
      },
      pre: ({ children }: { children?: ReactNode }) => <>{children}</>,
      h1: ({ children }: { children?: ReactNode }) => (
        <h1 className="mb-4 mt-6 text-2xl font-bold text-slate-100">{children}</h1>
      ),
      h2: ({ children }: { children?: ReactNode }) => (
        <h2 className="mb-3 mt-5 text-xl font-bold text-slate-100">{children}</h2>
      ),
      h3: ({ children }: { children?: ReactNode }) => (
        <h3 className="mb-2 mt-4 text-lg font-semibold text-slate-100">{children}</h3>
      ),
      p: ({ children }: { children?: ReactNode }) => (
        <p className="my-2 leading-relaxed text-slate-200">{children}</p>
      ),
      ul: ({ children }: { children?: ReactNode }) => (
        <ul className="my-2 list-inside list-disc space-y-1 text-slate-200">{children}</ul>
      ),
      ol: ({ children }: { children?: ReactNode }) => (
        <ol className="my-2 list-inside list-decimal space-y-1 text-slate-200">{children}</ol>
      ),
      li: ({ children }: { children?: ReactNode }) => (
        <li className="leading-relaxed">{children}</li>
      ),
      blockquote: ({ children }: { children?: ReactNode }) => (
        <blockquote className="my-3 border-l-4 border-blue-400 pl-4 italic text-slate-300">
          {children}
        </blockquote>
      ),
      hr: () => <hr className="my-6 border-white/15" />,
      table: ({ children }: { children?: ReactNode }) => (
        <div className="my-4 overflow-x-auto">
          <table className="min-w-full border-collapse border border-white/15">{children}</table>
        </div>
      ),
      thead: ({ children }: { children?: ReactNode }) => (
        <thead className="bg-white/5">{children}</thead>
      ),
      th: ({ children }: { children?: ReactNode }) => (
        <th className="border border-white/15 px-4 py-2 text-left font-semibold text-slate-100">
          {children}
        </th>
      ),
      td: ({ children }: { children?: ReactNode }) => (
        <td className="border border-white/15 px-4 py-2 text-slate-200">{children}</td>
      ),
      strong: ({ children }: { children?: ReactNode }) => (
        <strong className="font-semibold text-slate-50">{children}</strong>
      ),
      em: ({ children }: { children?: ReactNode }) => <em className="italic">{children}</em>,
      del: ({ children }: { children?: ReactNode }) => (
        <del className="text-slate-400 line-through">{children}</del>
      ),
      a: ({ href, children, ...props }: ComponentPropsWithoutRef<'a'>) => (
        <a
          href={href}
          target="_blank"
          rel="noreferrer"
          className="text-blue-300 underline underline-offset-2 hover:text-blue-200"
          {...props}
        >
          {children}
        </a>
      ),
    }),
    []
  );

  if (!content) {
    return null;
  }

  const safeContent = typeof content === 'string' ? content : String(content);

  return (
    <MarkdownErrorBoundary content={safeContent}>
      <div className={`markdown-content ${className ?? ''}`}>
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
          {safeContent}
        </ReactMarkdown>
      </div>
    </MarkdownErrorBoundary>
  );
});

function extractTextContent(children: ReactNode): string {
  if (typeof children === 'string') {
    return children;
  }

  if (Array.isArray(children)) {
    return children.map(extractTextContent).join('');
  }

  if (children && typeof children === 'object' && 'props' in children) {
    return extractTextContent((children as { props: { children?: ReactNode } }).props.children);
  }

  return '';
}
