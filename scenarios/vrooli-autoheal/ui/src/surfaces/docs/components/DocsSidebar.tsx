import { useEffect, useState } from "react";
import { ChevronDown, ChevronRight, FileText, Search, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import type { DocsManifest } from "../../../hooks/useDocs";
import { Input } from "../../../shared/ui/primitives";

interface DocsSidebarProps {
  manifest: DocsManifest | undefined;
  selectedPath: string | null;
  onSelectDoc: (path: string) => void;
  searchQuery: string;
  onSearchChange: (query: string) => void;
}

const MOBILE_MEDIA_QUERY = "(max-width: 1023px)";

function getIsMobileViewport(): boolean {
  return typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia(MOBILE_MEDIA_QUERY).matches;
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
  const [isMobile, setIsMobile] = useState(getIsMobileViewport);
  const [mobileNavOpen, setMobileNavOpen] = useState(() => !getIsMobileViewport());
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(manifest?.sections.map((s) => s.id) ?? [])
  );

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      setIsMobile(false);
      setMobileNavOpen(true);
      return;
    }
    const mediaQuery = window.matchMedia(MOBILE_MEDIA_QUERY);
    const updateViewport = () => {
      const mobile = mediaQuery.matches;
      setIsMobile(mobile);
      if (!mobile) {
        setMobileNavOpen(true);
      }
    };

    updateViewport();
    mediaQuery.addEventListener("change", updateViewport);
    return () => mediaQuery.removeEventListener("change", updateViewport);
  }, []);

  useEffect(() => {
    if (selectedPath && isMobile) {
      setMobileNavOpen(false);
    }
  }, [isMobile, selectedPath]);

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
      className="w-full shrink-0 rounded-xl border border-border-default/70 bg-surface-elevated/40 p-3 sm:p-4 lg:w-64"
      data-testid="docs-sidebar"
    >
      {isMobile && (
        <button
          type="button"
          onClick={() => setMobileNavOpen((prev) => !prev)}
          className="mb-3 flex w-full items-center justify-between rounded-lg border border-border-default/70 bg-surface-overlay/40 px-3 py-2 text-sm font-medium text-text-primary transition-colors hover:bg-surface-overlay/70"
          aria-expanded={mobileNavOpen}
          aria-controls="docs-sidebar-content"
        >
          <span>Browse documents</span>
          {mobileNavOpen ? <PanelLeftClose className="h-4 w-4" /> : <PanelLeftOpen className="h-4 w-4" />}
        </button>
      )}

      {(mobileNavOpen || !isMobile) && (
        <div id="docs-sidebar-content">
      {/* Search */}
          <div className="relative mb-4">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted/70" />
            <Input
              className="pl-9"
              placeholder="Search docs..."
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              data-testid="docs-search"
            />
          </div>

      {/* Navigation */}
          <nav className="max-h-[min(45vh,22rem)] space-y-2 overflow-y-auto pr-1 lg:max-h-[calc(100dvh-16rem)]">
            {!manifest && (
              <p className="p-2 text-sm text-text-muted">Loading documentation...</p>
            )}

            {filteredSections?.map((section) => (
              <div key={section.id}>
                <button
                  type="button"
                  onClick={() => toggleSection(section.id)}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm font-medium text-text-primary transition-colors hover:bg-surface-overlay/50"
                >
                  {expandedSections.has(section.id) ? (
                    <ChevronDown className="h-4 w-4 text-text-muted" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-text-muted" />
                  )}
                  {section.title}
                  {section.visibility === "developers-only" && (
                    <span className="ml-auto text-[10px] uppercase text-text-muted/70">Dev</span>
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
                            ? "bg-accent-primary/20 text-accent-primary"
                            : "text-text-muted hover:bg-surface-overlay/50 hover:text-text-primary"
                        )}
                        title={doc.path}
                      >
                        <FileText className="h-4 w-4 shrink-0" />
                        <span className="truncate text-left">{doc.title}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ))}

            {searchQuery && filteredSections?.length === 0 && (
              <p className="p-2 text-sm text-text-muted">
                No documents match &quot;{searchQuery}&quot;
              </p>
            )}
          </nav>
        </div>
      )}
    </aside>
  );
}
