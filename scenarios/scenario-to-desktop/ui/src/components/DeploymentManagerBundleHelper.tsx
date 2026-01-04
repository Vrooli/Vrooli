import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from "react";
import type { AutoBuildStatusResponse } from "../lib/api";
import { exportBundleFromDeploymentManager, fetchDeploymentManagerAutoBuildStatus, startDeploymentManagerAutoBuild } from "../lib/api";
import { Button } from "./ui/button";
import { Label } from "./ui/label";
import { Progress } from "./ui/progress";
import { Select } from "./ui/select";
import { AlertTriangle, CheckCircle, Loader2 } from "lucide-react";

export type DeploymentManagerBundleHelperHandle = {
  exportBundle: () => void;
};

type DeploymentManagerBundleHelperProps = {
  scenarioName: string;
  onBundleManifestChange: (value: string) => void;
  onBundleExported?: (manifestPath: string) => void;
  onDeploymentManagerUrlChange?: (url: string | null) => void;
};

export const DeploymentManagerBundleHelper = forwardRef<DeploymentManagerBundleHelperHandle, DeploymentManagerBundleHelperProps>(
  ({ scenarioName, onBundleManifestChange, onBundleExported, onDeploymentManagerUrlChange }, ref) => {
    const [tier, setTier] = useState("tier-2-desktop");
    const [generationPhase, setGenerationPhase] = useState<"idle" | "building" | "exporting">("idle");
    const [exportError, setExportError] = useState<string | null>(null);
    const [buildError, setBuildError] = useState<string | null>(null);
    const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
    const [downloadName, setDownloadName] = useState<string | null>(null);
    const [exportMeta, setExportMeta] = useState<{ checksum?: string; generated_at?: string } | null>(null);
    const [deploymentManagerUrl, setDeploymentManagerUrl] = useState<string | null>(null);
    const [manifestPath, setManifestPath] = useState<string | null>(null);
    const [buildJobId, setBuildJobId] = useState<string | null>(null);
    const [buildStatus, setBuildStatus] = useState<AutoBuildStatusResponse | null>(null);
    const exportTriggeredRef = useRef(false);

    useImperativeHandle(ref, () => ({
      exportBundle: () => {
        handleExport();
      }
    }));

    useEffect(() => {
      setGenerationPhase("idle");
      setExportError(null);
      setBuildError(null);
      setBuildJobId(null);
      setBuildStatus(null);
      exportTriggeredRef.current = false;
    }, [scenarioName]);

    const resetGenerationState = () => {
      setExportError(null);
      setBuildError(null);
      setExportMeta(null);
      setDownloadUrl(null);
      setDownloadName(null);
      setManifestPath(null);
      setDeploymentManagerUrl(null);
      setBuildJobId(null);
      setBuildStatus(null);
      exportTriggeredRef.current = false;
      if (onDeploymentManagerUrlChange) {
        onDeploymentManagerUrlChange(null);
      }
    };

    const runExport = useCallback(async () => {
      setGenerationPhase("exporting");
      try {
        const response = await exportBundleFromDeploymentManager({
          scenario: scenarioName,
          tier
        });
        const blob = new Blob([JSON.stringify(response.manifest, null, 2)], { type: "application/json" });
        const url = URL.createObjectURL(blob);
        const filename = `${scenarioName}-${tier}-bundle.json`;
        setDownloadUrl(url);
        setDownloadName(filename);
        setExportMeta({ checksum: response.checksum, generated_at: response.generated_at });
        setDeploymentManagerUrl(response.deployment_manager_url ?? null);
        if (onDeploymentManagerUrlChange) {
          onDeploymentManagerUrlChange(response.deployment_manager_url ?? null);
        }
        if (response.manifest_path) {
          setManifestPath(response.manifest_path);
          onBundleManifestChange(response.manifest_path);
          onBundleExported?.(response.manifest_path);
        }
      } catch (error) {
        setExportError((error as Error).message);
      } finally {
        setGenerationPhase("idle");
      }
    }, [scenarioName, tier, onBundleManifestChange, onBundleExported, onDeploymentManagerUrlChange]);

    const handleExport = async () => {
      resetGenerationState();
      if (!scenarioName.trim()) {
        setExportError("Enter a scenario name above before exporting the bundle.");
        return;
      }
      setGenerationPhase("building");
      try {
        const response = await startDeploymentManagerAutoBuild({
          scenario: scenarioName
        });
        setBuildStatus(response);
        if (response.build_id) {
          setBuildJobId(response.build_id);
        }
        if (response.status === "skipped") {
          exportTriggeredRef.current = true;
          await runExport();
        }
      } catch (error) {
        setBuildError((error as Error).message);
        setGenerationPhase("idle");
      }
    };

    const buildProgress = useMemo(() => {
      const targets = buildStatus?.targets ?? [];
      let total = 0;
      let completed = 0;
      for (const target of targets) {
        for (const platform of target.platforms || []) {
          total += 1;
          if (["success", "failed", "skipped"].includes(platform.status)) {
            completed += 1;
          }
        }
      }
      if (total == 0) {
        return 0;
      }
      return Math.round((completed / total) * 100);
    }, [buildStatus]);

    const buildTargets = buildStatus?.targets ?? [];
    const buildLog = buildStatus?.build_log ?? [];
    const buildErrors = buildStatus?.error_log ?? [];

    useEffect(() => {
      if (!buildJobId) {
        return;
      }
      let active = true;
      let timeoutId: number | undefined;

      const poll = async () => {
        try {
          const status = await fetchDeploymentManagerAutoBuildStatus(buildJobId);
          if (!active) return;
          setBuildStatus(status);
          const doneStatuses = new Set(["success", "partial", "failed", "skipped", "dry_run"]);
          if (doneStatuses.has(status.status)) {
            if ((status.status === "success" || status.status === "skipped") && !exportTriggeredRef.current) {
              exportTriggeredRef.current = true;
              await runExport();
            } else if (status.status !== "success" && status.status !== "skipped") {
              setGenerationPhase("idle");
              setBuildError(`Auto build finished with status "${status.status}".`);
            }
            return;
          }
          const delay = status.poll_after_ms && status.poll_after_ms > 0 ? status.poll_after_ms : 2000;
          timeoutId = window.setTimeout(poll, delay);
        } catch (error) {
          if (!active) return;
          setBuildError((error as Error).message);
          timeoutId = window.setTimeout(poll, 4000);
        }
      };

      poll();

      return () => {
        active = false;
        if (timeoutId) {
          window.clearTimeout(timeoutId);
        }
      };
    }, [buildJobId, runExport]);

    const isBusy = generationPhase !== "idle";
    const buttonLabel =
      generationPhase === "building"
        ? "Building bundle..."
        : generationPhase === "exporting"
          ? "Exporting bundle..."
          : "Generate bundle";
    const buildStatusTone = (status?: string) => {
      switch (status) {
        case "success":
          return "bg-emerald-500/10 text-emerald-200 border-emerald-500/40";
        case "partial":
          return "bg-amber-500/10 text-amber-200 border-amber-500/40";
        case "failed":
          return "bg-rose-500/10 text-rose-200 border-rose-500/40";
        case "building":
          return "bg-blue-500/10 text-blue-200 border-blue-500/40";
        default:
          return "bg-slate-500/10 text-slate-200 border-slate-500/40";
      }
    };

    return (
      <div className="rounded border border-slate-800 bg-black/20 p-3 space-y-3 text-xs text-slate-200">
        <div className="flex items-center justify-between gap-2">
          <div>
            <p className="text-xs uppercase tracking-wide text-slate-400">Bundle helper</p>
            <p className="text-sm font-semibold text-slate-100">Get bundle.json from deployment-manager</p>
          </div>
        </div>
        <p className="text-xs text-slate-400">
          Generate a bundle manifest and stage binaries/assets for offline use.
        </p>
        <div className="space-y-1">
          <Label className="text-[11px] uppercase tracking-wide text-slate-400">Tier</Label>
          <Select value={tier} onChange={(e) => setTier(e.target.value)} className="mt-0.5">
            <option value="tier-2-desktop">tier-2-desktop</option>
            <option value="tier-3-mobile" disabled>
              tier-3-mobile (desktop export only)
            </option>
          </Select>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" variant="outline" onClick={handleExport} disabled={isBusy}>
            {buttonLabel}
          </Button>
          {downloadUrl && downloadName && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                const link = document.createElement("a");
                link.href = downloadUrl;
                link.download = downloadName;
                link.click();
              }}
            >
              Download bundle
            </Button>
          )}
        </div>
        {generationPhase === "building" && !buildStatus && (
          <p className="text-[11px] text-slate-400">Starting auto build...</p>
        )}
        {buildStatus && (
          <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3 space-y-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <p className="text-[11px] uppercase tracking-wide text-slate-400">Auto build</p>
                <p className="text-sm font-semibold text-slate-100">Bundle binaries & assets</p>
              </div>
              <span className={`rounded-full border px-2 py-0.5 text-[11px] ${buildStatusTone(buildStatus.status)}`}>
                {buildStatus.status}
              </span>
            </div>
            <div className="text-[11px] text-slate-400">
              Build ID: <span className="font-mono text-slate-300">{buildStatus.build_id}</span>
            </div>
            <div className="space-y-1">
              <div className="flex justify-between text-[11px] text-slate-400">
                <span>Progress</span>
                <span className="text-slate-300">{buildProgress}%</span>
              </div>
              <Progress value={buildProgress} />
            </div>
            {buildTargets.length > 0 && (
              <div className="space-y-2">
                <p className="text-[11px] uppercase tracking-wide text-slate-500">Targets</p>
                {buildTargets.map((target) => (
                  <div key={target.id} className="rounded-md border border-slate-800/70 bg-slate-950/70 p-2 space-y-1">
                    <div className="flex items-center justify-between gap-2">
                      <p className="text-xs font-semibold text-slate-200">{target.id}</p>
                      <span className="text-[11px] text-slate-500">{target.folder}</span>
                    </div>
                    <div className="space-y-1">
                      {(target.platforms || []).map((platform) => {
                        const isBuilding = platform.status === "building";
                        const isSuccess = platform.status === "success";
                        const isFailed = platform.status === "failed";
                        return (
                          <div key={platform.name} className="flex flex-wrap items-center justify-between gap-2 text-[11px]">
                            <span className="flex items-center gap-2 text-slate-300">
                              {isSuccess ? (
                                <CheckCircle className="h-4 w-4 text-emerald-400" />
                              ) : isFailed ? (
                                <AlertTriangle className="h-4 w-4 text-rose-400" />
                              ) : isBuilding ? (
                                <Loader2 className="h-4 w-4 text-blue-400 animate-spin" />
                              ) : (
                                <span className="h-4 w-4 rounded-full border border-slate-600" />
                              )}
                              {platform.name}
                            </span>
                            <span className="max-w-[240px] truncate text-slate-500">{platform.output_path}</span>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                ))}
              </div>
            )}
            {buildLog.length > 0 && (
              <div>
                <p className="text-[11px] uppercase tracking-wide text-slate-500 mb-1">Build log</p>
                <div className="max-h-40 overflow-auto rounded-md border border-slate-800/70 bg-slate-950/80 p-2 font-mono text-[11px] text-slate-300">
                  {buildLog.map((line, idx) => (
                    <div key={idx}>{line}</div>
                  ))}
                </div>
              </div>
            )}
            {buildErrors.length > 0 && (
              <div>
                <p className="text-[11px] uppercase tracking-wide text-rose-300 mb-1">Build errors</p>
                <div className="max-h-40 overflow-auto rounded-md border border-rose-500/40 bg-rose-950/40 p-2 font-mono text-[11px] text-rose-200">
                  {buildErrors.map((line, idx) => (
                    <div key={idx}>{line}</div>
                  ))}
                </div>
              </div>
            )}
            {buildStatus.check_command && (
              <p className="text-[11px] text-slate-400">
                Check progress: <span className="font-mono text-slate-300">{buildStatus.check_command}</span>
              </p>
            )}
          </div>
        )}
        {buildError && (
          <p className="rounded border border-amber-500/40 bg-amber-500/10 px-2 py-1 text-amber-100">
            {buildError}
          </p>
        )}
        {exportMeta && (
          <p className="text-[11px] text-slate-300">
            Generated {exportMeta.generated_at || "just now"}
            {exportMeta.checksum ? ` · checksum ${exportMeta.checksum.slice(0, 8)}...` : ""}
          </p>
        )}
        {manifestPath && (
          <p className="text-[11px] text-slate-300">
            Saved to {manifestPath}
          </p>
        )}
        {exportError && (
          <p className="rounded border border-amber-500/40 bg-amber-500/10 px-2 py-1 text-amber-100">
            {exportError}
          </p>
        )}
        {!exportError && !downloadUrl && (
          <p className="text-[11px] text-slate-400">
            Start deployment-manager (`vrooli scenario start deployment-manager`), then click Generate. We'll auto-build
            scenario binaries first, then export the manifest + staged files into the scenario bundle folder.
          </p>
        )}
      </div>
    );
  }
);
