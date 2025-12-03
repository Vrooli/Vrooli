import { useState } from "react";
import { Copy, Check, ExternalLink } from "lucide-react";
import { Button } from "../../components/ui/button";
import { Breadcrumb } from "../../components/layout/Breadcrumb";
import { selectors } from "../../consts/selectors";

interface MarkdownViewerProps {
  content: string;
  path: string;
  isLoading: boolean;
  error?: Error | null;
  onBack: () => void;
}

// Simple markdown to HTML converter for basic rendering
// In production, you'd use react-markdown with plugins
function parseMarkdown(markdown: string): string {
  if (!markdown) return "";

  let html = markdown
    // Code blocks (must come first)
    .replace(/```(\w+)?\n([\s\S]*?)```/g, (_, lang, code) =>
      `<pre class="bg-black/50 rounded-lg p-4 overflow-x-auto text-sm"><code class="language-${lang || "text"}">${escapeHtml(code.trim())}</code></pre>`
    )
    // Inline code
    .replace(/`([^`]+)`/g, '<code class="bg-black/50 px-1.5 py-0.5 rounded text-cyan-400">$1</code>')
    // Headers
    .replace(/^### (.+)$/gm, '<h3 class="text-lg font-semibold mt-6 mb-2">$1</h3>')
    .replace(/^## (.+)$/gm, '<h2 class="text-xl font-semibold mt-8 mb-3">$1</h2>')
    .replace(/^# (.+)$/gm, '<h1 class="text-2xl font-bold mt-4 mb-4">$1</h1>')
    // Bold and italic
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/\*(.+?)\*/g, "<em>$1</em>")
    // Links
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" class="text-cyan-400 hover:underline">$1</a>')
    // Lists
    .replace(/^\- (.+)$/gm, '<li class="ml-4">$1</li>')
    .replace(/^\d+\. (.+)$/gm, '<li class="ml-4 list-decimal">$1</li>')
    // Tables (basic support)
    .replace(/^\|(.+)\|$/gm, (match, content) => {
      const cells = content.split("|").map((c: string) => c.trim());
      const isHeader = cells.every((c: string) => c.startsWith("-"));
      if (isHeader) return "";
      const tag = match.includes("---") ? "th" : "td";
      return `<tr>${cells.map((c: string) => `<${tag} class="border border-white/10 px-3 py-2">${c}</${tag}>`).join("")}</tr>`;
    })
    // Horizontal rules
    .replace(/^---$/gm, '<hr class="border-white/10 my-6" />')
    // Paragraphs
    .replace(/\n\n/g, "</p><p class=\"my-3\">");

  // Wrap in paragraph if needed
  if (!html.startsWith("<")) {
    html = `<p class="my-3">${html}</p>`;
  }

  return html;
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

export function MarkdownViewer({
  content,
  path,
  isLoading,
  error,
  onBack
}: MarkdownViewerProps) {
  const [copied, setCopied] = useState(false);

  const handleCopyPath = async () => {
    const fullPath = `scenarios/test-genie/docs/${path}`;
    try {
      await navigator.clipboard.writeText(fullPath);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy path:", err);
    }
  };

  const breadcrumbItems = [
    { label: "Docs", onClick: onBack },
    { label: path.replace(/\.md$/, "") }
  ];

  if (isLoading) {
    return (
      <div
        className="flex-1 rounded-2xl border border-white/10 bg-white/[0.02] p-6"
        data-testid={selectors.docs.viewer}
      >
        <p className="text-slate-400">Loading document...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div
        className="flex-1 rounded-2xl border border-white/10 bg-white/[0.02] p-6"
        data-testid={selectors.docs.viewer}
      >
        <p className="text-red-400">Failed to load document: {error.message}</p>
        <Button variant="outline" size="sm" className="mt-4" onClick={onBack}>
          Go back
        </Button>
      </div>
    );
  }

  if (!content && !path) {
    return (
      <div
        className="flex-1 rounded-2xl border border-white/10 bg-white/[0.02] p-6"
        data-testid={selectors.docs.viewer}
      >
        <div className="text-center py-12">
          <p className="text-slate-400">Select a document from the sidebar to view it here.</p>
        </div>
      </div>
    );
  }

  const htmlContent = parseMarkdown(content);

  return (
    <div
      className="flex-1 rounded-2xl border border-white/10 bg-white/[0.02] p-6 overflow-hidden"
      data-testid={selectors.docs.viewer}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <Breadcrumb items={breadcrumbItems} />
        <Button
          variant="outline"
          size="sm"
          onClick={handleCopyPath}
          data-testid={selectors.docs.copyPath}
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
        className="prose prose-invert prose-sm max-w-none overflow-y-auto max-h-[calc(100vh-300px)]"
        dangerouslySetInnerHTML={{ __html: htmlContent }}
      />
    </div>
  );
}
