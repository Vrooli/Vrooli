import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";
import { Button } from "./components/ui/button";
import {
  buildBundle,
  buildPlan,
  fetchHealth,
  validateManifest,
  type BundleArtifact,
  type PlanStep,
  type ValidationIssue
} from "./lib/api";
import { selectors } from "./consts/selectors";

const DEFAULT_MANIFEST = `{
  "version": "1.0.0",
  "target": { "type": "vps", "vps": { "host": "203.0.113.10" } },
  "scenario": { "id": "landing-page-business-suite" },
  "dependencies": { "scenarios": ["landing-page-business-suite"], "resources": [] },
  "bundle": { "include_packages": true, "include_autoheal": true },
  "ports": { "ui": 3000, "api": 3001, "ws": 3002 },
  "edge": { "domain": "example.com", "caddy": { "enabled": true } }
}`;

export default function App() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth
  });

  const [manifestText, setManifestText] = useState(DEFAULT_MANIFEST);
  const [validationIssues, setValidationIssues] = useState<ValidationIssue[] | null>(null);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [isValidating, setIsValidating] = useState(false);

  const [plan, setPlan] = useState<PlanStep[] | null>(null);
  const [planError, setPlanError] = useState<string | null>(null);
  const [isPlanning, setIsPlanning] = useState(false);

  const [bundleArtifact, setBundleArtifact] = useState<BundleArtifact | null>(null);
  const [bundleError, setBundleError] = useState<string | null>(null);
  const [isBuildingBundle, setIsBuildingBundle] = useState(false);

  const parsedManifest = useMemo(() => {
    try {
      return { ok: true as const, value: JSON.parse(manifestText) as unknown };
    } catch (e) {
      return { ok: false as const, error: e instanceof Error ? e.message : String(e) };
    }
  }, [manifestText]);

  const validate = async () => {
    setPlan(null);
    setPlanError(null);
    setBundleArtifact(null);
    setBundleError(null);
    setValidationError(null);
    setIsValidating(true);
    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid JSON: ${parsedManifest.error}`);
      }
      const res = await validateManifest(parsedManifest.value);
      setValidationIssues(res.issues ?? []);
    } catch (e) {
      setValidationIssues(null);
      setValidationError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsValidating(false);
    }
  };

  const planDeploy = async () => {
    setPlan(null);
    setPlanError(null);
    setBundleArtifact(null);
    setBundleError(null);
    setIsPlanning(true);
    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid JSON: ${parsedManifest.error}`);
      }
      const res = await buildPlan(parsedManifest.value);
      setPlan(res.plan ?? []);
    } catch (e) {
      setPlanError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsPlanning(false);
    }
  };

  const buildMiniBundle = async () => {
    setBundleArtifact(null);
    setBundleError(null);
    setIsBuildingBundle(true);
    try {
      if (!parsedManifest.ok) {
        throw new Error(`Invalid JSON: ${parsedManifest.error}`);
      }
      const res = await buildBundle(parsedManifest.value);
      setBundleArtifact(res.artifact);
    } catch (e) {
      setBundleError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsBuildingBundle(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 flex flex-col items-center justify-center p-6">
      <div className="w-full max-w-3xl rounded-2xl border border-white/10 bg-white/5 p-8 shadow-2xl backdrop-blur">
        <p className="text-sm uppercase tracking-[0.2em] text-slate-400">Operator Console</p>
        <h1 className="mt-3 text-3xl font-semibold">Scenario To Cloud</h1>
        <p className="mt-2 text-slate-300">
          Validate and plan a deployment-manager exported cloud manifest. P0 focuses on Ubuntu VPS.
        </p>

        <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className="rounded-xl border border-white/10 bg-black/20 p-4">
          <p className="text-sm font-medium text-slate-400">API Health</p>
          {isLoading && <p className="mt-2 text-slate-200">Checking API status…</p>}
          {error && (
            <p className="mt-2 text-red-400">
              Unable to reach the API. Make sure the scenario is running through `vrooli scenario start`.
            </p>
          )}
          {data && (
            <div className="mt-2 text-sm text-slate-200">
              <p>Status: {data.status}</p>
              <p>Service: {data.service}</p>
              <p>Timestamp: {new Date(data.timestamp).toLocaleString()}</p>
            </div>
          )}
          <Button className="mt-4" onClick={() => refetch()}>
            Refresh
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>

          <div className="rounded-xl border border-white/10 bg-black/20 p-4">
            <p className="text-sm font-medium text-slate-400">Cloud Manifest</p>
            <textarea
              data-testid={selectors.manifest.input}
              className="mt-2 h-56 w-full rounded-lg border border-white/10 bg-black/30 p-3 font-mono text-xs text-slate-100 focus:outline-none focus:ring-2 focus:ring-slate-500"
              value={manifestText}
              onChange={(e) => setManifestText(e.target.value)}
              spellCheck={false}
            />

            <div className="mt-3 flex flex-wrap gap-2">
              <Button
                data-testid={selectors.manifest.validateButton}
                disabled={isValidating}
                onClick={validate}
              >
                {isValidating ? "Validating…" : "Validate"}
              </Button>
              <Button
                data-testid={selectors.manifest.planButton}
                disabled={isPlanning}
                onClick={planDeploy}
                variant="outline"
              >
                {isPlanning ? "Planning…" : "Plan"}
              </Button>
              <Button
                data-testid={selectors.manifest.bundleBuildButton}
                disabled={isBuildingBundle}
                onClick={buildMiniBundle}
                variant="outline"
              >
                {isBuildingBundle ? "Building bundle…" : "Build bundle"}
              </Button>
            </div>

            {validationError && <p className="mt-3 text-sm text-red-400">{validationError}</p>}
            {validationIssues && (
              <div data-testid={selectors.manifest.validateResult} className="mt-3 text-sm">
                {validationIssues.length === 0 ? (
                  <p className="text-emerald-300">Valid manifest (no issues).</p>
                ) : (
                  <ul className="space-y-2">
                    {validationIssues.map((issue) => (
                      <li key={`${issue.severity}:${issue.path}:${issue.message}`}>
                        <p className={issue.severity === "error" ? "text-red-300" : "text-amber-200"}>
                          {issue.severity.toUpperCase()} {issue.path}: {issue.message}
                        </p>
                        {issue.hint && <p className="text-slate-300">{issue.hint}</p>}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}

            {planError && <p className="mt-3 text-sm text-red-400">{planError}</p>}
            {plan && (
              <div data-testid={selectors.manifest.planResult} className="mt-3 text-sm">
                <p className="text-slate-200 font-medium">Plan steps</p>
                <ol className="mt-2 space-y-2 list-decimal list-inside">
                  {plan.map((step) => (
                    <li key={step.id}>
                      <span className="text-slate-100">{step.title}</span>
                      <p className="text-slate-300">{step.description}</p>
                    </li>
                  ))}
                </ol>
              </div>
            )}

            {bundleError && <p className="mt-3 text-sm text-red-400">{bundleError}</p>}
            {bundleArtifact && (
              <div data-testid={selectors.manifest.bundleBuildResult} className="mt-3 text-sm">
                <p className="text-slate-200 font-medium">Bundle artifact</p>
                <p className="mt-1 text-slate-100 font-mono text-xs break-all">{bundleArtifact.path}</p>
                <p className="mt-1 text-slate-300 text-xs">sha256: {bundleArtifact.sha256}</p>
                <p className="mt-1 text-slate-300 text-xs">size_bytes: {bundleArtifact.size_bytes}</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
