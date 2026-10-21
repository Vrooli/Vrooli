import { Component, createContext, memo, useContext, useEffect, useMemo, useState, type ComponentPropsWithoutRef, type MouseEvent, type ReactNode, type ErrorInfo } from "react";
import ReactMarkdown, { defaultUrlTransform } from "react-markdown";
import remarkGfm from "remark-gfm";
import { remarkProsePaths } from "./utils/remarkProsePaths";
import { CodeBlock } from "./components/CodeBlock";
import { InlineCode } from "./components/InlineCode";
import { MermaidDiagram } from "./components/MermaidDiagram";
import { ResizableMarkdownTable } from "@vrooli/react-component-library/markdown-renderer/0";
import { isExternalHref } from "../../lib/fileReferences";
import { resolveFilePreview } from "../../api/filePreview";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";

// Schemes allowed in markdown links beyond react-markdown's built-in safe set
// (http, https, irc(s), mailto, xmpp). `webapp:` is how an installed scenario
// PWA is opened on the operator's device; without this, react-markdown's
// defaultUrlTransform strips such an href to "" and the link renders inert.
// Everything else still flows through defaultUrlTransform, so javascript:,
// data:, and other unsafe schemes remain neutralized.
const EXTRA_SAFE_SCHEME = /^webapp:/i;

function markdownUrlTransform(url: string): string {
  return EXTRA_SAFE_SCHEME.test(url.trim()) ? url : defaultUrlTransform(url);
}

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
  /** Open a file in the preview dialog when an inline-code chip, or a fenced block holding only paths, looks like a path. */
  onFileReferenceClick?: (path: string) => void;
  /** Open a Mermaid diagram in the full-screen zoomable viewer. */
  onMermaidOpen?: (code: string) => void;
  /** Session whose local files can be resolved for inline message images. */
  imageSessionId?: string;
}

// Renderer functions are React component types. Keep them stable across parent
// updates so an in-flight press, focus, and selection keep their DOM nodes.
// Context delivers the latest callbacks without replacing the rendered links.
type MarkdownCallbacks = Pick<MarkdownRendererProps, "onLinkClick" | "onFileReferenceClick" | "onMermaidOpen" | "imageSessionId">;
const MarkdownCallbacksContext = createContext<MarkdownCallbacks>({});

function MarkdownImage({ src, alt = "", title }: ComponentPropsWithoutRef<"img">) {
  const { t } = useTranslation();
  const { imageSessionId, onFileReferenceClick } = useContext(MarkdownCallbacksContext);
  const localPath = imageSessionId && src && !/^https?:\/\//i.test(src) ? src : null;
  const [image, setImage] = useState<{ path: string; href?: string; failed?: boolean } | null>(null);

  useEffect(() => {
    if (!localPath || !imageSessionId) return;
    let active = true;
    resolveFilePreview(imageSessionId, localPath, "message_link")
      .then((preview) => {
        if (!active) return;
        if (preview.kind === "image" && preview.blobHref && preview.canPreview) {
          setImage({ path: localPath, href: preview.blobHref });
        } else {
          setImage({ path: localPath, failed: true });
        }
      })
      .catch(() => { if (active) setImage({ path: localPath, failed: true }); });
    return () => { active = false; };
  }, [imageSessionId, localPath]);

  if (!localPath) return <img src={src} alt={alt} title={title} className="max-w-full h-auto" />;
  if (image?.path === localPath && image.failed) return (
    <button type="button" className="text-wc-accent underline break-all" onClick={() => { onFileReferenceClick?.(localPath); }}>
      {alt || localPath} — {t(strings.messagesFileViewer.unavailable)}
    </button>
  );
  if (image?.path !== localPath || !image.href) return <span role="status">{t(strings.messagesFileViewer.loadingPreview)} {alt || localPath}</span>;
  return <img src={image.href} alt={alt} title={title} className="max-w-full h-auto" onError={() => { setImage({ path: localPath, failed: true }); }} />;
}

const markdownComponents = {
  img: MarkdownImage,
  code: function MarkdownCode({
    inline,
    className: codeClassName,
    children,
    ...props
  }: ComponentPropsWithoutRef<"code"> & { inline?: boolean }) {
    const { onFileReferenceClick, onMermaidOpen } = useContext(MarkdownCallbacksContext);
    const codeContent = extractTextContent(children);
    const isInline = inline ?? (!codeClassName && !codeContent.includes("\n"));

    if (isInline) return <InlineCode onFileReferenceClick={onFileReferenceClick}>{children}</InlineCode>;
    if (codeClassName === "language-mermaid") return <MermaidDiagram code={codeContent} onOpenFullscreen={onMermaidOpen} />;
    return <CodeBlock code={codeContent} className={codeClassName} onFileReferenceClick={onFileReferenceClick} {...props} />;
  },

  pre: ({ children }: { children?: ReactNode }) => <>{children}</>,

  a: function MarkdownLink({ href, children, ...props }: ComponentPropsWithoutRef<"a"> & { "data-prose-path"?: string }) {
    const { onLinkClick } = useContext(MarkdownCallbacksContext);
    const safeHref = href ?? "";
    const external = isExternalHref(safeHref);
    // Auto-detected bare paths get a quieter treatment than authored
    // links: dotted underline, solid on hover.
    const isProsePath = props["data-prose-path"] === "true";
    return (
      <a
        href={safeHref}
        target={external ? "_blank" : undefined}
        rel={external ? "noopener noreferrer" : undefined}
        className={
          isProsePath
            ? "text-wc-accent/90 underline decoration-dotted underline-offset-2 hover:decoration-solid hover:text-wc-accent"
            : "text-wc-accent underline underline-offset-2 hover:text-wc-accent/80"
        }
        title={isProsePath ? `Open ${safeHref}` : undefined}
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
    <ResizableMarkdownTable>{children}</ResizableMarkdownTable>
  ),
  thead: ({ children }: { children?: ReactNode }) => (
    <thead className="bg-wc-surface-raised">{children}</thead>
  ),
  th: ({ children, ...props }: ComponentPropsWithoutRef<"th"> & { children?: ReactNode }) => (
    <th {...props} className="border border-wc-default px-4 py-2 text-start font-semibold align-top min-w-[8rem]">{children}</th>
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
};

/** Renders markdown content with syntax highlighting, mermaid diagrams, and GFM support. */
export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
  className,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
  imageSessionId,
}: MarkdownRendererProps) {
  const callbacks = useMemo(() => ({ onLinkClick, onFileReferenceClick, onMermaidOpen, imageSessionId }),
    [onLinkClick, onFileReferenceClick, onMermaidOpen, imageSessionId]);

  if (!content) return null;

  const safeContent = typeof content === "string" ? content : String(content);

  // If search is active, wrap rendered output with highlighting
  const rendered = (
    <ReactMarkdown
      remarkPlugins={[remarkGfm, remarkProsePaths]}
      components={markdownComponents}
      urlTransform={markdownUrlTransform}
    >
      {safeContent}
    </ReactMarkdown>
  );

  return (
    <MarkdownCallbacksContext.Provider value={callbacks}>
      <MarkdownErrorBoundary content={safeContent}>
        <div className={`markdown-content min-w-0 max-w-full break-words [overflow-wrap:anywhere] ${className || ""}`}>
          {rendered}
        </div>
      </MarkdownErrorBoundary>
    </MarkdownCallbacksContext.Provider>
  );
}, (prevProps, nextProps) => {
  return (
    prevProps.content === nextProps.content &&
    prevProps.className === nextProps.className &&
    prevProps.onLinkClick === nextProps.onLinkClick &&
    prevProps.onFileReferenceClick === nextProps.onFileReferenceClick &&
    prevProps.onMermaidOpen === nextProps.onMermaidOpen
    && prevProps.imageSessionId === nextProps.imageSessionId
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
