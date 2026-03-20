// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
import { useState } from "react";
import { Layout, PenLine, GitBranch } from "lucide-react";
import { SchemeList } from "./components/SchemeList";
import { TextCapture } from "./components/TextCapture";
import { CanvasView } from "./components/CanvasView";
import { GraphView } from "./components/GraphView";
import { ProviderStatus } from "./components/ProviderStatus";
import { ConnectionStatus } from "./components/ConnectionStatus";
import { ExportButton } from "./components/ExportButton";
import { PanelErrorBoundary } from "./components/PanelErrorBoundary";
import type { Scheme } from "./lib/types";

type ViewMode = "canvas" | "graph";

export default function App() {
  const [activeScheme, setActiveScheme] = useState<Scheme | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("canvas");

  return (
    <div data-testid="app-root" className="h-full flex bg-slate-950 text-slate-50">
      {/* Sidebar */}
      <SchemeList activeSchemeId={activeScheme?.id ?? null} onSelect={setActiveScheme} />

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0">
        <ConnectionStatus />
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-2 border-b border-white/10 bg-slate-900/50">
          <h1 className="text-sm font-medium truncate">
            {activeScheme ? activeScheme.name : "Stream of Consciousness"}
          </h1>
          <ProviderStatus />
          {activeScheme && (
            <div className="flex items-center gap-1">
              <ExportButton schemeId={activeScheme.id} schemeName={activeScheme.name} />
              <button
                data-testid="view-canvas-btn"
                onClick={() => setViewMode("canvas")}
                className={`p-1.5 rounded text-sm ${viewMode === "canvas" ? "bg-white/10 text-white" : "text-slate-400 hover:text-white"}`}
                aria-label="Canvas view"
                aria-pressed={viewMode === "canvas"}
              >
                <Layout className="h-4 w-4" aria-hidden="true" />
              </button>
              <button
                data-testid="view-graph-btn"
                onClick={() => setViewMode("graph")}
                className={`p-1.5 rounded text-sm ${viewMode === "graph" ? "bg-white/10 text-white" : "text-slate-400 hover:text-white"}`}
                aria-label="Graph view"
                aria-pressed={viewMode === "graph"}
              >
                <GitBranch className="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
          )}
        </div>

        {/* Content area */}
        {activeScheme ? (
          <>
            <PanelErrorBoundary panelName={viewMode === "canvas" ? "Canvas" : "Graph"} key={viewMode}>
              {viewMode === "canvas" ? (
                <CanvasView schemeId={activeScheme.id} />
              ) : (
                <GraphView schemeId={activeScheme.id} />
              )}
            </PanelErrorBoundary>
            <PanelErrorBoundary panelName="Text Capture">
              <TextCapture schemeId={activeScheme.id} />
            </PanelErrorBoundary>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <PenLine className="h-12 w-12 text-slate-700 mx-auto mb-4" />
              <h2 className="text-lg font-medium text-slate-400">Select or create a scheme</h2>
              <p className="text-sm text-slate-600 mt-1">Use the sidebar to get started</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
