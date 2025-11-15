import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StatsPanel } from "./components/StatsPanel";
import { GeneratorForm } from "./components/GeneratorForm";
import { TemplateGrid } from "./components/TemplateGrid";
import { BuildStatus } from "./components/BuildStatus";
import { ScenarioInventory } from "./components/scenario-inventory";
import { Card, CardHeader, CardTitle, CardContent } from "./components/ui/card";
import { Button } from "./components/ui/button";
import { Monitor, Zap, List } from "lucide-react";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1
    }
  }
});

type ViewMode = "generator" | "inventory";

function AppContent() {
  const [selectedTemplate, setSelectedTemplate] = useState("basic");
  const [currentBuildId, setCurrentBuildId] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("inventory");

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-blue-950 to-slate-950 text-slate-50">
      <div className="mx-auto max-w-7xl p-6">
        {/* Header */}
        <div className="mb-8 text-center">
          <div className="mb-3 flex items-center justify-center gap-3">
            <Monitor className="h-10 w-10 text-blue-400" />
            <h1 className="text-4xl font-bold">Scenario to Desktop</h1>
          </div>
          <p className="text-lg text-slate-300">
            Transform Vrooli scenarios into professional desktop applications
          </p>
        </div>

        {/* View Mode Selector */}
        <div className="mb-6 flex justify-center gap-3">
          <Button
            variant={viewMode === "inventory" ? "default" : "outline"}
            onClick={() => setViewMode("inventory")}
            className="gap-2"
          >
            <List className="h-4 w-4" />
            Scenario Inventory
          </Button>
          <Button
            variant={viewMode === "generator" ? "default" : "outline"}
            onClick={() => setViewMode("generator")}
            className="gap-2"
          >
            <Zap className="h-4 w-4" />
            Generate Desktop App
          </Button>
        </div>

        {/* System Stats - Only show in generator mode */}
        {viewMode === "generator" && (
          <div className="mb-8">
            <StatsPanel />
          </div>
        )}

        {/* Conditional Content */}
        {viewMode === "inventory" ? (
          <ScenarioInventory />
        ) : (
          <>
            {/* Main Content Grid */}
            <div className="grid gap-6 lg:grid-cols-2">
              {/* Generator Form */}
              <div>
                <GeneratorForm
                  selectedTemplate={selectedTemplate}
                  onTemplateChange={setSelectedTemplate}
                  onBuildStart={setCurrentBuildId}
                />
              </div>

              {/* Template Browser */}
              <Card>
                <CardHeader>
                  <CardTitle>Available Templates</CardTitle>
                </CardHeader>
                <CardContent>
                  <TemplateGrid
                    selectedTemplate={selectedTemplate}
                    onSelect={setSelectedTemplate}
                  />
                </CardContent>
              </Card>
            </div>

            {/* Build Status */}
            {currentBuildId && (
              <div className="mt-6">
                <BuildStatus buildId={currentBuildId} />
              </div>
            )}
          </>
        )}

        {/* Footer */}
        <div className="mt-12 text-center text-sm text-slate-400">
          <p>
            Built with ❤️ by the{" "}
            <a
              href="https://vrooli.com"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-400 hover:underline"
            >
              Vrooli Platform
            </a>
            {" | "}
            <a
              href="https://github.com/vrooli/vrooli"
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-400 hover:underline"
            >
              View on GitHub
            </a>
          </p>
        </div>
      </div>
    </div>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppContent />
    </QueryClientProvider>
  );
}
