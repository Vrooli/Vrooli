import { useState, useEffect, useRef } from "react";
import { createRoot } from "react-dom/client";
import { Copy, Check, ArrowLeft } from "lucide-react";
import mermaid from "mermaid";
import { CodePreview } from "../../../shared/components";
import { themeColors } from "../../../shared/theme/colors";
import { Button } from "../../../shared/ui/primitives";

// Initialize mermaid
mermaid.initialize({
  startOnLoad: false,
  theme: "dark",
  securityLevel: "loose",
});

interface MarkdownViewerProps {
  content: string;
  path: string;
  isLoading: boolean;
  error?: Error | null;
  onBack: () => void;
}

interface MermaidBlock {
  id: string;
  code: string;
}

interface CodeBlock {
  id: string;
  code: string;
  language: string;
}

// Extract mermaid blocks and return both the modified markdown and the blocks
function extractMermaidBlocks(markdown: string): { html: string; blocks: MermaidBlock[] } {
  const blocks: MermaidBlock[] = [];
  let counter = 0;

  const html = markdown.replace(/```mermaid\n([\s\S]*?)```/g, (_match: string, code: string) => {
    const id = `mermaid-${Date.now()}-${counter++}`;
    blocks.push({ id, code: code.trim() });
    return `<div class="mermaid-container my-4 overflow-x-auto rounded-lg border border-border-default/70 bg-surface-overlay/40 p-4"><div id="${id}" class="mermaid-placeholder flex items-center justify-center py-8 text-text-muted">Loading diagram...</div></div>`;
  });

  return { html, blocks };
}

// Simple markdown to HTML converter with mermaid support
function parseMarkdown(markdown: string): { html: string; mermaidBlocks: MermaidBlock[]; codeBlocks: CodeBlock[] } {
  if (!markdown) return { html: "", mermaidBlocks: [], codeBlocks: [] };

  // First extract mermaid blocks
  const { html: withoutMermaid, blocks } = extractMermaidBlocks(markdown);

  const codeBlocks: CodeBlock[] = [];
  let codeCounter = 0;

  let html = withoutMermaid
    // Other code blocks — replace with placeholder divs for React hydration
    .replace(/```(\w+)?\n([\s\S]*?)```/g, (_match: string, lang: string | undefined, code: string) => {
      const id = `code-block-${Date.now()}-${codeCounter++}`;
      codeBlocks.push({ id, code: code.trim(), language: lang || "text" });
      return `<div id="${id}" class="my-4"></div>`;
    })
    // Inline code
    .replace(/`([^`]+)`/g, '<code class="rounded bg-surface-overlay/70 px-1.5 py-0.5 text-sm text-accent-primary">$1</code>')
    // Headers
    .replace(/^#### (.+)$/gm, '<h4 class="mt-6 mb-2 text-base font-semibold text-text-primary">$1</h4>')
    .replace(/^### (.+)$/gm, '<h3 class="mt-6 mb-2 text-lg font-semibold text-text-primary">$1</h3>')
    .replace(/^## (.+)$/gm, '<h2 class="mt-8 mb-3 text-xl font-semibold text-text-primary">$1</h2>')
    .replace(/^# (.+)$/gm, '<h1 class="mt-4 mb-4 text-2xl font-bold text-text-primary">$1</h1>')
    // Bold and italic
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/\*(.+?)\*/g, "<em>$1</em>")
    // Links
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" class="text-accent-primary hover:underline">$1</a>')
    // Unordered lists
    .replace(/^- (.+)$/gm, '<li class="ml-4 list-inside list-disc text-text-primary">$1</li>')
    // Ordered lists
    .replace(/^\d+\. (.+)$/gm, '<li class="ml-4 list-inside list-decimal text-text-primary">$1</li>')
    // Tables
    .replace(/^\|(.+)\|$/gm, (_match: string, content: string) => {
      const cells = content.split("|").map((c: string) => c.trim());
      const isHeader = cells.every((c: string) => /^-+$/.test(c));
      if (isHeader) return "";
      const tag = "td";
      return `<tr>${cells.map((c: string) => `<${tag} class="border border-border-default/70 px-3 py-2 text-text-primary">${c}</${tag}>`).join("")}</tr>`;
    })
    // Horizontal rules
    .replace(/^---$/gm, '<hr class="my-6 border-border-default/70" />')
    // Paragraphs (double newlines)
    .replace(/\n\n/g, '</p><p class="my-3 text-text-primary">');

  // Wrap in paragraph if needed
  if (!html.startsWith("<")) {
    html = `<p class="my-3 text-text-primary">${html}</p>`;
  }

  return { html, mermaidBlocks: blocks, codeBlocks };
}

// Post-process SVG to apply dark theme colors directly
function applyDarkThemeToSvg(svg: string): string {
  const parser = new DOMParser();
  const doc = parser.parseFromString(svg, "image/svg+xml");
  const svgEl = doc.querySelector("svg");
  if (!svgEl) return svg;

  svgEl.style.backgroundColor = "transparent";

  const setFill = (el: Element, color: string) => {
    (el as HTMLElement).style.fill = color;
    el.setAttribute("fill", color);
  };

  const setStroke = (el: Element, color: string) => {
    (el as HTMLElement).style.stroke = color;
    el.setAttribute("stroke", color);
  };

  // Fix rect elements
  doc.querySelectorAll("rect").forEach((rect) => {
    const currentFill = rect.getAttribute("fill") || window.getComputedStyle(rect).fill;
    if (currentFill === "none" || currentFill === "transparent") return;

    const isCluster = rect.closest(".cluster") !== null;
    const isLabel = rect.closest(".edgeLabel, .labelBkg") !== null;

    if (isCluster) {
      setFill(rect, themeColors.mermaid.clusterBg);
      setStroke(rect, themeColors.mermaid.border);
    } else if (isLabel) {
      setFill(rect, themeColors.mermaid.clusterBg);
    } else {
      setFill(rect, themeColors.mermaid.nodeBg);
      setStroke(rect, themeColors.mermaid.accent);
    }
  });

  // Fix polygon elements
  doc.querySelectorAll("polygon").forEach((poly) => {
    const currentFill = poly.getAttribute("fill");
    if (currentFill === "none" || currentFill === "transparent") return;
    setFill(poly, themeColors.mermaid.nodeBg);
    setStroke(poly, themeColors.mermaid.accent);
  });

  // Fix circle and ellipse
  doc.querySelectorAll("circle, ellipse").forEach((el) => {
    const currentFill = el.getAttribute("fill");
    if (currentFill === "none" || currentFill === "transparent") return;
    setFill(el, themeColors.mermaid.nodeBg);
    setStroke(el, themeColors.mermaid.accent);
  });

  // Fix text
  doc.querySelectorAll("text, tspan").forEach((text) => {
    setFill(text, themeColors.mermaid.text);
    (text as HTMLElement).style.color = themeColors.mermaid.text;
  });

  // Fix foreignObject content
  doc.querySelectorAll("foreignObject div, foreignObject span, foreignObject p").forEach((el) => {
    (el as HTMLElement).style.color = themeColors.mermaid.text;
    (el as HTMLElement).style.fill = themeColors.mermaid.text;
  });

  // Fix paths (edges)
  doc.querySelectorAll("path:not(marker path)").forEach((path) => {
    const stroke = path.getAttribute("stroke");
    if (stroke && stroke !== "none") {
      setStroke(path, themeColors.mermaid.line);
    }
  });

  // Fix marker elements
  doc.querySelectorAll("marker path").forEach((path) => {
    setFill(path, themeColors.mermaid.line);
    setStroke(path, themeColors.mermaid.line);
  });

  // Fix lines
  doc.querySelectorAll("line").forEach((line) => {
    setStroke(line, themeColors.mermaid.line);
  });

  const serializer = new XMLSerializer();
  return serializer.serializeToString(doc);
}

export function MarkdownViewer({
  content,
  path,
  isLoading,
  error,
  onBack
}: MarkdownViewerProps) {
  const [copied, setCopied] = useState(false);
  const [mermaidBlocks, setMermaidBlocks] = useState<MermaidBlock[]>([]);
  const [codeBlocks, setCodeBlocks] = useState<CodeBlock[]>([]);
  const [htmlContent, setHtmlContent] = useState("");
  const articleRef = useRef<HTMLElement>(null);
  const codeRootsRef = useRef<Array<ReturnType<typeof createRoot>>>([]);

  // Parse markdown and extract mermaid blocks
  useEffect(() => {
    if (!content) {
      setHtmlContent("");
      setMermaidBlocks([]);
      setCodeBlocks([]);
      return;
    }
    const { html, mermaidBlocks: mBlocks, codeBlocks: cBlocks } = parseMarkdown(content);
    setHtmlContent(html);
    setMermaidBlocks(mBlocks);
    setCodeBlocks(cBlocks);
  }, [content]);

  // Hydrate code block placeholders with CodePreview components
  useEffect(() => {
    // Cleanup previous roots
    for (const root of codeRootsRef.current) {
      root.unmount();
    }
    codeRootsRef.current = [];

    if (codeBlocks.length === 0 || !articleRef.current) return;

    for (const block of codeBlocks) {
      const element = document.getElementById(block.id);
      if (!element) continue;

      const root = createRoot(element);
      root.render(<CodePreview code={block.code} language={block.language} />);
      codeRootsRef.current.push(root);
    }

    return () => {
      for (const root of codeRootsRef.current) {
        root.unmount();
      }
      codeRootsRef.current = [];
    };
  }, [codeBlocks, htmlContent]);

  // Render mermaid diagrams after content is inserted into DOM
  useEffect(() => {
    if (mermaidBlocks.length === 0 || !articleRef.current) return;

    const renderDiagrams = async () => {
      for (const block of mermaidBlocks) {
        const element = document.getElementById(block.id);
        if (!element) continue;

        try {
          const { svg } = await mermaid.render(`${block.id}-svg`, block.code);
          const styledSvg = applyDarkThemeToSvg(svg);
          element.innerHTML = styledSvg;
          element.classList.remove("mermaid-placeholder");
        } catch (err) {
          console.error(`Failed to render mermaid diagram ${block.id}:`, err);
          element.innerHTML = `<div class="p-4 text-sm text-accent-danger">Failed to render diagram. Check console for details.</div>`;
          element.classList.remove("mermaid-placeholder");
        }
      }
    };

    const timeoutId = setTimeout(renderDiagrams, 10);
    return () => clearTimeout(timeoutId);
  }, [mermaidBlocks, htmlContent]);

  const handleCopyPath = async () => {
    const fullPath = `scenarios/vrooli-autoheal/docs/${path}`;
    try {
      await navigator.clipboard.writeText(fullPath);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy path:", err);
    }
  };

  if (isLoading) {
    return (
      <div
        className="flex-1 rounded-xl border border-border-default/70 bg-surface-elevated/40 p-3 sm:p-6"
        data-testid="docs-viewer"
      >
        <p className="text-text-muted">Loading document...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex-1 rounded-xl border border-border-default/70 bg-surface-elevated/40 p-3 sm:p-6"
        data-testid="docs-viewer"
      >
        <p className="text-accent-danger">Failed to load document: {error.message}</p>
        <Button variant="outline" size="sm" className="mt-4" onClick={onBack}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Go back
        </Button>
      </div>
    );
  }

  if (!content && !path) {
    return (
      <div
        className="flex-1 rounded-xl border border-border-default/70 bg-surface-elevated/40 p-3 sm:p-6"
        data-testid="docs-viewer"
      >
        <div className="text-center py-12">
          <p className="text-text-muted">Select a document from the sidebar to view it here.</p>
        </div>
      </div>
    );
  }

  return (
    <div
      className="flex-1 overflow-hidden rounded-xl border border-border-default/70 bg-surface-elevated/40 p-3 pb-[max(env(safe-area-inset-bottom),0.75rem)] sm:p-6"
      data-testid="docs-viewer"
    >
      {/* Header */}
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2 sm:mb-6">
        <div className="flex min-w-0 items-center gap-2 text-sm text-text-muted">
          <button onClick={onBack} className="transition hover:text-text-primary">
            Docs
          </button>
          <span>/</span>
          <span className="truncate break-all text-text-primary">{path.replace(/\.md$/, "")}</span>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={handleCopyPath}
          data-testid="docs-copy-path"
        >
          {copied ? (
            <>
              <Check className="mr-2 h-4 w-4" />
              Copied!
            </>
          ) : (
            <>
              <Copy className="mr-2 h-4 w-4" />
              Copy path
            </>
          )}
        </Button>
      </div>

      {/* Content */}
      <article
        ref={articleRef}
        className="prose prose-invert prose-sm max-h-[60dvh] max-w-none overflow-y-auto pb-2 sm:max-h-[calc(100dvh-20rem)] lg:max-h-[calc(100dvh-16rem)]"
        dangerouslySetInnerHTML={{ __html: htmlContent }}
      />
    </div>
  );
}
