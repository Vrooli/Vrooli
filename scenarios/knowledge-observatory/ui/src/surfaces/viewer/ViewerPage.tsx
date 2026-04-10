// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import { useEffect, useMemo, useState } from "react";
import { FileText, RefreshCw } from "lucide-react";
import { ErrorBoundary } from "../../shared/components/ErrorBoundary";
import { PageShell } from "../../shared/components/PageShell";
import { Panel, PanelHeader } from "../../shared/components/Panel";
import { SectionErrorState } from "../../shared/components/SectionErrorState";
import type { Route } from "../../shared/controllers/routeController";
import { useDocViewer } from "../../shared/hooks/viewerHooks";
import { Button } from "../../shared/ui/button";
import { selectors } from "../../consts/selectors";
import { CodeView } from "./components/CodeView";
import { PreviewView } from "./components/PreviewView";
import { ResetPanel } from "./components/ResetPanel";
import { ViewModeToggle } from "./components/ViewModeToggle";

export type ViewerPageProps = {
  onNavigate: (route: Route) => void;
};

const parseViewerPath = (hash: string) => {
  const normalized = hash.replace(/^#\/?/, "");
  const [routePart, query] = normalized.split("?");
  if (!routePart?.startsWith("viewer")) return "";
  if (!query) return "";
  const params = new URLSearchParams(query);
  return params.get("path") ?? "";
};

const updateViewerHash = (path: string) => {
  if (typeof window === "undefined") return;
  const trimmed = path.trim();
  if (!trimmed) {
    window.location.hash = "#/viewer";
    return;
  }
  window.location.hash = `#/viewer?path=${encodeURIComponent(trimmed)}`;
};

export function ViewerPage({ onNavigate }: ViewerPageProps) {
  const initialPath = useMemo(
    () => (typeof window === "undefined" ? "" : parseViewerPath(window.location.hash)),
    []
  );
  const {
    path,
    setPath,
    viewMode,
    setViewMode,
    content,
    meta,
    isLoading,
    hasError,
    errorMessage,
    refresh,
    resetResult,
    resetError,
    isResetting,
    runReset,
  } = useDocViewer(initialPath);
  const [pathInput, setPathInput] = useState(initialPath);

  useEffect(() => {
    setPathInput(path);
  }, [path]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const handleHashChange = () => {
      const nextPath = parseViewerPath(window.location.hash);
      if (nextPath !== path) {
        setPath(nextPath);
        setPathInput(nextPath);
      }
    };
    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, [path, setPath]);

  const handleLoad = () => {
    const trimmed = pathInput.trim();
    setPath(trimmed);
    updateViewerHash(trimmed);
  };

  const handlePreview = (config: { maxAgeDays: number; keepMinEntries: number }) =>
    runReset({ ...config, previewOnly: true });

  const handleApply = (config: { maxAgeDays: number; keepMinEntries: number }) =>
    runReset({ ...config, previewOnly: false });

  const canOpen = Boolean(pathInput.trim());

  return (
    <ErrorBoundary
      fallback={({ error, reset }) => (
        <PageShell>
          <SectionErrorState
            title="Document Viewer Unavailable"
            description="The viewer UI encountered an unexpected error. You can retry or return to the dashboard."
            errorMessage={error.message}
            actions={[
              { label: "Retry Section", onClick: reset },
              { label: "Back to Dashboard", onClick: () => onNavigate("dashboard"), variant: "secondary" },
            ]}
          />
        </PageShell>
      )}
    >
      <PageShell className="ko-viewer-shell">
        <Panel className="ko-viewer-panel">
          <PanelHeader
            title="Document Viewer"
            description="Load and inspect documentation with code and preview modes."
            icon={<FileText className="h-5 w-5 ko-icon" />}
          />
          <div className="ko-viewer-toolbar">
            <input
              className="ko-input ko-viewer-input"
              data-testid={selectors.viewer.pathInput}
              value={pathInput}
              onChange={(event) => setPathInput(event.target.value)}
              placeholder="scenarios/knowledge-observatory/docs/QUICKSTART.md"
            />
            <Button
              type="button"
              variant="primary"
              size="compact"
              onClick={handleLoad}
              disabled={!canOpen}
              data-testid={selectors.viewer.loadButton}
            >
              Load
            </Button>
            <Button type="button" variant="outline" size="compact" onClick={refresh} disabled={!path.trim()}>
              <RefreshCw className="h-4 w-4 mr-2" />
              Refresh
            </Button>
          </div>
          <p className="ko-input-help">
            Use repo-relative paths (for example: <span className="ko-code">scenarios/knowledge-observatory/docs/manifest.json</span>).
          </p>
        </Panel>

        <div className="ko-viewer-grid">
          <Panel className="ko-viewer-panel">
            <PanelHeader title="Metadata & Reset" description="Context and cleanup controls." />
            <div className="ko-viewer-meta">
              {meta ? (
                <>
                  <div>
                    <p className="ko-meta">Path</p>
                    <p className="ko-text-sm">{meta.path}</p>
                  </div>
                  <div className="ko-viewer-meta-row">
                    <div>
                      <p className="ko-meta">Doc type</p>
                      <p className="ko-text-sm">{meta.docTypeLabel}</p>
                    </div>
                    <div>
                      <p className="ko-meta">Size</p>
                      <p className="ko-text-sm">{meta.sizeLabel}</p>
                    </div>
                  </div>
                  <div>
                    <p className="ko-meta">Last modified</p>
                    <p className="ko-text-sm">{meta.modifiedLabel}</p>
                  </div>
                </>
              ) : (
                <p className="ko-text-sm ko-subtle">Select a document to view metadata.</p>
              )}
            </div>

            <div className="mt-6">
              <PanelHeader title="Reset/Clean" description="Remove stale entries from PROBLEMS and PROGRESS docs." />
              <ResetPanel
                canReset={Boolean(meta?.canReset)}
                defaults={meta?.resetDefaults ?? { maxAgeDays: 30, keepMinEntries: 3 }}
                isBusy={isResetting}
                resetError={resetError}
                result={resetResult}
                onPreview={handlePreview}
                onApply={handleApply}
              />
            </div>
          </Panel>

          <Panel className="ko-viewer-panel">
            <PanelHeader title="Document" description="Switch between code, preview, or split view." />
            <ViewModeToggle mode={viewMode} onChange={setViewMode} />
            <div className={viewMode === "split" ? "ko-viewer-split" : "ko-viewer-stack"}>
              {(viewMode === "code" || viewMode === "split") && (
                <CodeView
                  content={content?.content}
                  path={content?.path}
                  isLoading={isLoading}
                  hasError={hasError}
                  errorMessage={errorMessage}
                />
              )}
              {(viewMode === "preview" || viewMode === "split") && (
                <PreviewView
                  content={content?.content}
                  isLoading={isLoading}
                  hasError={hasError}
                  errorMessage={errorMessage}
                />
              )}
            </div>
          </Panel>
        </div>
      </PageShell>
    </ErrorBoundary>
  );
}
