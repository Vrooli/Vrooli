import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { Search, Loader2, AlertCircle, CheckCircle2, AlertTriangle, HelpCircle, Maximize2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Badge } from "../components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Tip } from "../components/ui/tip";
import { analyzeDependencies, listProfiles } from "../lib/api";

const TIER_NAMES: Record<string, string> = {
  local: "Local/Dev",
  desktop: "Desktop",
  mobile: "Mobile",
  saas: "SaaS/Cloud",
  enterprise: "Enterprise",
};

const TIER_KEY_BY_ID: Record<string, string> = {
  "1": "local",
  "2": "desktop",
  "3": "mobile",
  "4": "saas",
  "5": "enterprise",
};

function getFitnessColor(score: number): string {
  if (score >= 75) return "text-green-400";
  if (score >= 50) return "text-yellow-400";
  if (score > 0) return "text-orange-400";
  return "text-red-400";
}

function getFitnessIcon(score: number) {
  if (score >= 75) return CheckCircle2;
  if (score >= 50) return AlertTriangle;
  return AlertCircle;
}

export function Analyze() {
  const [showHelp, setShowHelp] = useState(false);
  const [searchParams] = useSearchParams();
  const [scenario, setScenario] = useState("");
  const [queryScenario, setQueryScenario] = useState("");
  const [focusedTier, setFocusedTier] = useState<string | null>(null);
  const { data: profiles } = useQuery({
    queryKey: ["profiles"],
    queryFn: listProfiles,
  });

  const { data, isLoading, error } = useQuery({
    queryKey: ["analyze", queryScenario],
    queryFn: () => analyzeDependencies(queryScenario),
    enabled: !!queryScenario,
  });

  const [analyzerTarget, setAnalyzerTarget] = useState<string | null>(null);
  const [fullscreen, setFullscreen] = useState(false);

  useEffect(() => {
    fetch("/embedded/analyzer/target")
      .then((res) => res.json())
      .then((json) => {
        if (json?.url) setAnalyzerTarget(json.url as string);
      })
      .catch(() => {});
  }, []);

  const handleAnalyze = (e: React.FormEvent) => {
    e.preventDefault();
    setQueryScenario(scenario);
  };

  useEffect(() => {
    const fromQuery = searchParams.get("scenario");
    const tierQuery = searchParams.get("tier");
    if (fromQuery) {
      setScenario(fromQuery);
      setQueryScenario(fromQuery);
    }
    if (tierQuery) {
      if (TIER_KEY_BY_ID[tierQuery]) {
        setFocusedTier(TIER_KEY_BY_ID[tierQuery]);
      } else if (TIER_NAMES[tierQuery]) {
        setFocusedTier(tierQuery);
      }
    }
  }, [searchParams]);

  const scenarioOptions = useMemo(() => {
    const existing = Array.from(
      new Set((profiles ?? []).map((p) => p.scenario).filter(Boolean))
    );
    const defaults = ["picker-wheel", "system-monitor", "browser-automation-studio"];
    const combined = Array.from(new Set([...existing, ...defaults]));
    const typed = scenario.trim().toLowerCase();
    const filtered = combined.filter((name) =>
      typed ? name.toLowerCase().includes(typed) : true
    );
    if (typed && !combined.some((name) => name.toLowerCase() === typed)) {
      filtered.unshift(scenario);
    }
    return filtered;
  }, [profiles, scenario]);

  const selectScenario = (name: string) => {
    setScenario(name);
    setQueryScenario(name);
  };

  const iframeUrl = useMemo(() => {
    const params = new URLSearchParams();
    if (queryScenario) params.set("scenario", queryScenario);
    params.set("graph_type", "combined");
    params.set("layout", "force");
    const base = analyzerTarget || "/embedded/analyzer/";
    const separator = base.endsWith("/") ? "" : "/";
    return `${base}${separator}?${params.toString()}`;
  }, [analyzerTarget, queryScenario]);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-start justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-3xl font-bold">Dependency Analysis</h1>
          <p className="text-slate-400 mt-1">
            Pick a scenario to see dependencies, fitness, and blockers for each tier.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => setShowHelp((v) => !v)} className="gap-2">
          <HelpCircle className="h-4 w-4" />
          {showHelp ? "Hide help" : "How this works"}
        </Button>
      </div>

      {showHelp && (
        <Tip title="How this works">
          <p>Select a scenario card below (pre-filled from profiles + examples) or type to filter.</p>
          <p>We’ll auto-run analysis and show readiness by tier. From here, jump to swaps/secrets or export.</p>
        </Tip>
      )}

      {/* Search & choices */}
      <Card>
        <CardHeader>
          <CardTitle>Choose a scenario</CardTitle>
          <CardDescription>
            Start from an existing profile or suggested scenario; typing filters cards.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={handleAnalyze} className="flex gap-3">
            <div className="flex-1">
              <Label htmlFor="scenario" className="sr-only">Scenario Name</Label>
              <Input
                id="scenario"
                placeholder="Search or type a scenario name"
                value={scenario}
                onChange={(e) => setScenario(e.target.value)}
              />
            </div>
            <Button type="submit" disabled={!scenario || isLoading}>
              {isLoading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Search className="mr-2 h-4 w-4" />
              )}
              Analyze
            </Button>
          </form>

          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {scenarioOptions.map((name) => (
              <button
                key={name}
                type="button"
                onClick={() => selectScenario(name)}
                className="rounded-lg border border-white/10 bg-white/5 p-3 text-left transition hover:border-cyan-400 hover:bg-cyan-500/10"
              >
                <div className="flex items-center justify-between">
                  <p className="font-semibold">{name}</p>
                  {queryScenario === name && (
                    <Badge variant="secondary">Selected</Badge>
                  )}
                </div>
                <p className="text-xs text-slate-400 mt-1">
                  Click to run analysis and view fitness per tier.
                </p>
              </button>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Error */}
      {error && (
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-200">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5" />
            <p className="text-sm">Failed to analyze: {(error as Error).message}</p>
          </div>
        </div>
      )}

      {/* Results */}
      {data && (
        <div className="space-y-6">
          <div className="flex flex-wrap gap-2">
            <Badge variant="secondary">Scenario: {data.scenario}</Badge>
            <Badge variant="outline">Dependencies: {Object.keys(data.dependencies || {}).length}</Badge>
            <Badge variant="outline">Tiers scored: {Object.keys(data.tiers || {}).length}</Badge>
          </div>

          {/* Circular Dependencies Warning */}
          {data.circular_dependencies.length > 0 && (
            <div className="rounded-lg border border-yellow-500/20 bg-yellow-500/10 p-4 text-yellow-200">
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-5 w-5 mt-0.5" />
                <div>
                  <p className="font-medium">Circular Dependencies Detected</p>
                  <ul className="text-sm mt-2 space-y-1">
                    {data.circular_dependencies.map((dep, i) => (
                      <li key={i}>{dep}</li>
                    ))}
                  </ul>
                </div>
              </div>
            </div>
          )}

          {/* Fitness Scores */}
          <Card>
            <CardHeader>
              <CardTitle>Deployment Fitness Scores</CardTitle>
              <CardDescription>
                Overall fitness for each deployment tier (0-100 scale)
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {Object.entries(data.tiers).map(([tierKey, scores]) => {
                  const Icon = getFitnessIcon(scores.overall);
                  return (
                    <div
                      key={tierKey}
                      className={`rounded-lg border p-4 transition ${
                        focusedTier === tierKey
                          ? "border-cyan-400 bg-cyan-500/10 shadow-lg shadow-cyan-500/10"
                          : "border-white/10 bg-white/5"
                      }`}
                    >
                      <div className="flex items-center justify-between mb-3">
                        <h3 className="font-medium flex items-center gap-2">
                          {TIER_NAMES[tierKey] || tierKey}
                          {focusedTier === tierKey && (
                            <Badge variant="secondary">From profile</Badge>
                          )}
                        </h3>
                        <Icon className={`h-5 w-5 ${getFitnessColor(scores.overall)}`} />
                      </div>
                      <div className={`text-3xl font-bold ${getFitnessColor(scores.overall)}`}>
                        {scores.overall}
                      </div>
                      <div className="mt-3 space-y-1 text-xs">
                        <div className="flex justify-between">
                          <span className="text-slate-400">Portability</span>
                          <span className={getFitnessColor(scores.portability)}>
                            {scores.portability}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-slate-400">Resources</span>
                          <span className={getFitnessColor(scores.resources)}>
                            {scores.resources}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-slate-400">Licensing</span>
                          <span className={getFitnessColor(scores.licensing)}>
                            {scores.licensing}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-slate-400">Platform Support</span>
                          <span className={getFitnessColor(scores.platform_support)}>
                            {scores.platform_support}
                          </span>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </CardContent>
          </Card>

          {/* Resource Requirements */}
          <Card>
            <CardHeader>
              <CardTitle>Aggregate Resource Requirements</CardTitle>
              <CardDescription>
                Total resources needed to run this scenario
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3">
                <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                  <div className="text-sm text-slate-400">Memory</div>
                  <div className="text-lg font-semibold mt-1">
                    {data.aggregate_requirements.memory}
                  </div>
                </div>
                <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                  <div className="text-sm text-slate-400">CPU</div>
                  <div className="text-lg font-semibold mt-1">
                    {data.aggregate_requirements.cpu}
                  </div>
                </div>
                <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                  <div className="text-sm text-slate-400">GPU</div>
                  <div className="text-lg font-semibold mt-1">
                    {data.aggregate_requirements.gpu}
                  </div>
                </div>
                <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                  <div className="text-sm text-slate-400">Storage</div>
                  <div className="text-lg font-semibold mt-1">
                    {data.aggregate_requirements.storage}
                  </div>
                </div>
                <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                  <div className="text-sm text-slate-400">Network</div>
                  <div className="text-lg font-semibold mt-1">
                    {data.aggregate_requirements.network}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Dependencies Details */}
          <Card>
            <CardHeader>
              <CardTitle>Dependency Tree</CardTitle>
              <CardDescription>
                All resources and scenarios required by {queryScenario}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Tabs defaultValue="overview">
                <TabsList>
                  <TabsTrigger value="overview">Overview</TabsTrigger>
                  <TabsTrigger value="raw">Raw Data</TabsTrigger>
                </TabsList>
                <TabsContent value="overview" className="mt-4">
                  <div className={`rounded-lg border border-white/10 bg-black/30 p-2 ${fullscreen ? "fixed inset-4 z-50 bg-slate-950" : ""}`}>
                    <div className="flex items-center justify-between pb-2">
                      <p className="text-xs text-slate-400">
                        Visualization powered by scenario-dependency-analyzer. Use the search above to update the view.
                      </p>
                      <button
                        type="button"
                        onClick={() => setFullscreen((v) => !v)}
                        className="inline-flex items-center gap-2 rounded border border-white/10 px-2 py-1 text-xs text-slate-200 hover:border-cyan-400 hover:text-cyan-100"
                      >
                        <Maximize2 className="h-3.5 w-3.5" />
                        {fullscreen ? "Exit full screen" : "Full screen"}
                      </button>
                    </div>
                    <iframe
                      src={iframeUrl}
                      title="Scenario Dependency Analyzer"
                      className={`w-full rounded-md border border-white/10 bg-slate-950 ${fullscreen ? "h-[80vh]" : "h-[560px]"}`}
                      allowFullScreen
                    />
                  </div>
                </TabsContent>
                <TabsContent value="raw" className="mt-4">
                  <pre className="rounded-lg bg-black/40 p-4 text-xs overflow-auto max-h-96">
                    {JSON.stringify(data.dependencies, null, 2)}
                  </pre>
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
