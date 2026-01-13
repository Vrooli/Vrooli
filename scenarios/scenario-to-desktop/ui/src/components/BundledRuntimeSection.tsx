import { useEffect, useState, type Ref } from "react";
import type { BundleManifestResponse } from "../lib/api";
import { fetchBundleManifest } from "../lib/api";
import { Braces, Copy, Download, LayoutList } from "lucide-react";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { DeploymentManagerBundleHelper, type DeploymentManagerBundleHelperHandle, type BundleResult } from "./DeploymentManagerBundleHelper";

interface BundledRuntimeSectionProps {
  bundleManifestPath: string;
  onBundleManifestChange: (value: string) => void;
  scenarioName: string;
  bundleHelperRef: Ref<DeploymentManagerBundleHelperHandle>;
  onBundleExported?: (manifestPath: string) => void;
  onDeploymentManagerUrlChange?: (url: string | null) => void;
  /** Called when bundle export completes successfully. Use to persist stage results. */
  onBundleComplete?: (result: BundleResult) => void;
  /** Initial state for bundle helper restoration from server persistence. */
  initialBundleResult?: BundleResult | null;
}

export function BundledRuntimeSection({
  bundleManifestPath,
  onBundleManifestChange,
  scenarioName,
  bundleHelperRef,
  onBundleExported,
  onDeploymentManagerUrlChange,
  onBundleComplete,
  initialBundleResult,
}: BundledRuntimeSectionProps) {
  const [viewMode, setViewMode] = useState<"summary" | "json">("summary");
  const [manifestResult, setManifestResult] = useState<BundleManifestResponse | null>(null);
  const [manifestError, setManifestError] = useState<string | null>(null);
  const [manifestLoading, setManifestLoading] = useState(false);
  const [manifestCopyStatus, setManifestCopyStatus] = useState<"idle" | "copied" | "error">("idle");

  const manifestName = bundleManifestPath.trim()
    ? bundleManifestPath.trim().split(/[\\/]/).pop() || "bundle.json"
    : "bundle.json";

  useEffect(() => {
    setManifestResult(null);
    setManifestError(null);
    setManifestLoading(false);
  }, [bundleManifestPath]);

  useEffect(() => {
    if (viewMode !== "json") {
      return;
    }
    if (!bundleManifestPath.trim()) {
      setManifestResult(null);
      setManifestError("Add a bundle manifest path to view the JSON.");
      return;
    }
    let isActive = true;
    setManifestLoading(true);
    setManifestError(null);
    fetchBundleManifest({ bundle_manifest_path: bundleManifestPath })
      .then((result) => {
        if (!isActive) return;
        setManifestResult(result);
        setManifestError(null);
      })
      .catch((error) => {
        if (!isActive) return;
        setManifestResult(null);
        setManifestError((error as Error).message);
      })
      .finally(() => {
        if (!isActive) return;
        setManifestLoading(false);
      });
    return () => {
      isActive = false;
    };
  }, [bundleManifestPath, viewMode]);

  const manifestJson = manifestResult?.manifest ? JSON.stringify(manifestResult.manifest, null, 2) : "";

  const copyManifestJson = async () => {
    if (!manifestJson) {
      return;
    }
    try {
      await navigator.clipboard.writeText(manifestJson);
      setManifestCopyStatus("copied");
      setTimeout(() => setManifestCopyStatus("idle"), 1500);
    } catch (error) {
      console.warn("Failed to copy manifest JSON", error);
      setManifestCopyStatus("error");
      setTimeout(() => setManifestCopyStatus("idle"), 2000);
    }
  };

  const downloadManifestJson = () => {
    if (!manifestJson) {
      return;
    }
    const blob = new Blob([manifestJson], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = manifestName;
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <Label htmlFor="bundleManifest">Bundle manifest</Label>
          <p className="text-xs text-slate-400 mt-1">
            Generate a bundle manifest (bundle.json) and staged files for offline use.
          </p>
        </div>
        <div className="flex items-center gap-1 rounded-full border border-slate-800/70 bg-slate-950/40 p-1 text-[11px]">
          <Button
            type="button"
            size="sm"
            variant={viewMode === "summary" ? "default" : "ghost"}
            onClick={() => setViewMode("summary")}
            aria-label="Show bundle manifest summary"
            className="h-10 w-10 p-0"
          >
            <LayoutList className="h-4 w-4" />
          </Button>
          <Button
            type="button"
            size="sm"
            variant={viewMode === "json" ? "default" : "ghost"}
            onClick={() => setViewMode("json")}
            aria-label="Show bundle manifest JSON"
            className="h-10 w-10 p-0"
          >
            <Braces className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {viewMode === "summary" && (
        <>
          <Input
            id="bundleManifest"
            value={bundleManifestPath}
            onChange={(e) => onBundleManifestChange(e.target.value)}
            placeholder="/home/you/Vrooli/scenarios/my-scenario/platforms/electron/bundle/bundle.json"
          />
          <div className="rounded-md border border-slate-800 bg-slate-950/60 p-3 text-xs text-slate-200 space-y-2">
            <p className="font-semibold text-slate-100">Current manifest</p>
            {bundleManifestPath.trim() ? (
              <>
                <p className="text-[11px] text-slate-300">{bundleManifestPath}</p>
                <p className="text-[11px] text-slate-400">
                  Expect this file to live alongside staged binaries/assets.
                </p>
              </>
            ) : (
              <p className="text-[11px] text-slate-400">
                Paste a bundle.json path or export one from deployment-manager below.
              </p>
            )}
          </div>
        </>
      )}

      {viewMode === "json" && (
        <div className="rounded-md border border-slate-800 bg-slate-950/60 p-3 space-y-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-xs font-semibold text-slate-200">Bundle manifest JSON</p>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={copyManifestJson}
                disabled={!manifestJson}
                aria-label="Copy bundle manifest JSON"
                className="h-10 w-10 p-0"
              >
                <Copy className="h-4 w-4" />
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={downloadManifestJson}
                disabled={!manifestJson}
                aria-label="Download bundle manifest JSON"
                className="h-10 w-10 p-0"
              >
                <Download className="h-4 w-4" />
              </Button>
            </div>
          </div>
          {manifestLoading && (
            <p className="text-[11px] text-slate-400">Loading bundle manifest...</p>
          )}
          {manifestError && (
            <p className="text-[11px] text-amber-200">{manifestError}</p>
          )}
          {!manifestLoading && !manifestError && (
            <pre className="max-h-96 overflow-auto whitespace-pre-wrap rounded-md border border-slate-800/70 bg-slate-950/80 p-3 text-[11px] text-slate-200">
              {manifestJson || "No manifest loaded."}
            </pre>
          )}
          {manifestCopyStatus !== "idle" && (
            <p className="text-[11px] text-slate-400">
              {manifestCopyStatus === "copied" ? "Copied to clipboard." : "Copy failed."}
            </p>
          )}
        </div>
      )}

      <DeploymentManagerBundleHelper
        ref={bundleHelperRef}
        scenarioName={scenarioName}
        onBundleManifestChange={onBundleManifestChange}
        onBundleExported={onBundleExported}
        onDeploymentManagerUrlChange={onDeploymentManagerUrlChange}
        onBundleComplete={onBundleComplete}
        initialBuildStatus={initialBundleResult?.buildStatus}
        initialManifestPath={initialBundleResult?.manifestPath}
        initialExportMeta={
          initialBundleResult?.checksum || initialBundleResult?.generatedAt
            ? { checksum: initialBundleResult.checksum, generated_at: initialBundleResult.generatedAt }
            : null
        }
        initialDeploymentManagerUrl={initialBundleResult?.deploymentManagerUrl}
      />
    </div>
  );
}
