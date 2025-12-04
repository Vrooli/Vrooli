import { useState, useEffect, useRef } from "react";
import { Copy, Check, ArrowLeft } from "lucide-react";
import mermaid from "mermaid";
import { Button } from "../../components/ui/button";

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

// Extract mermaid blocks and return both the modified markdown and the blocks
function extractMermaidBlocks(markdown: string): { html: string; blocks: MermaidBlock[] } {
  const blocks: MermaidBlock[] = [];
  let counter = 0;

  const html = markdown.replace(/```mermaid\n([\s\S]*?)```/g, (_, code) => {
    const id = `mermaid-${Date.now()}-${counter++}`;
    blocks.push({ id, code: code.trim() });
    return `<div class="mermaid-container my-4 p-4 bg-slate-900/50 rounded-lg border border-white/10 overflow-x-auto"><div id="${id}" class="mermaid-placeholder flex items-center justify-center py-8 text-slate-400">Loading diagram...</div></div>`;
  });

  return { html, blocks };
}

// Simple markdown to HTML converter with mermaid support
function parseMarkdown(markdown: string): { html: string; mermaidBlocks: MermaidBlock[] } {
  if (!markdown) return { html: "", mermaidBlocks: [] };

  // First extract mermaid blocks
  const { html: withoutMermaid, blocks } = extractMermaidBlocks(markdown);

  let html = withoutMermaid
    // Other code blocks
    .replace(/```(\w+)?\n([\s\S]*?)```/g, (_, lang, code) =>
      `<pre class="bg-black/50 rounded-lg p-4 overflow-x-auto text-sm my-4"><code class="language-${lang || "text"}">${escapeHtml(code.trim())}</code></pre>`
    )
    // Inline code
    .replace(/`([^`]+)`/g, '<code class="bg-black/50 px-1.5 py-0.5 rounded text-blue-400 text-sm">$1</code>')
    // Headers
    .replace(/^#### (.+)$/gm, '<h4 class="text-base font-semibold mt-6 mb-2 text-slate-200">$1</h4>')
    .replace(/^### (.+)$/gm, '<h3 class="text-lg font-semibold mt-6 mb-2 text-slate-100">$1</h3>')
    .replace(/^## (.+)$/gm, '<h2 class="text-xl font-semibold mt-8 mb-3 text-white">$1</h2>')
    .replace(/^# (.+)$/gm, '<h1 class="text-2xl font-bold mt-4 mb-4 text-white">$1</h1>')
    // Bold and italic
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/\*(.+?)\*/g, "<em>$1</em>")
    // Links
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" class="text-blue-400 hover:underline">$1</a>')
    // Unordered lists
    .replace(/^- (.+)$/gm, '<li class="ml-4 list-disc list-inside text-slate-300">$1</li>')
    // Ordered lists
    .replace(/^\d+\. (.+)$/gm, '<li class="ml-4 list-decimal list-inside text-slate-300">$1</li>')
    // Tables
    .replace(/^\|(.+)\|$/gm, (match, content) => {
      const cells = content.split("|").map((c: string) => c.trim());
      const isHeader = cells.every((c: string) => /^-+$/.test(c));
      if (isHeader) return "";
      const tag = "td";
      return `<tr>${cells.map((c: string) => `<${tag} class="border border-white/10 px-3 py-2 text-slate-300">${c}</${tag}>`).join("")}</tr>`;
    })
    // Horizontal rules
    .replace(/^---$/gm, '<hr class="border-white/10 my-6" />')
    // Paragraphs (double newlines)
    .replace(/\n\n/g, '</p><p class="my-3 text-slate-300">');

  // Wrap in paragraph if needed
  if (!html.startsWith("<")) {
    html = `<p class="my-3 text-slate-300">${html}</p>`;
  }

  return { html, mermaidBlocks: blocks };
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

// Dark theme colors for mermaid diagrams
const darkTheme = {
  bgDark: "#1e293b",      // slate-800 - node backgrounds
  bgDarker: "#0f172a",    // slate-900 - cluster backgrounds
  textLight: "#f1f5f9",   // slate-100 - primary text
  blue: "#3b82f6",        // blue-500 - accent/borders
  lineMuted: "#94a3b8",   // slate-400 - lines/arrows
  borderMedium: "#334155", // slate-700 - borders
};

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
      setFill(rect, darkTheme.bgDarker);
      setStroke(rect, darkTheme.borderMedium);
    } else if (isLabel) {
      setFill(rect, darkTheme.bgDarker);
    } else {
      setFill(rect, darkTheme.bgDark);
      setStroke(rect, darkTheme.blue);
    }
  });

  // Fix polygon elements
  doc.querySelectorAll("polygon").forEach((poly) => {
    const currentFill = poly.getAttribute("fill");
    if (currentFill === "none" || currentFill === "transparent") return;
    setFill(poly, darkTheme.bgDark);
    setStroke(poly, darkTheme.blue);
  });

  // Fix circle and ellipse
  doc.querySelectorAll("circle, ellipse").forEach((el) => {
    const currentFill = el.getAttribute("fill");
    if (currentFill === "none" || currentFill === "transparent") return;
    setFill(el, darkTheme.bgDark);
    setStroke(el, darkTheme.blue);
  });

  // Fix text
  doc.querySelectorAll("text, tspan").forEach((text) => {
    setFill(text, darkTheme.textLight);
    (text as HTMLElement).style.color = darkTheme.textLight;
  });

  // Fix foreignObject content
  doc.querySelectorAll("foreignObject div, foreignObject span, foreignObject p").forEach((el) => {
    (el as HTMLElement).style.color = darkTheme.textLight;
    (el as HTMLElement).style.fill = darkTheme.textLight;
  });

  // Fix paths (edges)
  doc.querySelectorAll("path:not(marker path)").forEach((path) => {
    const stroke = path.getAttribute("stroke");
    if (stroke && stroke !== "none") {
      setStroke(path, darkTheme.lineMuted);
    }
  });

  // Fix marker elements
  doc.querySelectorAll("marker path").forEach((path) => {
    setFill(path, darkTheme.lineMuted);
    setStroke(path, darkTheme.lineMuted);
  });

  // Fix lines
  doc.querySelectorAll("line").forEach((line) => {
    setStroke(line, darkTheme.lineMuted);
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
  const [htmlContent, setHtmlContent] = useState("");
  const articleRef = useRef<HTMLElement>(null);

  // Parse markdown and extract mermaid blocks
  useEffect(() => {
    if (!content) {
      setHtmlContent("");
      setMermaidBlocks([]);
      return;
    }
    const { html, mermaidBlocks: blocks } = parseMarkdown(content);
    setHtmlContent(html);
    setMermaidBlocks(blocks);
  }, [content]);

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
          element.innerHTML = `<div class="text-red-400 text-sm p-4">Failed to render diagram. Check console for details.</div>`;
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
        className="flex-1 rounded-xl border border-white/10 bg-white/5 p-6"
        data-testid="docs-viewer"
      >
        <p className="text-slate-400">Loading document...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex-1 rounded-xl border border-white/10 bg-white/5 p-6"
        data-testid="docs-viewer"
      >
        <p className="text-red-400">Failed to load document: {error.message}</p>
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
        className="flex-1 rounded-xl border border-white/10 bg-white/5 p-6"
        data-testid="docs-viewer"
      >
        <div className="text-center py-12">
          <p className="text-slate-400">Select a document from the sidebar to view it here.</p>
        </div>
      </div>
    );
  }

  return (
    <div
      className="flex-1 rounded-xl border border-white/10 bg-white/5 p-6 overflow-hidden"
      data-testid="docs-viewer"
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <button onClick={onBack} className="hover:text-white transition">
            Docs
          </button>
          <span>/</span>
          <span className="text-slate-200">{path.replace(/\.md$/, "")}</span>
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
        className="prose prose-invert prose-sm max-w-none overflow-y-auto max-h-[calc(100vh-300px)]"
        dangerouslySetInnerHTML={{ __html: htmlContent }}
      />
    </div>
  );
}
