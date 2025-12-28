import { useState, useEffect, useCallback } from "react";
import { DocsSidebar } from "./DocsSidebar";
import { MarkdownViewer } from "./MarkdownViewer";
import { useDocsManifest, useDocContent } from "../../hooks/useDocs";
import { Card, CardContent } from "../ui/card";

interface DocsPageProps {
  onBack?: () => void;
  /** Initial doc path from URL (e.g., "guides/vps-setup") */
  initialDocPath?: string | null;
  /** Callback when doc path changes - updates URL */
  onDocPathChange?: (path: string) => void;
}

export function DocsPage({ onBack, initialDocPath, onDocPathChange }: DocsPageProps) {
  const { data: manifest, isLoading: manifestLoading, error: manifestError } = useDocsManifest();
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  // Initialize from URL path or default document
  useEffect(() => {
    if (!manifest) return;

    // If we have an initial path from URL, validate and use it
    if (initialDocPath) {
      // Check if the path exists in the manifest
      const pathWithExt = initialDocPath.endsWith(".md") ? initialDocPath : `${initialDocPath}.md`;
      const allDocs = manifest.sections.flatMap(s => s.documents);
      const docExists = allDocs.some(d => d.path === pathWithExt || d.path === initialDocPath);

      if (docExists) {
        setSelectedPath(pathWithExt);
        return;
      }
    }

    // Fall back to default document if no valid path from URL
    if (!selectedPath) {
      setSelectedPath(manifest.defaultDocument);
    }
  }, [manifest, initialDocPath, selectedPath]);

  const { data: content, isLoading: contentLoading, error: contentError } = useDocContent(selectedPath);

  const handleSelectDoc = useCallback((path: string) => {
    setSelectedPath(path);
    // Update URL to reflect the selected doc
    if (onDocPathChange) {
      // Remove .md extension for cleaner URLs
      const cleanPath = path.replace(/\.md$/, "");
      onDocPathChange(cleanPath);
    }
  }, [onDocPathChange]);

  const handleBack = useCallback(() => {
    if (manifest) {
      const defaultPath = manifest.defaultDocument;
      setSelectedPath(defaultPath);
      if (onDocPathChange) {
        const cleanPath = defaultPath.replace(/\.md$/, "");
        onDocPathChange(cleanPath);
      }
    }
  }, [manifest, onDocPathChange]);

  // If manifest failed to load, show a fallback with basic info
  if (manifestError) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="py-6">
            <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Documentation</p>
            <h1 className="mt-2 text-2xl font-semibold text-white">Scenario-to-Cloud Docs</h1>
            <p className="mt-4 text-slate-300">
              Documentation is available in the <code className="bg-black/50 px-1.5 py-0.5 rounded text-blue-400">scenarios/scenario-to-cloud/docs/</code> directory.
            </p>
            <p className="mt-4 text-sm text-slate-400">
              The docs API is not currently running. Start the scenario-to-cloud scenario to enable in-app documentation browsing.
            </p>
            <div className="mt-6">
              <p className="text-sm text-slate-300 mb-2">Available documents:</p>
              <ul className="list-disc list-inside text-sm text-slate-400 space-y-1">
                <li>QUICKSTART.md - Getting started guide</li>
                <li>GLOSSARY.md - Key terms and concepts</li>
                <li>guides/manifest-reference.md - Configuration reference</li>
                <li>guides/vps-setup.md - VPS preparation</li>
                <li>guides/troubleshooting.md - Common issues</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <Card>
        <CardContent className="py-6">
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Documentation</p>
          <h1 className="mt-2 text-2xl font-semibold text-white">
            {manifest?.title ?? "Scenario-to-Cloud Docs"}
          </h1>
          <p className="mt-2 text-sm text-slate-300">
            {manifest?.description ?? "Learn how to deploy Vrooli scenarios to production VPS."}
          </p>
        </CardContent>
      </Card>

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
