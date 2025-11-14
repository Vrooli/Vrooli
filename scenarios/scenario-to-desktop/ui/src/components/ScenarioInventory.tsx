import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Badge } from "./ui/badge";
import { Input } from "./ui/input";
import { Button } from "./ui/button";
import { Monitor, Package, Search, Download, CheckCircle, XCircle, Loader2, Zap } from "lucide-react";

interface ScenarioDesktopStatus {
  name: string;
  display_name?: string;
  has_desktop: boolean;
  desktop_path?: string;
  version?: string;
  platforms?: string[];
  built?: boolean;
  dist_path?: string;
  last_modified?: string;
  package_size?: number;
}

interface ScenariosResponse {
  scenarios: ScenarioDesktopStatus[];
  stats: {
    total: number;
    with_desktop: number;
    built: number;
    web_only: number;
  };
}

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

interface GenerateDesktopButtonProps {
  scenarioName: string;
}

function GenerateDesktopButton({ scenarioName }: GenerateDesktopButtonProps) {
  const queryClient = useQueryClient();
  const [buildId, setBuildId] = useState<string | null>(null);

  const generateMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(buildUrl('/desktop/generate/quick'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scenario_name: scenarioName,
          template_type: 'basic'
        })
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to generate desktop app');
      }
      return res.json();
    },
    onSuccess: (data) => {
      setBuildId(data.build_id);
      // Refresh scenarios list to show the new desktop version
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
      }, 3000);
    }
  });

  // Poll build status if we have a buildId
  const { data: buildStatus } = useQuery({
    queryKey: ['build-status', buildId],
    queryFn: async () => {
      if (!buildId) return null;
      const res = await fetch(buildUrl(`/desktop/status/${buildId}`));
      if (!res.ok) throw new Error('Failed to fetch build status');
      return res.json();
    },
    enabled: !!buildId,
    refetchInterval: (data) => {
      // Stop polling if build is complete or failed
      if (data?.status === 'ready' || data?.status === 'failed') {
        return false;
      }
      return 2000; // Poll every 2 seconds while building
    }
  });

  const isBuilding = generateMutation.isPending || buildStatus?.status === 'building';
  const isComplete = buildStatus?.status === 'ready';
  const isFailed = buildStatus?.status === 'failed' || generateMutation.isError;

  if (isComplete) {
    return (
      <div className="ml-4 flex flex-col items-end gap-2">
        <Badge variant="success" className="gap-1">
          <CheckCircle className="h-3 w-3" />
          Generated!
        </Badge>
        <p className="text-xs text-slate-400">
          {buildStatus.output_path}
        </p>
      </div>
    );
  }

  if (isFailed) {
    return (
      <div className="ml-4 flex flex-col items-end gap-2">
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Failed
        </Badge>
        <p className="text-xs text-red-400">
          {buildStatus?.error_log?.[0] || generateMutation.error?.message || 'Unknown error'}
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setBuildId(null);
            generateMutation.reset();
          }}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (isBuilding) {
    return (
      <div className="ml-4 flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-purple-400" />
        <span className="text-sm text-slate-400">Generating...</span>
      </div>
    );
  }

  return (
    <Button
      variant="outline"
      size="sm"
      className="ml-4 gap-2"
      onClick={() => generateMutation.mutate()}
      disabled={isBuilding}
    >
      <Zap className="h-4 w-4" />
      Generate Desktop
    </Button>
  );
}

export function ScenarioInventory() {
  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState<"all" | "desktop" | "web">("all");

  const { data, isLoading, error } = useQuery<ScenariosResponse>({
    queryKey: ['scenarios-desktop-status'],
    queryFn: async () => {
      const res = await fetch(buildUrl('/scenarios/desktop-status'));
      if (!res.ok) throw new Error('Failed to fetch scenarios');
      return res.json();
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-8 w-8 animate-spin text-purple-400" />
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

        <Card className="border-purple-700 bg-purple-950/20">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <Download className="h-8 w-8 text-purple-400" />
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
            <Card
              key={scenario.name}
              className={`border ${
                scenario.has_desktop
                  ? 'border-green-700 bg-green-950/10'
                  : 'border-slate-700 bg-slate-800/50'
              } transition-all hover:border-purple-600`}
            >
              <CardContent className="p-4">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-700">
                        {scenario.has_desktop ? (
                          <Monitor className="h-5 w-5 text-green-400" />
                        ) : (
                          <Package className="h-5 w-5 text-slate-400" />
                        )}
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h4 className="font-semibold">{scenario.name}</h4>
                          {scenario.has_desktop && (
                            <Badge variant="success" className="text-xs">
                              <CheckCircle className="mr-1 h-3 w-3" />
                              Desktop
                            </Badge>
                          )}
                          {scenario.built && (
                            <Badge variant="default" className="text-xs">
                              <Download className="mr-1 h-3 w-3" />
                              Built
                            </Badge>
                          )}
                        </div>
                        {scenario.display_name && (
                          <p className="text-sm text-slate-400">{scenario.display_name}</p>
                        )}
                      </div>
                    </div>

                    {scenario.has_desktop && (
                      <div className="mt-3 ml-13 grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
                        {scenario.version && (
                          <div>
                            <span className="text-slate-500">Version:</span>{" "}
                            <span className="text-slate-300">{scenario.version}</span>
                          </div>
                        )}
                        {scenario.platforms && scenario.platforms.length > 0 && (
                          <div>
                            <span className="text-slate-500">Platforms:</span>{" "}
                            <span className="text-slate-300">{scenario.platforms.join(", ")}</span>
                          </div>
                        )}
                        {scenario.package_size && scenario.package_size > 0 && (
                          <div>
                            <span className="text-slate-500">Size:</span>{" "}
                            <span className="text-slate-300">{formatBytes(scenario.package_size)}</span>
                          </div>
                        )}
                        {scenario.last_modified && (
                          <div>
                            <span className="text-slate-500">Modified:</span>{" "}
                            <span className="text-slate-300">{scenario.last_modified}</span>
                          </div>
                        )}
                      </div>
                    )}

                    {scenario.desktop_path && (
                      <div className="mt-2 ml-13 text-xs text-slate-500">
                        {scenario.desktop_path}
                      </div>
                    )}
                  </div>

                  {!scenario.has_desktop && (
                    <GenerateDesktopButton scenarioName={scenario.name} />
                  )}
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
}
