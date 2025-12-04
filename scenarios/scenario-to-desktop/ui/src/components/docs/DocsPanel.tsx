import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { BookOpen, FileText, Loader2, Search } from "lucide-react";
import type { DocsContentResponse, DocsDocument, DocsManifest } from "../../lib/api";
import { fetchDocContent, fetchDocsManifest } from "../../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { cn } from "../../lib/utils";
import { logger } from "../../lib/logger";

function findDocTitle(manifest: DocsManifest | undefined, path: string | null): string {
  if (!manifest || !path) return "Document";
  for (const section of manifest.sections) {
    const match = section.documents.find((doc) => doc.path === path);
    if (match?.title) return match.title;
  }
  return path;
}

function SectionList({
  manifest,
  selectedPath,
  searchQuery,
  onSelect
}: {
  manifest?: DocsManifest;
  selectedPath: string | null;
  searchQuery: string;
  onSelect: (path: string) => void;
}) {
  if (!manifest) return null;

  const normalizedQuery = searchQuery.trim().toLowerCase();

  const sections = (manifest.sections || []).map((section) => {
    if (!normalizedQuery) return section;
    const filteredDocs = section.documents.filter((doc) => {
      const haystack = `${doc.title} ${doc.description || ""} ${doc.path}`.toLowerCase();
      return haystack.includes(normalizedQuery);
    });
    return { ...section, documents: filteredDocs };
  }).filter((section) => section.documents.length > 0);

  return (
    <div className="space-y-4">
      {sections.map((section) => (
        <div key={section.id} className="rounded-lg border border-slate-800/70 bg-slate-900/60 p-3">
          <div className="mb-2 text-xs font-semibold uppercase text-slate-400">{section.title}</div>
          <div className="space-y-1">
            {section.documents.map((doc: DocsDocument) => {
              const isActive = doc.path === selectedPath;
              return (
                <button
                  key={doc.path}
                  type="button"
                  className={cn(
                    "flex w-full items-start gap-2 rounded-md px-2 py-2 text-left text-sm transition",
                    isActive ? "bg-blue-600/20 text-blue-100 border border-blue-700/50" : "text-slate-200 hover:bg-slate-800/60"
                  )}
                  onClick={() => onSelect(doc.path)}
                >
                  <FileText className="h-4 w-4 shrink-0 text-blue-300" />
                  <div className="flex flex-col">
                    <span className="font-semibold">{doc.title}</span>
                    {doc.description && <span className="text-xs text-slate-400">{doc.description}</span>}
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

const markedLoader = (() => {
  let markedPromise: Promise<any> | null = null;
  return () => {
    if (!markedPromise) {
      markedPromise = import("https://cdn.jsdelivr.net/npm/marked/lib/marked.esm.js");
    }
    return markedPromise;
  };
})();

const mermaidLoader = (() => {
  let mermaidPromise: Promise<any> | null = null;
  return () => {
    if (!mermaidPromise) {
      mermaidPromise = import("https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs");
    }
    return mermaidPromise;
  };
})();

function escapeHtml(input: string): string {
  return input
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function sanitizeHtml(html: string): string {
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
}

export function DocsPanel() {
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [renderedHtml, setRenderedHtml] = useState<string | null>(null);
  const [renderError, setRenderError] = useState<string | null>(null);
  const contentRef = useRef<HTMLDivElement | null>(null);

  const {
    data: manifest,
    isLoading: loadingManifest,
    error: manifestError
  } = useQuery<DocsManifest>({
    queryKey: ["docs-manifest"],
    queryFn: fetchDocsManifest,
    staleTime: 5 * 60 * 1000
  });

  const defaultDoc = useMemo(() => {
    if (!manifest) return null;
    if (manifest.defaultDocument) return manifest.defaultDocument;
    const firstSection = manifest.sections?.[0];
    return firstSection?.documents?.[0]?.path ?? null;
  }, [manifest]);

  useEffect(() => {
    if (defaultDoc && !selectedPath) {
      setSelectedPath(defaultDoc);
    }
  }, [defaultDoc, selectedPath]);

  const {
    data: content,
    isLoading: loadingContent,
    error: contentError,
    refetch
  } = useQuery<DocsContentResponse>({
    queryKey: ["docs-content", selectedPath],
    queryFn: () => fetchDocContent(selectedPath || ""),
    enabled: !!selectedPath
  });

  const title = findDocTitle(manifest, selectedPath);

  useEffect(() => {
    let cancelled = false;
    async function render() {
      if (!content?.content) {
        setRenderedHtml(null);
        return;
      }
      setRenderError(null);
      try {
        const marked = await markedLoader();
        const renderer = new marked.Renderer();
        renderer.code = (code: string, infostring: string) => {
          const safeCode = typeof code === "string" ? code : String(code ?? "");
          const lang = typeof infostring === "string" ? infostring.trim() : "";
          if (lang === "mermaid") {
            return `<div class="mermaid">${safeCode}</div>`;
          }
          const escaped = escapeHtml(safeCode);
          const langClass = lang ? ` class="language-${lang}"` : "";
          return `<pre><code${langClass}>${escaped}</code></pre>`;
        };
        const html = marked.marked(content.content, {
          gfm: true,
          breaks: true,
          renderer
        });
        const safe = sanitizeHtml(html);
        if (!cancelled) setRenderedHtml(safe);
      } catch (err) {
        logger.error("Failed to render docs markdown", err);
        if (!cancelled) setRenderError("Failed to render markdown");
      }
    }
    render();
    return () => {
      cancelled = true;
    };
  }, [content]);

  useEffect(() => {
    async function renderMermaid() {
      if (!contentRef.current) return;
      const hasMermaid = contentRef.current.querySelector(".mermaid");
      if (!hasMermaid) return;
      try {
        const mermaid = await mermaidLoader();
        mermaid.initialize({ startOnLoad: false, theme: "dark" });
        await mermaid.run({
          nodes: contentRef.current.querySelectorAll(".mermaid")
        });
      } catch (err) {
        logger.error("Failed to render mermaid", err);
      }
    }
    renderMermaid();
  }, [renderedHtml]);

  return (
    <div className="space-y-6">
      <Card className="border-slate-800/70 bg-slate-900/70">
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-lg text-slate-50">
              <BookOpen className="h-5 w-5 text-blue-300" />
              Documentation
            </CardTitle>
            <p className="text-sm text-slate-300">
              Browse scenario-to-desktop docs in-app. Content mirrors the manifest at <code>docs/manifest.json</code>.
            </p>
          </div>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-3">
          <div className="md:col-span-1">
            <div className="mb-3 flex items-center gap-2 rounded-md border border-slate-800 bg-slate-900/60 px-3 py-2">
              <Search className="h-4 w-4 text-slate-400" />
              <input
                type="search"
                placeholder="Search docs..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full bg-transparent text-sm text-slate-100 outline-none placeholder:text-slate-500"
              />
            </div>
            {loadingManifest && <div className="text-slate-400">Loading manifest…</div>}
            {manifestError && <div className="text-red-300">Failed to load docs manifest</div>}
            {!loadingManifest && !manifestError && (
              <SectionList manifest={manifest} selectedPath={selectedPath} searchQuery={searchQuery} onSelect={setSelectedPath} />
            )}
          </div>
          <div className="md:col-span-2 space-y-3">
            <div className="rounded-lg border border-slate-800/70 bg-slate-950/70 p-4">
              <div className="mb-3 flex items-center justify-between gap-3">
                <div>
                  <div className="text-sm font-semibold text-slate-100">{title}</div>
                  {selectedPath && <div className="text-xs text-slate-400">{selectedPath}</div>}
                </div>
                <button
                  type="button"
                  className="rounded-md border border-slate-800 bg-slate-900 px-3 py-1 text-xs text-slate-200 hover:border-blue-700 hover:text-white"
                  onClick={() => {
                    if (selectedPath) {
                      navigator.clipboard?.writeText(selectedPath).catch(() => {});
                    }
                  }}
                  disabled={!selectedPath}
                >
                  Copy path
                </button>
              </div>
              {loadingContent && (
                <div className="flex items-center gap-2 text-slate-400">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Loading document…
                </div>
              )}
              {contentError && (
                <div className="text-red-300">
                  Failed to load document. <button onClick={() => refetch()} className="underline">Retry</button>
                </div>
              )}
              {!loadingContent && !contentError && renderError && (
                <div className="text-red-300">{renderError}</div>
              )}
              {!loadingContent && !contentError && !renderError && content && (
                <div
                  ref={contentRef}
                  className="prose prose-invert max-w-none"
                  dangerouslySetInnerHTML={{ __html: renderedHtml || "" }}
                />
              )}
              {!loadingContent && !contentError && !content && (
                <div className="text-slate-400">Select a document to view content.</div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
