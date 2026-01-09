import { useState } from "react";
import { ChevronDown, ChevronRight, FileText, Search } from "lucide-react";
import type { DocsManifest } from "../../hooks/useDocs";

interface DocsSidebarProps {
  manifest: DocsManifest | undefined;
  selectedPath: string | null;
  onSelectDoc: (path: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
}

function cn(...classes: (string | boolean | undefined)[]): string {
  return classes.filter(Boolean).join(" ");
}

export function DocsSidebar({
  manifest,
  selectedPath,
  onSelectDoc,
  searchQuery,
  onSearchChange
}: DocsSidebarProps) {
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(manifest?.sections.map((s) => s.id) ?? [])
  );

  const toggleSection = (sectionId: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(sectionId)) {
        next.delete(sectionId);
      } else {
        next.add(sectionId);
      }
      return next;
    });
  };

  // Filter documents based on search query
  const filteredSections = manifest?.sections
    .map((section) => {
      if (!searchQuery.trim()) return section;

      const query = searchQuery.toLowerCase();
      const filteredDocs = section.documents.filter(
        (doc) =>
          doc.title.toLowerCase().includes(query) ||
          doc.path.toLowerCase().includes(query) ||
          (doc.description?.toLowerCase().includes(query) ?? false)
      );

      return { ...section, documents: filteredDocs };
    })
    .filter((section) => section.documents.length > 0);

  return (
    <aside
      className="w-64 shrink-0 rounded-xl border border-white/10 bg-white/5 p-4"
      data-testid="docs-sidebar"
    >
      {/* Search */}
      <div className="relative mb-4">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
        <input
          className="w-full rounded-lg border border-white/10 bg-black/30 pl-9 pr-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
          placeholder="Search docs..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          data-testid="docs-search"
        />
      </div>

      {/* Navigation */}
      <nav className="space-y-2">
        {!manifest && (
          <p className="text-sm text-slate-400 p-2">Loading documentation...</p>
        )}

        {filteredSections?.map((section) => (
          <div key={section.id}>
            <button
              type="button"
              onClick={() => toggleSection(section.id)}
              className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm font-medium text-slate-200 hover:bg-white/5"
            >
              {expandedSections.has(section.id) ? (
                <ChevronDown className="h-4 w-4 text-slate-400" />
              ) : (
                <ChevronRight className="h-4 w-4 text-slate-400" />
              )}
              {section.title}
              {section.visibility === "developers-only" && (
                <span className="ml-auto text-[10px] uppercase text-slate-500">Dev</span>
              )}
            </button>

            {expandedSections.has(section.id) && (
              <div className="ml-4 mt-1 space-y-1">
                {section.documents.map((doc) => (
                  <button
                    key={doc.path}
                    type="button"
                    onClick={() => onSelectDoc(doc.path)}
                    className={cn(
                      "flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition",
                      selectedPath === doc.path
                        ? "bg-blue-400/10 text-blue-400"
                        : "text-slate-400 hover:bg-white/5 hover:text-white"
                    )}
                  >
                    <FileText className="h-4 w-4" />
                    {doc.title}
                  </button>
                ))}
              </div>
            )}
          </div>
        ))}

        {searchQuery && filteredSections?.length === 0 && (
          <p className="text-sm text-slate-400 p-2">
            No documents match &quot;{searchQuery}&quot;
          </p>
        )}
      </nav>
    </aside>
  );
}
