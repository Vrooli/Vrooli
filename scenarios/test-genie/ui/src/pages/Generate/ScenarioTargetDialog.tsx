import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, ChevronLeft, ChevronRight, X, Check } from "lucide-react";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { ScenarioDirectoryEntry } from "../../hooks/useScenarios";
import { fetchScenarioFiles, type ScenarioFileNode } from "../../lib/api";
import { cn } from "../../lib/utils";

interface ScenarioTargetDialogProps {
  open: boolean;
  onClose: () => void;
  initialScenario: string;
  initialTargets: string[];
  scenarios: ScenarioDirectoryEntry[];
  onSave: (scenario: string, targets: string[]) => void;
}

const normalizePath = (raw: string): string => {
  const trimmed = raw.trim().replace(/^\/+/, "");
  if (!trimmed) return "";
  if (trimmed.startsWith("api/") || trimmed.startsWith("ui/")) return trimmed;
  if (trimmed === "api" || trimmed === "ui") return `${trimmed}/`;
  // Default to api/ when not specified to keep scope safe
  return `api/${trimmed}`;
};

function useScenarioFilesQuery(scenarioName: string, path: string, search: string) {
  return useQuery({
    queryKey: ["scenario-files", scenarioName, path, search],
    queryFn: () =>
      fetchScenarioFiles(scenarioName, search ? { search, limit: 200 } : { path }),
    enabled: Boolean(scenarioName),
    staleTime: 10_000
  });
}

interface FileBrowserProps {
  scenarioName: string;
  selectedPaths: string[];
  onSelectPath: (path: string) => void;
  onRemovePath: (path: string) => void;
}

function FileBrowser({ scenarioName, selectedPaths, onSelectPath, onRemovePath }: FileBrowserProps) {
  const [currentPath, setCurrentPath] = useState("");
  const [searchTerm, setSearchTerm] = useState("");
  const { data: nodes = [], isLoading, error, refetch } = useScenarioFilesQuery(
    scenarioName,
    currentPath,
    searchTerm
  );

  useEffect(() => {
    setCurrentPath("");
    setSearchTerm("");
    refetch();
  }, [scenarioName, refetch]);

  const breadcrumb = useMemo(() => {
    if (!currentPath) return ["root"];
    const parts = currentPath.split("/").filter(Boolean);
    return ["root", ...parts];
  }, [currentPath]);

  const handleNavigate = (path: string) => {
    setCurrentPath(path);
    setSearchTerm("");
  };

  const handleAdd = (path: string) => {
    const normalized = normalizePath(path);
    if (!normalized) return;
    if (!selectedPaths.includes(normalized)) {
      onSelectPath(normalized);
    }
  };

  const handleManualAdd = () => {
    if (!searchTerm.trim()) return;
    handleAdd(searchTerm);
    setSearchTerm("");
  };

  const isRoot = currentPath === "" || currentPath === "/";
  const parentPath = useMemo(() => {
    if (isRoot) return "";
    const parts = currentPath.split("/").filter(Boolean);
    if (parts.length <= 1) return "";
    return parts.slice(0, -1).join("/");
  }, [currentPath, isRoot]);

  const content = (() => {
    if (!scenarioName) {
      return <p className="text-sm text-slate-400">Select a scenario first.</p>;
    }
    if (isLoading) {
      return <p className="text-sm text-slate-300">Loading paths...</p>;
    }
    if (error) {
      return <p className="text-sm text-rose-300">Failed to load paths.</p>;
    }
    if (!nodes || nodes.length === 0) {
      return <p className="text-sm text-slate-400">No paths found.</p>;
    }
    const isSearch = Boolean(searchTerm.trim());
    return (
      <div className="divide-y divide-white/5 rounded-xl border border-white/10 bg-black/20">
        {nodes.map((node) => (
          <div
            key={node.path}
            className="flex items-center justify-between px-4 py-3"
          >
            <div
              className={cn(
                "flex items-center gap-3",
                node.isDir && !isSearch ? "cursor-pointer" : "cursor-default"
              )}
              onClick={() => {
                if (node.isDir && !isSearch) handleNavigate(node.path);
              }}
            >
              <span
                className={cn(
                  "flex h-8 w-8 items-center justify-center rounded border text-xs",
                  node.isDir
                    ? "border-cyan-400/60 bg-cyan-400/10 text-cyan-200"
                    : "border-white/20 bg-white/5 text-white"
                )}
              >
                {node.isDir ? "DIR" : "FILE"}
              </span>
              <div>
                <p className="text-sm font-semibold text-white">{node.name}</p>
                <p className="text-xs text-slate-400">{node.path}</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {node.isDir && !isSearch && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleNavigate(node.path)}
                >
                  <ChevronRight className="mr-1 h-4 w-4" />
                  Open
                </Button>
              )}
              <Button
                size="sm"
                variant={selectedPaths.includes(normalizePath(node.path)) ? "outline" : "default"}
                className={selectedPaths.includes(normalizePath(node.path)) ? "border-cyan-400 text-white bg-cyan-400/10" : ""}
                onClick={() => handleAdd(node.path)}
              >
                {selectedPaths.includes(normalizePath(node.path)) ? (
                  <>
                    <Check className="mr-1 h-4 w-4" />
                    Selected
                  </>
                ) : (
                  "Add"
                )}
              </Button>
            </div>
          </div>
        ))}
      </div>
    );
  })();

  return (
    <div className="space-y-4" data-testid={selectors.generate.targetSelector}>
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
          <span className="uppercase tracking-[0.25em] text-slate-500">Path</span>
          <div className="flex items-center gap-1">
            {breadcrumb.map((crumb, idx) => (
              <span key={crumb} className="flex items-center gap-1">
                <span>{crumb}</span>
                {idx < breadcrumb.length - 1 && <ChevronRight className="h-3 w-3 text-slate-600" />}
              </span>
            ))}
          </div>
        </div>
        <div className="flex gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
            <input
              className="w-72 rounded-lg border border-white/10 bg-black/30 py-2 pl-9 pr-3 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
              placeholder="Search or type a path (e.g., api/services/user/)"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              spellCheck={false}
            />
          </div>
          <Button variant="outline" onClick={handleManualAdd} disabled={!searchTerm.trim()}>
            Add typed path
          </Button>
          {!isRoot && !searchTerm && (
            <Button variant="outline" onClick={() => handleNavigate(parentPath)}>
              <ChevronLeft className="mr-1 h-4 w-4" />
              Up one
            </Button>
          )}
        </div>
      </div>

      {content}

      <div className="flex flex-wrap gap-2">
        {selectedPaths.length === 0 ? (
          <p className="text-xs text-slate-400">No targets selected. Add files or folders to focus generation.</p>
        ) : (
          selectedPaths.map((path) => (
            <span
              key={path}
              className="inline-flex items-center gap-2 rounded-full border border-cyan-400/50 bg-cyan-400/10 px-3 py-1 text-xs text-cyan-100"
              data-testid={selectors.generate.targetItem}
            >
              {path}
              <button
                type="button"
                onClick={() => onRemovePath(path)}
                className="text-cyan-200 hover:text-white"
                aria-label={`Remove ${path}`}
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          ))
        )}
      </div>
    </div>
  );
}

export function ScenarioTargetDialog({
  open,
  onClose,
  scenarios,
  initialScenario,
  initialTargets,
  onSave
}: ScenarioTargetDialogProps) {
  const [step, setStep] = useState<1 | 2>(1);
  const [draftScenario, setDraftScenario] = useState(initialScenario);
  const [draftTargets, setDraftTargets] = useState<string[]>(initialTargets);
  const [scenarioSearch, setScenarioSearch] = useState("");
  const [sortBy, setSortBy] = useState<"recent" | "name">("recent");

  useEffect(() => {
    if (open) {
      setStep(1);
      setDraftScenario(initialScenario);
      setDraftTargets(initialTargets);
      setScenarioSearch("");
    }
  }, [open, initialScenario, initialTargets]);

  const sortedScenarios = useMemo(() => {
    const filtered = scenarios.filter((s) =>
      s.scenarioName.toLowerCase().includes(scenarioSearch.trim().toLowerCase())
    );
    if (sortBy === "name") {
      return [...filtered].sort((a, b) => a.scenarioName.localeCompare(b.scenarioName));
    }
    return [...filtered].sort((a, b) => b.lastActivity - a.lastActivity);
  }, [scenarios, scenarioSearch, sortBy]);

  const recent = useMemo(() => sortedScenarios.slice(0, 4), [sortedScenarios]);

  const handleSave = () => {
    if (!draftScenario.trim()) return;
    onSave(draftScenario.trim(), draftTargets);
    onClose();
  };

  const handleSelectScenario = (name: string) => {
    setDraftScenario(name);
  };

  const handleSelectPath = (path: string) => {
    const normalized = normalizePath(path);
    if (!normalized) return;
    if (!draftTargets.includes(normalized)) {
      setDraftTargets((prev) => [...prev, normalized]);
    }
  };

  const handleRemovePath = (path: string) => {
    setDraftTargets((prev) => prev.filter((p) => p !== path));
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
      <div className="flex w-full max-w-5xl flex-col rounded-2xl border border-white/10 bg-slate-950 p-6 shadow-2xl max-h-[90vh] overflow-hidden">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Scope</p>
            <h2 className="mt-1 text-xl font-semibold text-white">Select scenario & targets</h2>
            <p className="mt-2 text-sm text-slate-300">
              Step 1: pick a scenario. Step 2: choose files/folders under api/ or ui/ to focus test generation.
            </p>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-white">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="mt-4 flex items-center gap-3 text-sm text-slate-300">
          <span className={cn("px-3 py-1 rounded-full border", step === 1 ? "border-cyan-400 text-white" : "border-white/10 text-slate-400")}>
            1. Scenario
          </span>
          <ChevronRight className="h-4 w-4 text-slate-500" />
          <span className={cn("px-3 py-1 rounded-full border", step === 2 ? "border-cyan-400 text-white" : "border-white/10 text-slate-400")}>
            2. Targets
          </span>
        </div>

        <div className="mt-5 flex-1 overflow-y-auto pr-1 space-y-6">
          {step === 1 ? (
            <div className="flex flex-wrap items-center gap-3">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
                <input
                  className="w-80 rounded-lg border border-white/10 bg-black/30 py-2 pl-9 pr-3 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
                  placeholder="Search scenarios..."
                  value={scenarioSearch}
                  onChange={(e) => setScenarioSearch(e.target.value)}
                />
              </div>
              <div className="flex items-center gap-2 text-xs text-slate-400">
                Sort:
                <button
                  className={cn(
                    "rounded-full border px-3 py-1",
                    sortBy === "recent" ? "border-cyan-400 text-white" : "border-white/10 text-slate-400"
                  )}
                  onClick={() => setSortBy("recent")}
                >
                  Recent
                </button>
                <button
                  className={cn(
                    "rounded-full border px-3 py-1",
                    sortBy === "name" ? "border-cyan-400 text-white" : "border-white/10 text-slate-400"
                  )}
                  onClick={() => setSortBy("name")}
                >
                  A-Z
                </button>
              </div>
            </div>

            <div className="space-y-3">
              <p className="text-xs uppercase tracking-[0.25em] text-slate-500">Recent</p>
              <div className="grid gap-3 md:grid-cols-2">
                {recent.map((item) => (
                  <button
                    key={item.scenarioName}
                    className={cn(
                      "rounded-xl border p-4 text-left transition",
                      draftScenario === item.scenarioName
                        ? "border-cyan-400 bg-cyan-400/10"
                        : "border-white/10 bg-white/[0.02] hover:border-white/30"
                    )}
                    onClick={() => handleSelectScenario(item.scenarioName)}
                  >
                    <p className="text-sm font-semibold text-white">{item.scenarioName}</p>
                    <p className="mt-1 text-xs text-slate-400">{item.scenarioDescription || "No description"}</p>
                  </button>
                ))}
              </div>
            </div>

            <div className="space-y-3 pb-2">
              <p className="text-xs uppercase tracking-[0.25em] text-slate-500">All scenarios</p>
              <div className="grid gap-3 md:grid-cols-3">
                {sortedScenarios.map((item) => (
                  <button
                    key={item.scenarioName}
                    className={cn(
                      "rounded-xl border p-4 text-left transition",
                      draftScenario === item.scenarioName
                        ? "border-cyan-400 bg-cyan-400/10"
                        : "border-white/10 bg-white/[0.02] hover:border-white/30"
                    )}
                    onClick={() => handleSelectScenario(item.scenarioName)}
                  >
                    <p className="text-sm font-semibold text-white">{item.scenarioName}</p>
                    <p className="mt-1 text-xs text-slate-400">{item.scenarioDescription || "No description"}</p>
                  </button>
                ))}
              </div>
            </div>
          </div>
          ) : (
            <div className="mt-2 pb-2">
              <FileBrowser
                scenarioName={draftScenario}
                selectedPaths={draftTargets}
                onSelectPath={handleSelectPath}
                onRemovePath={handleRemovePath}
              />
            </div>
          )}
        </div>

        <div className="mt-6 flex justify-between">
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <span className="rounded-full border border-white/10 px-3 py-1">
              Scenario: {draftScenario || "Not selected"}
            </span>
            <span className="rounded-full border border-white/10 px-3 py-1">
              Targets: {draftTargets.length}
            </span>
          </div>
          <div className="flex gap-2">
            {step === 2 && (
              <Button variant="outline" onClick={() => setStep(1)}>
                Back
              </Button>
            )}
            {step === 1 && (
              <Button
                onClick={() => setStep(2)}
                disabled={!draftScenario.trim()}
              >
                Next
              </Button>
            )}
            {step === 2 && (
              <Button onClick={handleSave} disabled={!draftScenario.trim()}>
                Save scope
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
