import { useState, useEffect } from "react";
import { DocsSidebar } from "./DocsSidebar";
import { MarkdownViewer } from "./MarkdownViewer";
import { useDocsManifest, useDocContent } from "../../hooks/useDocs";

export function DocsPage() {
  const { data: manifest, isLoading: manifestLoading, error: manifestError } = useDocsManifest();
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  // Auto-select default document when manifest loads
  useEffect(() => {
    if (manifest && !selectedPath) {
      setSelectedPath(manifest.defaultDocument);
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
        <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Documentation</p>
          <h1 className="mt-2 text-2xl font-semibold">Test Genie Docs</h1>
          <p className="mt-4 text-slate-300">
            Documentation is available in the <code className="bg-black/50 px-1.5 py-0.5 rounded text-cyan-400">scenarios/test-genie/docs/</code> directory.
          </p>
          <p className="mt-4 text-sm text-slate-400">
            The docs API is not currently running. Start the test-genie scenario to enable in-app documentation browsing.
          </p>
          <div className="mt-6">
            <p className="text-sm text-slate-300 mb-2">Available documents:</p>
            <ul className="list-disc list-inside text-sm text-slate-400 space-y-1">
              <li>QUICKSTART.md - Getting started guide</li>
              <li>synchronous-execution-guide.md - Execution patterns</li>
              <li>sync-execution-cheatsheet.md - Quick reference</li>
              <li>RESEARCH.md - Research notes</li>
              <li>SEAMS.md - Architecture seams</li>
              <li>PROBLEMS.md - Known issues</li>
              <li>PROGRESS.md - Development progress</li>
            </ul>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <section className="rounded-2xl border border-white/10 bg-gradient-to-r from-emerald-500/10 via-transparent to-cyan-500/10 p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Documentation</p>
        <h1 className="mt-2 text-2xl font-semibold">Test Genie Docs</h1>
        <p className="mt-2 text-sm text-slate-300">
          Learn about test phases, execution patterns, and best practices.
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
