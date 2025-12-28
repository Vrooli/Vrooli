import { useState, useEffect } from "react";
import { DocsSidebar } from "./DocsSidebar";
import { MarkdownViewer } from "./MarkdownViewer";
import { useDocsManifest, useDocContent } from "../../hooks/useDocs";
import { Card, CardContent } from "../ui/card";

interface DocsPageProps {
  onBack?: () => void;
}

export function DocsPage({ onBack }: DocsPageProps) {
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
