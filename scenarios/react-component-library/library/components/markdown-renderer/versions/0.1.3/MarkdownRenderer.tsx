import {
  Component,
  type ErrorInfo,
  type MouseEvent,
  type ReactNode,
  useEffect,
  useMemo,
  useState,
} from "react";
import mermaid from "mermaid";
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

const highlightCache = new Map<string, string>();
let highlighter: Promise<import("shiki").Highlighter> | null = null;

function HighlightedCode({
  code,
  language,
}: {
  code: string;
  language: string;
}) {
  const key = `${language}\u0000${code}`;
  const [html, setHTML] = useState(() => highlightCache.get(key));
  useEffect(() => {
    if (highlightCache.has(key)) {
      setHTML(highlightCache.get(key));
      return;
    }
    let cancelled = false;
    highlighter ??= import("shiki").then((shiki) =>
      shiki.createHighlighter({
        themes: ["github-dark"],
        langs: [
          "typescript",
          "javascript",
          "python",
          "go",
          "json",
          "bash",
          "sql",
          "html",
          "css",
          "yaml",
          "markdown",
          "tsx",
          "jsx",
        ],
      }),
    );
    void highlighter
      .then((instance) =>
        instance.codeToHtml(code, {
          lang: instance.getLoadedLanguages().includes(language)
            ? language
            : "text",
          theme: "github-dark",
        }),
      )
      .then((result) => {
        if (highlightCache.size >= 400) {
          const oldest = highlightCache.keys().next().value;
          if (oldest) highlightCache.delete(oldest);
        }
        highlightCache.set(key, result);
        if (!cancelled) setHTML(result);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [code, key, language]);
  return html ? (
    <div
      className="overflow-x-auto p-3 text-sm [&>pre]:m-0 [&>pre]:bg-transparent [&>pre]:p-0"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  ) : (
    <pre className="overflow-x-auto p-3 text-sm text-slate-100">
      <code>{code}</code>
    </pre>
  );
}

function MermaidDiagram({
  code,
  onOpen,
}: {
  code: string;
  onOpen?: (code: string) => void;
}) {
  const [svg, setSVG] = useState<string>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    let active = true;
    const id = `rcl-mermaid-${Math.random().toString(36).slice(2)}`;
    mermaid.initialize({ startOnLoad: false, securityLevel: "strict" });
    void mermaid
      .render(id, code)
      .then(({ svg }) => {
        if (active) setSVG(svg);
      })
      .catch((reason: unknown) => {
        if (active)
          setError(
            reason instanceof Error
              ? reason.message
              : "Unable to render Mermaid diagram",
          );
      });
    return () => {
      active = false;
    };
  }, [code]);
  return (
    <section className="my-3 overflow-x-auto rounded border border-slate-700 bg-slate-950">
      <header className="flex items-center justify-between border-b border-slate-700 px-3 py-2 text-xs text-slate-400">
        <span>mermaid</span>
        {onOpen && (
          <button
            type="button"
            onClick={() => onOpen(code)}
            className="text-cyan-300"
          >
            Open
          </button>
        )}
      </header>
      {error ? (
        <>
          <p className="p-3 text-xs text-red-300">{error}</p>
          <pre className="p-3 text-xs text-slate-200">{code}</pre>
        </>
      ) : svg ? (
        <div
          className="p-3 [&>svg]:max-w-full"
          dangerouslySetInnerHTML={{ __html: svg }}
        />
      ) : (
        <p className="p-3 text-xs text-slate-400">Rendering diagram…</p>
      )}
    </section>
  );
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
 * @version 0.1.3
 * @tags ["markdown","rendering","gfm"]
 * @deps {"react":"^18","react-markdown":"^10.1.0","remark-gfm":"^4.0.1","shiki":"^4.3.1","mermaid":"^11.4.0"}
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
          return <MermaidDiagram code={text} onOpen={onMermaidOpen} />;
        if (codeClass)
          return (
            <div className="my-3 overflow-hidden rounded border border-slate-700 bg-slate-950">
              <HighlightedCode
                code={text}
                language={codeClass.replace(/^language-/, "")}
              />
            </div>
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
