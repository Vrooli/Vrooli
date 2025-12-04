import { useState, useEffect, useCallback } from "react";
import { DocsSidebar } from "./DocsSidebar";
import { MarkdownViewer } from "./MarkdownViewer";
import { useDocsManifest, useDocContent } from "../../hooks/useDocs";

/**
 * Parse the path parameter from the URL hash.
 * Expected format: #docs?path=some/doc.md
 */
function getPathFromHash(): string | null {
  const hash = window.location.hash;
  if (!hash.startsWith("#docs")) return null;

  const queryStart = hash.indexOf("?");
  if (queryStart === -1) return null;

  const params = new URLSearchParams(hash.slice(queryStart + 1));
  const path = params.get("path");
  return path ? decodeURIComponent(path) : null;
}

export function DocsPage() {
  const { data: manifest, isLoading: manifestLoading, error: manifestError } = useDocsManifest();
  const [selectedPath, setSelectedPath] = useState<string | null>(() => {
    // Initialize from URL hash if present
    return getPathFromHash();
  });
  const [searchQuery, setSearchQuery] = useState("");

  // Listen for hash changes (e.g., from Learn more button on another tab)
  useEffect(() => {
    const handleHashChange = () => {
      const pathFromHash = getPathFromHash();
      if (pathFromHash) {
        setSelectedPath(pathFromHash);
        // Clear the path from hash to avoid confusion on subsequent navigations
        // Keep just #docs so the tab stays active
        window.history.replaceState(null, "", "#docs");
      }
    };

    // Also check current hash on mount (in case it wasn't caught by initial state)
    const pathFromHash = getPathFromHash();
    if (pathFromHash) {
      setSelectedPath(pathFromHash);
      window.history.replaceState(null, "", "#docs");
    }

    // Listen for hash changes
    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  // Auto-select default document when manifest loads (only if no path selected)
  useEffect(() => {
    if (manifest && !selectedPath) {
      // Double-check there's no URL path before using default
      const pathFromHash = getPathFromHash();
      if (pathFromHash) {
        setSelectedPath(pathFromHash);
        window.history.replaceState(null, "", "#docs");
      } else {
        setSelectedPath(manifest.defaultDocument);
      }
    }
  }, [manifest, selectedPath]);

  const { data: content, isLoading: contentLoading, error: contentError } = useDocContent(selectedPath);

  const handleSelectDoc = (path: string) => {
    setSelectedPath(path);
  };

  const handleBack = () => {
    if (manifest) {
      setSelectedPath(manifest.defaultDocument);
    }
  };

  // If manifest failed to load, show a fallback with basic info
  if (manifestError) {
    return (
      <div className="space-y-6">
        <section className="rounded-xl border border-white/10 bg-white/5 p-6">
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Documentation</p>
          <h1 className="mt-2 text-2xl font-semibold">Autoheal Docs</h1>
          <p className="mt-4 text-slate-300">
            Documentation is available in the <code className="bg-black/50 px-1.5 py-0.5 rounded text-blue-400">scenarios/vrooli-autoheal/docs/</code> directory.
          </p>
          <p className="mt-4 text-sm text-slate-400">
            The docs API is not currently available. Ensure the autoheal scenario is running with the latest API.
          </p>
          <div className="mt-6">
            <p className="text-sm text-slate-300 mb-2">Available documents:</p>
            <ul className="list-disc list-inside text-sm text-slate-400 space-y-1">
              <li>QUICKSTART.md - Getting started guide</li>
              <li>GLOSSARY.md - Terminology reference</li>
              <li>concepts/architecture.md - System design</li>
              <li>guides/dashboard-guide.md - Using the dashboard</li>
              <li>reference/api-endpoints.md - API documentation</li>
              <li>PROGRESS.md - Development progress</li>
              <li>PROBLEMS.md - Known issues</li>
            </ul>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="space-y-6" data-testid="docs-page">
      {/* Header */}
      <section className="rounded-xl border border-white/10 bg-gradient-to-r from-blue-500/10 via-transparent to-slate-500/10 p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Documentation</p>
        <h1 className="mt-2 text-2xl font-semibold">Autoheal Docs</h1>
        <p className="mt-2 text-sm text-slate-300">
          Learn about health checks, architecture, and best practices.
        </p>
      </section>

      {/* Content */}
      <div className="flex gap-6">
        <DocsSidebar
          manifest={manifest}
          selectedPath={selectedPath}
          onSelectDoc={handleSelectDoc}
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
        />

        <MarkdownViewer
          content={content ?? ""}
          path={selectedPath ?? ""}
          isLoading={contentLoading || manifestLoading}
          error={contentError}
          onBack={handleBack}
        />
      </div>
    </div>
  );
}

export { DocsSidebar, MarkdownViewer };
