import { useEffect, useRef, useState } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import { escapeHtml, highlightCodeBlocks } from "../utils/highlighting";
import { selectors } from "../../../consts/selectors";
import mermaid from "mermaid";
import { marked, Renderer, type Tokens } from "marked";

const encodeDiagram = (value: string) => encodeURIComponent(value);
const decodeDiagram = (value: string) => decodeURIComponent(value);

const sanitizeHtml = (html: string): string => {
  const parser = new DOMParser();
  const doc = parser.parseFromString(html, "text/html");
  const blockedTags = ["script", "style", "iframe", "object", "embed", "form", "link"];
  blockedTags.forEach((tag) => doc.querySelectorAll(tag).forEach((node) => node.remove()));

  const sanitizeAttributes = (el: Element) => {
    [...el.attributes].forEach((attr) => {
      const name = attr.name.toLowerCase();
      const value = attr.value;
      if (name.startsWith("on")) {
        el.removeAttribute(attr.name);
        return;
      }
      if (value && value.toLowerCase().includes("javascript:")) {
        el.removeAttribute(attr.name);
      }
    });
    [...el.children].forEach(sanitizeAttributes);
  };
  [...doc.body.children].forEach(sanitizeAttributes);
  return doc.body.innerHTML;
};

export type PreviewViewProps = {
  content?: string;
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
};

export function PreviewView({ content, isLoading, hasError, errorMessage }: PreviewViewProps) {
  const [htmlContent, setHtmlContent] = useState("");
  const [renderError, setRenderError] = useState("");
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function renderMarkdown() {
      if (!content) {
        setHtmlContent("");
        return;
      }
      setRenderError("");
      try {
        const renderer = new Renderer();
        renderer.code = ({ text, lang }: Tokens.Code) => {
          const langValue = typeof lang === "string" ? lang.trim().toLowerCase() : "";
          if (langValue === "mermaid") {
            return `<div class=\"ko-mermaid\" data-diagram=\"${encodeDiagram(text)}\"></div>`;
          }
          const escaped = escapeHtml(text);
          const className = langValue ? ` class=\"language-${langValue}\"` : "";
          return `<pre><code${className}>${escaped}</code></pre>`;
        };
        const raw = await marked.parse(content, {
          gfm: true,
          breaks: true,
          renderer,
        });
        const safe = sanitizeHtml(raw);
        if (!cancelled) {
          setHtmlContent(safe);
        }
      } catch {
        if (!cancelled) {
          setRenderError("Failed to render markdown preview.");
        }
      }
    }
    renderMarkdown();
    return () => {
      cancelled = true;
    };
  }, [content]);

  useEffect(() => {
    if (!htmlContent || !containerRef.current) return;
    void highlightCodeBlocks(containerRef.current);
  }, [htmlContent]);

  useEffect(() => {
    let cancelled = false;
    async function renderMermaid() {
      if (!containerRef.current) return;
      const nodes = Array.from(containerRef.current.querySelectorAll<HTMLElement>(".ko-mermaid"));
      if (nodes.length === 0) return;
      try {
        mermaid.initialize({ startOnLoad: false, theme: "dark", securityLevel: "strict" });
        for (const node of nodes) {
          const diagram = node.dataset.diagram ? decodeDiagram(node.dataset.diagram) : "";
          if (!diagram.trim()) continue;
          const id = `ko-mermaid-${Math.random().toString(36).slice(2)}`;
          const { svg } = await mermaid.render(id, diagram);
          if (!cancelled) {
            node.innerHTML = svg;
            node.classList.add("ko-mermaid-rendered");
          }
        }
      } catch {
        // Mermaid rendering is best-effort.
      }
    }
    void renderMermaid();
    return () => {
      cancelled = true;
    };
  }, [htmlContent]);

  if (isLoading) {
    return (
      <div className="ko-code-state">
        <Loader2 className="h-5 w-5 animate-spin" />
        <span>Loading document...</span>
      </div>
    );
  }

  if (hasError) {
    return (
      <div className="ko-code-state ko-code-error">
        <AlertTriangle className="h-5 w-5" />
        <span>{errorMessage}</span>
      </div>
    );
  }

  if (!content) {
    return <div className="ko-code-state">Select a document to preview.</div>;
  }

  if (renderError) {
    return (
      <div className="ko-code-state ko-code-error">
        <AlertTriangle className="h-5 w-5" />
        <span>{renderError}</span>
      </div>
    );
  }

  return (
    <div
      className="ko-markdown-view"
      data-testid={selectors.viewer.previewView}
      ref={containerRef}
      dangerouslySetInnerHTML={{ __html: htmlContent }}
    />
  );
}
