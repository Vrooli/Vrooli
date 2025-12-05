import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams, Link } from "react-router-dom";
import { Search, Loader2, AlertCircle, CheckCircle2, AlertTriangle, HelpCircle, Maximize2, Rocket, Package } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Badge } from "../components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Tip } from "../components/ui/tip";
import { InfoCard } from "../components/ui/info-card";
import { analyzeDependencies, listProfiles } from "../lib/api";
import { TIER_NAMES, TIER_KEY_BY_ID, getFitnessColor, buildScenarioOptions, isTierKey } from "../lib/tiers";

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
      // Support both numeric tier IDs (1-5) and tier keys (local, desktop, etc.)
      const tierKey = TIER_KEY_BY_ID[tierQuery];
      if (tierKey) {
        setFocusedTier(tierKey);
      } else if (isTierKey(tierQuery)) {
        setFocusedTier(tierQuery);
      }
    }
  }, [searchParams]);

  const scenarioOptions = useMemo(
    () => buildScenarioOptions(profiles, scenario),
    [profiles, scenario],
  );

  const selectScenario = (name: string) => {
    setScenario(name);
    setQueryScenario(name);
  };

  const matchingProfile = useMemo(
    () => (profiles ?? []).find((p) => p.scenario === data?.scenario) ?? null,
    [profiles, data?.scenario],
  );

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

          <Card>
            <CardHeader>
              <CardTitle>Next steps</CardTitle>
              <CardDescription>
                Lift fitness, prep a bundle, then hand off to a scenario-to-* packager (deploy is stubbed today).
              </CardDescription>
            </CardHeader>
            <CardContent className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
              <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                <div className="flex items-center gap-2 text-sm font-semibold">
                  <Package className="h-4 w-4 text-cyan-300" />
                  Profiles & swaps
                </div>
                <p className="text-xs text-slate-400 mt-1">
                  Apply swaps and secrets in a profile to raise the score, then re-run analysis.
                </p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {matchingProfile ? (
                    <Link to={`/profiles/${matchingProfile.id}`}>
                      <Button size="sm" variant="secondary">Open profile</Button>
                    </Link>
                  ) : (
                    <Link to="/profiles/new">
                      <Button size="sm" variant="secondary">Create profile</Button>
                    </Link>
                  )}
                  <Link to="/profiles">
                    <Button size="sm" variant="outline">All profiles</Button>
                  </Link>
                </div>
              </div>

              <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                <div className="flex items-center gap-2 text-sm font-semibold">
                  <Rocket className="h-4 w-4 text-amber-300" />
                  Bundle & packager
                </div>
                <p className="text-xs text-slate-400 mt-1">
                  Export a bundle for your target tier and run the scenario-to-* packager (desktop/mobile/saas).
                </p>
                <div className="mt-2 flex flex-wrap gap-2">
                  <Link to="/deployments">
                    <Button size="sm">Open deployments</Button>
                  </Link>
                  <Button size="sm" variant="ghost" disabled title="Packager hand-off is stubbed here; use scenario-to-* manually.">
                    Send to packager
                  </Button>
                </div>
              </div>

              <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                <div className="flex items-center gap-2 text-sm font-semibold">
                  <AlertTriangle className="h-4 w-4 text-yellow-300" />
                  Validate & monitor
                </div>
                <p className="text-xs text-slate-400 mt-1">
                  Keep fitness ≥ target, capture secrets, and plan telemetry upload after the packaged app runs.
                </p>
                <div className="mt-2 flex flex-wrap gap-2">
                  <Link to="/telemetry">
                    <Button size="sm" variant="outline">Telemetry</Button>
                  </Link>
                  <Button size="sm" variant="ghost" onClick={() => setShowHelp((v) => !v)}>
                    How scoring works
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

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
                <InfoCard label="Memory" value={data.aggregate_requirements.memory} />
                <InfoCard label="CPU" value={data.aggregate_requirements.cpu} />
                <InfoCard label="GPU" value={data.aggregate_requirements.gpu} />
                <InfoCard label="Storage" value={data.aggregate_requirements.storage} />
                <InfoCard label="Network" value={data.aggregate_requirements.network} />
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
