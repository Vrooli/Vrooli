import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent } from "../ui/card";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import { Monitor, Package, Search, Download, Loader2, XCircle } from "lucide-react";
import { ScenarioCard } from "./ScenarioCard";
import { ScenarioDetails } from "./ScenarioDetails";
import { fetchScenarioDesktopStatus } from "../../lib/api";
import type { ScenariosResponse, FilterStatus, ScenarioDesktopStatus } from "./types";

export function ScenarioInventory() {
  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState<FilterStatus>("all");
  const [selectedScenario, setSelectedScenario] = useState<ScenarioDesktopStatus | null>(null);

  const { data, isLoading, error } = useQuery<ScenariosResponse>({
    queryKey: ['scenarios-desktop-status'],
    queryFn: fetchScenarioDesktopStatus,
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  useEffect(() => {
    if (!selectedScenario || !data?.scenarios) return;
    const updatedScenario = data.scenarios.find((scenario) => scenario.name === selectedScenario.name);
    if (updatedScenario && updatedScenario !== selectedScenario) {
      setSelectedScenario(updatedScenario);
    }
  }, [data, selectedScenario]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
        <span className="ml-3 text-slate-400">Loading scenarios...</span>
      </div>
    );
  }

  if (error) {
    return (
      <Card className="border-red-900 bg-red-950/20">
        <CardContent className="p-6">
          <div className="flex items-center gap-3 text-red-400">
            <XCircle className="h-6 w-6" />
            <div>
              <div className="font-semibold">Failed to load scenarios</div>
              <div className="text-sm text-red-300">{(error as Error).message}</div>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  // Filter scenarios based on search and status
  const filteredScenarios = data?.scenarios.filter(scenario => {
    const matchesSearch = scenario.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         (scenario.display_name?.toLowerCase().includes(searchTerm.toLowerCase()));
    const matchesFilter = filterStatus === "all" ||
                         (filterStatus === "desktop" && scenario.has_desktop) ||
                         (filterStatus === "web" && !scenario.has_desktop);
    return matchesSearch && matchesFilter;
  }) || [];

  const activeScenario = selectedScenario && filteredScenarios.some((scenario) => scenario.name === selectedScenario.name)
    ? selectedScenario
    : null;

  return (
    <div className="space-y-6">
      {/* Statistics Cards */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <Card className="border-slate-700 bg-slate-800/50">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <Package className="h-8 w-8 text-blue-400" />
              <div>
                <div className="text-2xl font-bold">{data?.stats.total || 0}</div>
                <div className="text-sm text-slate-400">Total Scenarios</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-green-700 bg-green-950/20">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <Monitor className="h-8 w-8 text-green-400" />
              <div>
                <div className="text-2xl font-bold">{data?.stats.with_desktop || 0}</div>
                <div className="text-sm text-slate-400">With Desktop</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-blue-700 bg-blue-950/20">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <Download className="h-8 w-8 text-blue-400" />
              <div>
                <div className="text-2xl font-bold">{data?.stats.built || 0}</div>
                <div className="text-sm text-slate-400">Built Packages</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-slate-700 bg-slate-800/50">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <Package className="h-8 w-8 text-slate-400" />
              <div>
                <div className="text-2xl font-bold">{data?.stats.web_only || 0}</div>
                <div className="text-sm text-slate-400">Web Only</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search and Filter */}
      <Card className="border-slate-700 bg-slate-800/50">
        <CardContent className="p-4">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <Input
                type="text"
                placeholder="Search scenarios..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-10"
              />
            </div>
            <div className="flex gap-2">
              <Button
                variant={filterStatus === "all" ? "default" : "outline"}
                onClick={() => setFilterStatus("all")}
                size="sm"
              >
                All
              </Button>
              <Button
                variant={filterStatus === "desktop" ? "default" : "outline"}
                onClick={() => setFilterStatus("desktop")}
                size="sm"
              >
                Desktop
              </Button>
              <Button
                variant={filterStatus === "web" ? "default" : "outline"}
                onClick={() => setFilterStatus("web")}
                size="sm"
              >
                Web Only
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Scenario Details */}
      {activeScenario && (
        <div>
          <ScenarioDetails
            scenario={activeScenario}
            onClose={() => setSelectedScenario(null)}
          />
        </div>
      )}

      {/* Scenarios List */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">
            Scenarios ({filteredScenarios.length})
          </h3>
        </div>

        {filteredScenarios.length === 0 ? (
          <Card className="border-slate-700 bg-slate-800/50">
            <CardContent className="p-8 text-center">
              <Package className="mx-auto h-12 w-12 text-slate-600" />
              <p className="mt-3 text-slate-400">No scenarios found matching your filters</p>
            </CardContent>
          </Card>
        ) : (
          filteredScenarios.map((scenario) => (
            <ScenarioCard
              key={scenario.name}
              scenario={scenario}
              onSelect={setSelectedScenario}
              isSelected={activeScenario?.name === scenario.name}
            />
          ))
        )}
      </div>
    </div>
  );
}
