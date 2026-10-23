import { Component, createContext, memo, useContext, useEffect, useMemo, useRef, useState, type ComponentPropsWithoutRef, type MouseEvent, type ReactNode, type ErrorInfo } from "react";
import ReactMarkdown, { defaultUrlTransform } from "react-markdown";
import remarkGfm from "remark-gfm";
import { remarkProsePaths } from "./utils/remarkProsePaths";
import { CodeBlock } from "./components/CodeBlock";
import { InlineCode } from "./components/InlineCode";
import { MermaidDiagram } from "./components/MermaidDiagram";
import { ResizableMarkdownTable } from "@vrooli/react-component-library/markdown-renderer/0";
import { isExternalHref } from "../../lib/fileReferences";
import { resolveFilePreview } from "../../api/filePreview";
import { splitMarkdownChunks } from "./utils/splitMarkdownChunks";
import { observeNear } from "./utils/lazyVisibility";
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
  /**
   * Directory that relative inline image paths resolve against — the markdown
   * file's own directory. Without it, a relative `src` like `./diagram.png`
   * would resolve against the session cwd (wrong for a file preview) instead of
   * sitting next to the document that references it.
   */
  imageBasePath?: string;
  /**
   * Render a large document in block-aligned chunks across idle frames instead
   * of one blocking parse. Use for file previews (whole files arrive at once);
   * leave off for streaming message bodies, which are small and append live.
   */
  progressive?: boolean;
}

// Resolve a markdown image `src` to the path handed to the preview backend.
// External URLs pass through untouched; an absolute, home, or file:// path is
// already addressable; a relative path is joined onto the document's own
// directory so it resolves next to the file that references it.
function resolveImageSrc(src: string, baseDir?: string): string {
  if (!baseDir || /^(\/|~\/|file:\/\/)/.test(src)) return src;
  const dir = baseDir.endsWith("/") ? baseDir : `${baseDir}/`;
  return `${dir}${src}`;
}

// Renderer functions are React component types. Keep them stable across parent
// updates so an in-flight press, focus, and selection keep their DOM nodes.
// Context delivers the latest callbacks without replacing the rendered links.
type MarkdownCallbacks = Pick<MarkdownRendererProps, "onLinkClick" | "onFileReferenceClick" | "onMermaidOpen" | "imageSessionId" | "imageBasePath">;
const MarkdownCallbacksContext = createContext<MarkdownCallbacks>({});

// A rough height held for an inline image that has never been on screen, so the
// document starts near its true length instead of collapsing and growing as
// images stream in. Replaced by each image's real rendered height once known.
const IMAGE_PLACEHOLDER_HEIGHT = 360;

function MarkdownImage({ src, alt = "", title }: ComponentPropsWithoutRef<"img">) {
  const { t } = useTranslation();
  const { imageSessionId, imageBasePath, onFileReferenceClick } = useContext(MarkdownCallbacksContext);
  const localPath = imageSessionId && src && !/^https?:\/\//i.test(src) ? resolveImageSrc(src, imageBasePath) : null;
  const [image, setImage] = useState<{ path: string; href?: string; failed?: boolean } | null>(null);
  // Defer resolving + decoding a local image until it first nears the viewport,
  // then keep it mounted. Resolving all of a large document's images on open
  // floods the preview API; but unmounting them again on scroll would mutate the
  // DOM mid-scroll and force the tall document to re-layout (costly on iOS), so
  // this is a one-shot reveal, not a window.
  const [seen, setSeen] = useState(false);
  const holderRef = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    if (!localPath || seen) return;
    const holder = holderRef.current;
    if (!holder) { setSeen(true); return; }
    return observeNear(holder, (near) => { if (near) setSeen(true); });
  }, [localPath, seen]);

  useEffect(() => {
    if (!localPath || !imageSessionId || !seen) return;
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
  }, [imageSessionId, localPath, seen]);

  if (!localPath) return <img src={src} alt={alt} title={title} loading="lazy" decoding="async" className="max-w-full h-auto" />;
  // Not yet scrolled near: a space-reserving placeholder so the document starts
  // near its true height instead of growing as images stream in.
  if (!seen) return <span ref={holderRef} className="block" style={{ minHeight: IMAGE_PLACEHOLDER_HEIGHT }} />;
  if (image?.path === localPath && image.failed) return (
    <button type="button" className="text-wc-accent underline break-all" onClick={() => { onFileReferenceClick?.(localPath); }}>
      {alt || localPath} — {t(strings.messagesFileViewer.unavailable)}
    </button>
  );
  if (image?.path !== localPath || !image.href) return <span ref={holderRef} role="status" className="block" style={{ minHeight: IMAGE_PLACEHOLDER_HEIGHT }}>{t(strings.messagesFileViewer.loadingPreview)} {alt || localPath}</span>;
  const resolved = <img src={image.href} alt={alt} title={title} loading="lazy" decoding="async" className="max-w-full h-auto" onError={() => { setImage({ path: localPath, failed: true }); }} />;
  // A resolved local image opens full-size in the file preview (zoom/pan) on
  // press. Without a preview handler (e.g. plain message bodies) it stays inert.
  if (!onFileReferenceClick) return resolved;
  return (
    <button
      type="button"
      onClick={() => { onFileReferenceClick(localPath); }}
      title={t(strings.messagesFileViewer.openImage)}
      aria-label={alt ? `${t(strings.messagesFileViewer.openImage)}: ${alt}` : t(strings.messagesFileViewer.openImage)}
      className="block max-w-full cursor-zoom-in rounded outline-none focus-visible:ring-2 focus-visible:ring-wc-accent"
    >
      {resolved}
    </button>
  );
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

// One parsed slice of the document. Memoized by its text so that, as later
// chunks stream in, the chunks already on screen are not re-parsed.
const MarkdownChunk = memo(function MarkdownChunk({ content }: { content: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm, remarkProsePaths]}
      components={markdownComponents}
      urlTransform={markdownUrlTransform}
    >
      {content}
    </ReactMarkdown>
  );
});

/** Renders markdown content with syntax highlighting, mermaid diagrams, and GFM support. */
export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
  className,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
  imageSessionId,
  imageBasePath,
  progressive = false,
}: MarkdownRendererProps) {
  const callbacks = useMemo(() => ({ onLinkClick, onFileReferenceClick, onMermaidOpen, imageSessionId, imageBasePath }),
    [onLinkClick, onFileReferenceClick, onMermaidOpen, imageSessionId, imageBasePath]);

  const safeContent = typeof content === "string" ? content : String(content);

  // In progressive mode a large document is split into block-aligned chunks;
  // the first renders now and the rest are revealed one per idle frame, turning
  // a single long parse into several short ones that never freeze the thread.
  const chunks = useMemo(() => (progressive ? splitMarkdownChunks(safeContent) : [safeContent]), [progressive, safeContent]);
  const [revealed, setRevealed] = useState(1);
  useEffect(() => { setRevealed(1); }, [chunks]);
  useEffect(() => {
    if (revealed >= chunks.length) return;
    const advance = () => { setRevealed((shown) => Math.min(chunks.length, shown + 1)); };
    if (typeof requestIdleCallback === "undefined") {
      const timer = window.setTimeout(advance, 16);
      return () => { window.clearTimeout(timer); };
    }
    // The timeout guards against a thread that never goes idle (e.g. the user
    // is scrolling), so the rest of the document still fills in.
    const handle = requestIdleCallback(advance, { timeout: 250 });
    return () => { cancelIdleCallback(handle); };
  }, [revealed, chunks.length]);

  if (!safeContent) return null;

  return (
    <MarkdownCallbacksContext.Provider value={callbacks}>
      <MarkdownErrorBoundary content={safeContent}>
        <div className={`markdown-content min-w-0 max-w-full break-words [overflow-wrap:anywhere] ${className || ""}`}>
          {chunks.slice(0, revealed).map((chunk, index) => <MarkdownChunk key={index} content={chunk} />)}
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
    && prevProps.imageBasePath === nextProps.imageBasePath
    && prevProps.progressive === nextProps.progressive
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
