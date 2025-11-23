import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  Loader2,
  AlertCircle,
  Info,
  Sparkles,
} from "lucide-react";
import { analyzeDependencies, DependencyAnalysisResponse } from "../lib/api";
import { cn } from "../lib/utils";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card";
import { Input } from "./ui/input";
import { Label } from "./ui/label";

type GuidedFlowProps = {
  open: boolean;
  onClose: () => void;
};

type TierOption = {
  id: number;
  key: keyof DependencyAnalysisResponse["tiers"];
  label: string;
  description: string;
};

const TIER_OPTIONS: TierOption[] = [
  { id: 1, key: "local", label: "Tier 1 · Local/Dev", description: "Reference stack, best for iteration" },
  { id: 2, key: "desktop", label: "Tier 2 · Desktop", description: "Windows/macOS/Linux bundles" },
  { id: 3, key: "mobile", label: "Tier 3 · Mobile", description: "iOS/Android packages" },
  { id: 4, key: "saas", label: "Tier 4 · SaaS/Cloud", description: "Managed or self-hosted cloud" },
  { id: 5, key: "enterprise", label: "Tier 5 · Appliance", description: "Dedicated hardware deployments" },
];

const getFitnessColor = (score: number) => {
  if (score >= 75) return "text-green-400";
  if (score >= 50) return "text-yellow-400";
  return "text-orange-400";
};

export function GuidedFlow({ open, onClose }: GuidedFlowProps) {
  const [step, setStep] = useState<1 | 2 | 3>(1);
  const [scenario, setScenario] = useState("");
  const [tierKey, setTierKey] = useState<TierOption["key"]>("desktop");
  const [submittedScenario, setSubmittedScenario] = useState<string | null>(null);

  const selectedTier = useMemo(
    () => TIER_OPTIONS.find((tier) => tier.key === tierKey),
    [tierKey],
  );

  const {
    data,
    isFetching,
    isError,
    error,
    refetch,
    isFetched,
  } = useQuery({
    queryKey: ["guided-analyze", submittedScenario],
    queryFn: () => analyzeDependencies(submittedScenario || ""),
    enabled: false,
  });

  useEffect(() => {
    if (open) {
      setStep(1);
    }
  }, [open]);

  const startAnalysis = () => {
    if (!scenario) return;
    setSubmittedScenario(scenario);
    refetch();
    setStep(2);
  };

  const reset = () => {
    setStep(1);
    setScenario("");
    setSubmittedScenario(null);
  };

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm px-4 py-6"
      role="dialog"
      aria-modal="true"
      data-testid="guided-flow-overlay"
    >
      <div className="w-full max-w-5xl overflow-hidden rounded-2xl border border-white/10 bg-slate-950 shadow-2xl">
        <div className="flex items-center justify-between border-b border-white/5 px-6 py-4">
          <div className="flex items-center gap-3">
            <Sparkles className="h-5 w-5 text-cyan-400" />
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-400">Guided deployment</p>
              <h2 className="text-xl font-semibold">Make this scenario deployable</h2>
            </div>
          </div>
          <Button variant="ghost" onClick={() => { onClose(); reset(); }}>
            Close
          </Button>
        </div>

        <div className="grid gap-6 p-6 md:grid-cols-[320px,1fr]">
          {/* Left rail */}
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">3-step flow</CardTitle>
                <CardDescription>Modeled after scenario-to-desktop onboarding</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                {[
                  { id: 1, title: "Pick scenario + tier", description: "Tell us what you’re targeting" },
                  { id: 2, title: "See readiness", description: "Fitness, blockers, swaps, secrets" },
                  { id: 3, title: "Export or hand-off", description: "Bundle download or send to packagers" },
                ].map((item, idx, arr) => {
                  const isActive = step === item.id;
                  const isComplete = step > item.id;
                  return (
                    <div key={item.id} className="relative pl-12">
                      <div
                        className={cn(
                          "absolute left-0 top-1 flex h-7 w-7 items-center justify-center rounded-full border text-xs",
                          isActive
                            ? "border-cyan-400 bg-cyan-500/20 text-cyan-100"
                            : isComplete
                              ? "border-emerald-400 bg-emerald-500/10 text-emerald-100"
                              : "border-white/15 bg-white/5 text-slate-300",
                        )}
                      >
                        {isComplete ? <CheckCircle2 className="h-3.5 w-3.5" /> : item.id}
                      </div>
                      <div
                        className={cn(
                          "absolute left-[13.5px] top-7 bottom-0 w-px",
                          idx === arr.length - 1 ? "hidden" : "bg-white/10",
                        )}
                      />
                      <div
                        className={cn(
                          "rounded-lg border px-3 py-2 text-sm",
                          isActive
                            ? "border-cyan-400 bg-cyan-500/10"
                            : "border-white/5 bg-white/5",
                        )}
                      >
                        <p className="font-medium">Step {item.id}</p>
                        <p className="text-slate-300">{item.title}</p>
                        <p className="text-xs text-slate-500">{item.description}</p>
                      </div>
                    </div>
                  );
                })}
              </CardContent>
            </Card>
          </div>

          {/* Main content */}
          <div className="space-y-6">
            {step === 1 && (
              <Card>
                <CardHeader>
                  <CardTitle>Pick scenario & target tier</CardTitle>
                  <CardDescription>We’ll run analysis and guide you through swaps and secrets</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="scenario-name">Scenario</Label>
                    <Input
                      id="scenario-name"
                      placeholder="e.g., picker-wheel"
                      value={scenario}
                      onChange={(e) => setScenario(e.target.value)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>Target tier</Label>
                    <div className="grid gap-2 sm:grid-cols-2">
                      {TIER_OPTIONS.map((tier) => (
                        <button
                          key={tier.id}
                          type="button"
                          onClick={() => setTierKey(tier.key)}
                          className={`rounded-lg border p-3 text-left transition ${
                            tierKey === tier.key
                              ? "border-cyan-400 bg-cyan-500/10"
                              : "border-white/10 bg-white/5 hover:border-white/20"
                          }`}
                        >
                          <p className="text-sm font-semibold">{tier.label}</p>
                          <p className="text-xs text-slate-400">{tier.description}</p>
                        </button>
                      ))}
                    </div>
                  </div>
                  <div className="flex items-center justify-between">
                    <p className="text-xs text-slate-400 flex items-center gap-1">
                      <Info className="h-4 w-4" />
                      We’ll auto-run analysis and show the next best action.
                    </p>
                    <Button onClick={startAnalysis} disabled={!scenario || isFetching}>
                      {isFetching && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                      Run analysis
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}

            {step === 2 && (
              <Card>
                <CardHeader>
                  <CardTitle>Readiness summary</CardTitle>
                  <CardDescription>Fitness, blockers, and immediate next steps</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  {isFetching && (
                    <div className="flex items-center gap-2 text-slate-300">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <span>Analyzing {submittedScenario}…</span>
                    </div>
                  )}
                  {isError && (
                    <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-100 flex items-center gap-2">
                      <AlertCircle className="h-4 w-4" />
                      {(error as Error).message}
                    </div>
                  )}
                  {data && !isFetching && (
                    <div className="space-y-4">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant="secondary">Scenario: {data.scenario}</Badge>
                        <Badge variant="secondary">Tier: {selectedTier?.label}</Badge>
                        <Badge variant="outline">Dependencies: {Object.keys(data.dependencies || {}).length}</Badge>
                      </div>
                      <div className="rounded-lg border border-white/10 bg-white/5 p-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <p className="text-sm text-slate-400">Overall fitness</p>
                            <p className="text-3xl font-bold">
                              {data.tiers?.[tierKey]?.overall ?? "—"}
                            </p>
                          </div>
                          <div className="text-right text-sm text-slate-400">
                            <p>Portability: <span className={getFitnessColor(data.tiers?.[tierKey]?.portability ?? 0)}>{data.tiers?.[tierKey]?.portability ?? "—"}</span></p>
                            <p>Resources: <span className={getFitnessColor(data.tiers?.[tierKey]?.resources ?? 0)}>{data.tiers?.[tierKey]?.resources ?? "—"}</span></p>
                            <p>Platform: <span className={getFitnessColor(data.tiers?.[tierKey]?.platform_support ?? 0)}>{data.tiers?.[tierKey]?.platform_support ?? "—"}</span></p>
                          </div>
                        </div>
                        <p className="mt-3 text-sm text-slate-300">
                          Next: jump into swaps and secrets to lift the score, then export a bundle.
                        </p>
                      </div>

                      {data.circular_dependencies.length > 0 && (
                        <div className="rounded-lg border border-yellow-500/30 bg-yellow-500/10 p-3 text-sm text-yellow-50">
                          <p className="font-semibold mb-2">Circular dependencies detected</p>
                          <ul className="list-disc pl-5 space-y-1">
                            {data.circular_dependencies.map((item, idx) => (
                              <li key={idx}>{item}</li>
                            ))}
                          </ul>
                        </div>
                      )}

                      <div className="grid gap-3 md:grid-cols-3">
                        <Card className="border-white/10 bg-white/5">
                          <CardHeader>
                            <CardTitle className="text-base">Analyze deeper</CardTitle>
                            <CardDescription>See the full dependency tree</CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <p className="text-sm text-slate-300">
                              Open the analyzer with this scenario prefilled.
                            </p>
                            <Link to={`/analyze?scenario=${encodeURIComponent(data.scenario)}`}>
                              <Button variant="secondary" className="w-full">
                                Open analyzer
                              </Button>
                            </Link>
                          </CardContent>
                        </Card>
                        <Card className="border-white/10 bg-white/5 relative overflow-hidden">
                          <Badge className="absolute right-3 top-3" variant="secondary">Recommended</Badge>
                          <CardHeader>
                            <CardTitle className="text-base">Plan swaps + secrets</CardTitle>
                            <CardDescription>Create a profile to manage decisions</CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <p className="text-sm text-slate-300">
                              Use the profile wizard with your target tier pre-selected.
                            </p>
                            <Link
                              to={`/profiles/new?scenario=${encodeURIComponent(data.scenario)}&tier=${selectedTier?.id ?? 2}`}
                            >
                              <Button className="w-full">
                                Create profile
                              </Button>
                            </Link>
                          </CardContent>
                        </Card>
                        <Card className="border-white/10 bg-white/5">
                          <CardHeader>
                            <CardTitle className="text-base">Export or deploy</CardTitle>
                            <CardDescription>Hand-off to scenario-to-*</CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <p className="text-sm text-slate-300">
                              Final step once swaps/secrets are settled.
                            </p>
                            <div className="flex flex-col gap-2">
                              <Button variant="outline" className="w-full" asChild>
                                <Link to="/deployments">
                                  Go to deployments
                                </Link>
                              </Button>
                              <Button variant="ghost" className="w-full" onClick={() => setStep(3)}>
                                Continue guided flow
                              </Button>
                            </div>
                          </CardContent>
                        </Card>
                      </div>
                    </div>
                  )}
                  <div className="flex items-center justify-between pt-2">
                    <Button variant="outline" onClick={() => { setStep(1); }}>
                      <ArrowLeft className="mr-2 h-4 w-4" />
                      Back
                    </Button>
                    <div className="flex gap-2">
                      <Button variant="ghost" onClick={reset}>Start over</Button>
                      <Button onClick={() => setStep(3)} disabled={!data || isFetching}>
                        Next
                        <ArrowRight className="ml-2 h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}

            {step === 3 && (
              <Card>
                <CardHeader>
                  <CardTitle>Export & hand-off</CardTitle>
                  <CardDescription>Keep users oriented with clear CTAs</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="rounded-lg border border-white/10 bg-white/5 p-4">
                    <p className="text-sm text-slate-300">
                      Use these entry points to finish: bundle export, send to scenario-to-desktop/mobile/cloud, or jump to deployments to monitor runs.
                    </p>
                  </div>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <Link to="/deployments">
                      <Button className="w-full">Open deployments hub</Button>
                    </Link>
                    <Link to="/profiles">
                      <Button variant="secondary" className="w-full">Review profiles</Button>
                    </Link>
                    <Link to={`/analyze${scenario ? `?scenario=${encodeURIComponent(scenario)}` : ""}`}>
                      <Button variant="outline" className="w-full">Re-run analysis</Button>
                    </Link>
                    <Button variant="ghost" className="w-full" onClick={() => { onClose(); reset(); }}>
                      Finish guided flow
                    </Button>
                  </div>
                  <div className="text-xs text-slate-500">
                    This flow mirrors the scenario-to-desktop 3-part onboarding so new users can follow the same mental model: choose a target, fix blockers, export/deploy.
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
