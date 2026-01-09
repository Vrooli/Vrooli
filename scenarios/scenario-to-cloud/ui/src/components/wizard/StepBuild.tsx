import { useEffect, useState } from "react";
import { Play, FileArchive, Hash, HardDrive, Clock, CheckCircle2, History, Trash2, Server, ChevronDown, ChevronRight, Info, ArrowLeft } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { LoadingState, Spinner } from "../ui/spinner";
import { Card, CardContent } from "../ui/card";
import { selectors } from "../../consts/selectors";
import type { useDeployment } from "../../hooks/useDeployment";
import { listBundles, cleanupBundles, deleteBundle, listVPSBundles, deleteVPSBundle, type BundleInfo, type VPSBundleInfo } from "../../lib/api";

interface StepBuildProps {
  deployment: ReturnType<typeof useDeployment>;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}

function formatDate(isoString: string): string {
  const date = new Date(isoString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins} minute${diffMins === 1 ? "" : "s"} ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours === 1 ? "" : "s"} ago`;
  if (diffDays < 7) return `${diffDays} day${diffDays === 1 ? "" : "s"} ago`;

  return date.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  });
}

export interface BuildStatusPanelProps {
  mode?: "interactive" | "readonly";
  isBuilding: boolean;
  bundleArtifact: { path: string; sha256: string; size_bytes: number } | null;
  bundleError?: string | null;
  canBuild?: boolean;
  onBuild?: () => void;
  onClearBundle?: () => void;
  bundlePathFallback?: string | null;
  bundleShaFallback?: string | null;
  bundleSizeBytesFallback?: number | null;
  buildButtonTestId?: string;
  bundleResultTestId?: string;
}

export function BuildStatusPanel({
  mode = "interactive",
  isBuilding,
  bundleArtifact,
  bundleError,
  canBuild = false,
  onBuild,
  onClearBundle,
  bundlePathFallback,
  bundleShaFallback,
  bundleSizeBytesFallback,
  buildButtonTestId,
  bundleResultTestId,
}: BuildStatusPanelProps) {
  const isInteractive = mode === "interactive";
  const bundlePath = bundleArtifact?.path ?? bundlePathFallback ?? null;
  const bundleSha = bundleArtifact?.sha256 ?? bundleShaFallback ?? null;
  const bundleSizeBytes = bundleArtifact?.size_bytes ?? bundleSizeBytesFallback ?? null;
  const hasBundleDetails = Boolean(bundlePath || bundleSha || bundleSizeBytes);

  return (
    <div className="space-y-4">
      {isInteractive && (
        <div className="flex items-center gap-3">
          <Button
            data-testid={buildButtonTestId}
            onClick={onBuild}
            disabled={isBuilding || !canBuild}
          >
            <Play className="h-4 w-4 mr-1.5" />
            {isBuilding ? "Building..." : "Build New Bundle"}
          </Button>
          {!bundleArtifact && !isBuilding && (
            <span className="text-sm text-slate-400">
              Creates a fresh deployable tarball with required dependencies
            </span>
          )}
        </div>
      )}

      {isBuilding && (
        <LoadingState message="Building mini-Vrooli bundle..." />
      )}

      {bundleError && (
        <Alert variant="error" title="Bundle Build Failed">
          {bundleError}
        </Alert>
      )}

      {!isBuilding && !bundleError && !hasBundleDetails && (
        <Alert variant="info" title="Bundle Status">
          {isInteractive
            ? "Build a new bundle or select an existing one to continue."
            : "Bundle details will appear here once the build completes."}
        </Alert>
      )}

      {hasBundleDetails && !isBuilding && (
        <div data-testid={bundleResultTestId} className="space-y-4">
          <Alert variant="success" title="Bundle Ready" className="flex-1">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4" />
              {isInteractive
                ? "Your deployment bundle is ready for upload to the target server."
                : "Bundle is ready for deployment."}
            </div>
          </Alert>

          {isInteractive && onClearBundle && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onClearBundle}
              className="text-slate-400"
            >
              <ArrowLeft className="h-4 w-4 mr-1.5" />
              Change Build
            </Button>
          )}

          <Card>
            <CardContent className="py-4">
              <h4 className="text-sm font-medium text-slate-300 mb-4">Bundle Artifact</h4>

              <div className="space-y-4">
                {bundlePath && (
                  <div className="flex items-start gap-3">
                    <FileArchive className="h-5 w-5 text-slate-400 flex-shrink-0 mt-0.5" />
                    <div className="min-w-0 flex-1">
                      <p className="text-xs text-slate-500 mb-1">File Path</p>
                      <code className="block text-sm text-slate-200 font-mono break-all bg-slate-800/50 rounded px-2 py-1">
                        {bundlePath}
                      </code>
                    </div>
                  </div>
                )}

                {bundleSha && (
                  <div className="flex items-start gap-3">
                    <Hash className="h-5 w-5 text-slate-400 flex-shrink-0 mt-0.5" />
                    <div className="min-w-0 flex-1">
                      <p className="text-xs text-slate-500 mb-1">SHA256 Checksum</p>
                      <code className="block text-sm text-slate-200 font-mono break-all bg-slate-800/50 rounded px-2 py-1">
                        {bundleSha}
                      </code>
                    </div>
                  </div>
                )}

                {typeof bundleSizeBytes === "number" && (
                  <div className="flex items-start gap-3">
                    <HardDrive className="h-5 w-5 text-slate-400 flex-shrink-0 mt-0.5" />
                    <div className="min-w-0 flex-1">
                      <p className="text-xs text-slate-500 mb-1">Bundle Size</p>
                      <p className="text-sm text-slate-200">
                        {formatBytes(bundleSizeBytes)}
                        <span className="text-slate-500 text-xs ml-2">
                          ({bundleSizeBytes.toLocaleString()} bytes)
                        </span>
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}

export function StepBuild({ deployment }: StepBuildProps) {
  const {
    bundleArtifact,
    bundleError,
    isBuildingBundle,
    build,
    parsedManifest,
    setBundleArtifact,
  } = deployment;

  const [existingBundles, setExistingBundles] = useState<BundleInfo[]>([]);
  const [isLoadingBundles, setIsLoadingBundles] = useState(true);
  const [bundlesError, setBundlesError] = useState<string | null>(null);
  const [isCleaningUp, setIsCleaningUp] = useState(false);
  const [cleanupMessage, setCleanupMessage] = useState<string | null>(null);
  const [deletingBundleId, setDeletingBundleId] = useState<string | null>(null);

  // VPS bundles state
  const [vpsBundles, setVpsBundles] = useState<VPSBundleInfo[]>([]);
  const [isLoadingVpsBundles, setIsLoadingVpsBundles] = useState(false);
  const [vpsBundlesError, setVpsBundlesError] = useState<string | null>(null);
  const [vpsExpanded, setVpsExpanded] = useState(false);
  const [isCleaningVps, setIsCleaningVps] = useState(false);
  const [vpsCleanupMessage, setVpsCleanupMessage] = useState<string | null>(null);
  const [deletingVpsBundleId, setDeletingVpsBundleId] = useState<string | null>(null);

  // Get the scenario ID and VPS config from the manifest
  const scenarioId = parsedManifest.ok ? parsedManifest.value?.scenario?.id : null;
  const vpsTarget = parsedManifest.ok ? parsedManifest.value?.target?.vps : null;
  const vpsConfig = vpsTarget ? {
    host: vpsTarget.host,
    port: vpsTarget.port || 22,
    user: vpsTarget.user || "root",
    key_path: vpsTarget.key_path,
    workdir: vpsTarget.workdir || "/opt/vrooli",
  } : null;
  const hasVpsConfig = vpsConfig?.host && vpsConfig?.key_path;

  // Fetch existing bundles on mount
  useEffect(() => {
    async function fetchBundles() {
      setIsLoadingBundles(true);
      setBundlesError(null);
      try {
        const response = await listBundles();
        setExistingBundles(response.bundles);
      } catch (err) {
        setBundlesError(err instanceof Error ? err.message : "Failed to load bundles");
      } finally {
        setIsLoadingBundles(false);
      }
    }
    fetchBundles();
  }, []);

  // Filter bundles for the current scenario
  const matchingBundles = existingBundles.filter(
    (b) => scenarioId && b.scenario_id === scenarioId
  );

  // Select an existing bundle
  const selectBundle = (bundle: BundleInfo) => {
    setBundleArtifact({
      path: bundle.path,
      sha256: bundle.sha256,
      size_bytes: bundle.size_bytes,
    });
  };

  // Handle bundle cleanup
  const handleCleanup = async () => {
    setIsCleaningUp(true);
    setCleanupMessage(null);
    try {
      const result = await cleanupBundles({ keep_latest: 3 });
      setCleanupMessage(result.message);
      // Refresh bundles list
      const response = await listBundles();
      setExistingBundles(response.bundles);
    } catch (err) {
      setCleanupMessage(err instanceof Error ? err.message : "Cleanup failed");
    } finally {
      setIsCleaningUp(false);
    }
  };

  // Handle individual bundle deletion
  const handleDeleteBundle = async (sha256: string, e: React.MouseEvent) => {
    e.stopPropagation(); // Don't trigger card click
    setDeletingBundleId(sha256);
    try {
      await deleteBundle(sha256);
      // Refresh bundles list
      const response = await listBundles();
      setExistingBundles(response.bundles);
    } catch (err) {
      setBundlesError(err instanceof Error ? err.message : "Failed to delete bundle");
    } finally {
      setDeletingBundleId(null);
    }
  };

  // Fetch VPS bundles
  const fetchVpsBundles = async () => {
    if (!hasVpsConfig || !vpsConfig) return;
    setIsLoadingVpsBundles(true);
    setVpsBundlesError(null);
    try {
      const response = await listVPSBundles({
        host: vpsConfig.host!,
        port: vpsConfig.port,
        user: vpsConfig.user,
        key_path: vpsConfig.key_path!,
        workdir: vpsConfig.workdir,
      });
      if (response.ok) {
        setVpsBundles(response.bundles);
      } else {
        setVpsBundlesError(response.error || "Failed to list VPS bundles");
      }
    } catch (err) {
      setVpsBundlesError(err instanceof Error ? err.message : "Failed to connect to VPS");
    } finally {
      setIsLoadingVpsBundles(false);
    }
  };

  // Handle VPS cleanup
  const handleVpsCleanup = async () => {
    if (!hasVpsConfig || !vpsConfig) return;
    setIsCleaningVps(true);
    setVpsCleanupMessage(null);
    try {
      const result = await cleanupBundles({
        keep_latest: 3,
        clean_vps: true,
        host: vpsConfig.host!,
        port: vpsConfig.port,
        user: vpsConfig.user,
        key_path: vpsConfig.key_path!,
        workdir: vpsConfig.workdir,
      });
      setVpsCleanupMessage(result.message);
      // Refresh VPS bundles
      await fetchVpsBundles();
    } catch (err) {
      setVpsCleanupMessage(err instanceof Error ? err.message : "VPS cleanup failed");
    } finally {
      setIsCleaningVps(false);
    }
  };

  // Handle individual VPS bundle deletion
  const handleDeleteVpsBundle = async (filename: string) => {
    if (!hasVpsConfig || !vpsConfig) return;
    setDeletingVpsBundleId(filename);
    setVpsBundlesError(null);
    try {
      const result = await deleteVPSBundle({
        host: vpsConfig.host!,
        port: vpsConfig.port,
        user: vpsConfig.user,
        key_path: vpsConfig.key_path!,
        workdir: vpsConfig.workdir,
        filename,
      });
      if (!result.ok) {
        setVpsBundlesError(result.error || "Failed to delete bundle");
      } else {
        // Remove from local state
        setVpsBundles((prev) => prev.filter((b) => b.filename !== filename));
      }
    } catch (err) {
      setVpsBundlesError(err instanceof Error ? err.message : "Failed to delete VPS bundle");
    } finally {
      setDeletingVpsBundleId(null);
    }
  };

  // Calculate storage
  const totalStorageBytes = existingBundles.reduce((sum, b) => sum + b.size_bytes, 0);
  const scenarioStorageBytes = matchingBundles.reduce((sum, b) => sum + b.size_bytes, 0);
  const vpsTotalBytes = vpsBundles.reduce((sum, b) => sum + b.size_bytes, 0);

  return (
    <div className="space-y-6">
      {/* Existing Bundles Section - visible during build too (dimmed) */}
      {!bundleArtifact && (
        <div className={`space-y-3 ${isBuildingBundle ? "opacity-50 pointer-events-none" : ""}`}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-sm text-slate-400">
              <History className="h-4 w-4" />
              <span>
                {matchingBundles.length} build{matchingBundles.length !== 1 ? "s" : ""} for {scenarioId || "this scenario"}
                {matchingBundles.length > 0 && (
                  <span className="text-slate-500 ml-1">({formatBytes(scenarioStorageBytes)})</span>
                )}
              </span>
            </div>
          </div>

          {isLoadingBundles && (
            <div className="flex items-center gap-2 text-sm text-slate-500 py-2">
              <Spinner size="sm" />
              <span>Loading existing builds...</span>
            </div>
          )}

          {bundlesError && (
            <Alert variant="warning" title="Could not load existing builds">
              {bundlesError}
            </Alert>
          )}

          {!isLoadingBundles && !bundlesError && matchingBundles.length === 0 && (
            <p className="text-sm text-slate-500 py-2">
              No existing builds found{scenarioId ? ` for "${scenarioId}"` : ""}. Click "Build Bundle" to create one.
            </p>
          )}

          {!isLoadingBundles && matchingBundles.length > 0 && (
            <div className="space-y-2">
              {matchingBundles.map((bundle) => (
                <Card
                  key={bundle.sha256}
                  className="cursor-pointer hover:border-blue-500/50 transition-colors"
                  onClick={() => selectBundle(bundle)}
                >
                  <CardContent className="py-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3 min-w-0">
                        <FileArchive className="h-5 w-5 text-slate-400 flex-shrink-0" />
                        <div className="min-w-0">
                          <p className="text-sm text-slate-200 truncate">
                            {bundle.filename}
                          </p>
                          <div className="flex items-center gap-3 text-xs text-slate-500">
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              {formatDate(bundle.created_at)}
                            </span>
                            <span>{formatBytes(bundle.size_bytes)}</span>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-slate-400 hover:text-red-400"
                          onClick={(e) => handleDeleteBundle(bundle.sha256, e)}
                          disabled={deletingBundleId === bundle.sha256}
                        >
                          {deletingBundleId === bundle.sha256 ? (
                            <Spinner size="sm" />
                          ) : (
                            <Trash2 className="h-4 w-4" />
                          )}
                        </Button>
                        <Button variant="outline" size="sm">
                          Use This Build
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

          {/* Storage Summary - now shows total across all scenarios */}
          {!isLoadingBundles && existingBundles.length > 0 && (
            <Card className="bg-slate-800/30 border-slate-700/50">
              <CardContent className="py-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-sm text-slate-400">
                    <HardDrive className="h-4 w-4" />
                    <span>
                      Total: {existingBundles.length} bundle{existingBundles.length !== 1 ? "s" : ""} across all scenarios using{" "}
                      {formatBytes(totalStorageBytes)}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-slate-500 hidden sm:inline" title="Keeps the 3 most recent builds per scenario, deletes older ones">
                      <Info className="h-3 w-3 inline mr-1" />
                      Keeps 3 latest per scenario
                    </span>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleCleanup}
                      disabled={isCleaningUp}
                      title="Keeps 3 most recent builds per scenario, deletes older ones"
                    >
                      {isCleaningUp ? (
                        <>
                          <Spinner size="sm" className="mr-1.5" />
                          Cleaning...
                        </>
                      ) : (
                        <>
                          <Trash2 className="h-4 w-4 mr-1.5" />
                          Clean up old
                        </>
                      )}
                    </Button>
                  </div>
                </div>
                {cleanupMessage && (
                  <p className="text-xs text-slate-500 mt-2">{cleanupMessage}</p>
                )}
              </CardContent>
            </Card>
          )}

          {/* VPS Bundles Section */}
          {hasVpsConfig && (
            <Card className="bg-slate-800/30 border-slate-700/50">
              <CardContent className="py-3">
                <button
                  type="button"
                  className="w-full flex items-center justify-between text-left"
                  onClick={() => {
                    setVpsExpanded(!vpsExpanded);
                    if (!vpsExpanded && vpsBundles.length === 0 && !isLoadingVpsBundles) {
                      fetchVpsBundles();
                    }
                  }}
                >
                  <div className="flex items-center gap-2 text-sm text-slate-400">
                    <Server className="h-4 w-4" />
                    <span>VPS Bundles ({vpsConfig?.host})</span>
                  </div>
                  {vpsExpanded ? (
                    <ChevronDown className="h-4 w-4 text-slate-500" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-slate-500" />
                  )}
                </button>

                {vpsExpanded && (
                  <div className="mt-3 space-y-3">
                    {isLoadingVpsBundles && (
                      <div className="flex items-center gap-2 text-sm text-slate-500">
                        <Spinner size="sm" />
                        <span>Connecting to VPS...</span>
                      </div>
                    )}

                    {vpsBundlesError && (
                      <p className="text-sm text-red-400">{vpsBundlesError}</p>
                    )}

                    {!isLoadingVpsBundles && !vpsBundlesError && vpsBundles.length === 0 && (
                      <p className="text-sm text-slate-500">No bundles found on VPS.</p>
                    )}

                    {!isLoadingVpsBundles && vpsBundles.length > 0 && (
                      <>
                        <div className="space-y-1">
                          {vpsBundles.map((bundle) => (
                            <div key={bundle.sha256} className="flex items-center justify-between text-sm py-1 group">
                              <div className="flex items-center gap-2 min-w-0">
                                <FileArchive className="h-4 w-4 text-slate-500 flex-shrink-0" />
                                <span className="text-slate-300 truncate">{bundle.filename}</span>
                              </div>
                              <div className="flex items-center gap-2">
                                <span className="text-slate-500 text-xs">{formatBytes(bundle.size_bytes)}</span>
                                <button
                                  type="button"
                                  onClick={() => handleDeleteVpsBundle(bundle.filename)}
                                  disabled={deletingVpsBundleId === bundle.filename}
                                  className="opacity-0 group-hover:opacity-100 text-slate-400 hover:text-red-400 transition-opacity p-1"
                                  title="Delete this bundle from VPS"
                                >
                                  {deletingVpsBundleId === bundle.filename ? (
                                    <Spinner size="sm" />
                                  ) : (
                                    <Trash2 className="h-3.5 w-3.5" />
                                  )}
                                </button>
                              </div>
                            </div>
                          ))}
                        </div>
                        <div className="flex items-center justify-between pt-2 border-t border-slate-700/50">
                          <span className="text-xs text-slate-500">
                            {vpsBundles.length} bundle{vpsBundles.length !== 1 ? "s" : ""} using {formatBytes(vpsTotalBytes)}
                          </span>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={handleVpsCleanup}
                            disabled={isCleaningVps}
                            title="Keeps 3 most recent builds per scenario on VPS"
                          >
                            {isCleaningVps ? (
                              <>
                                <Spinner size="sm" className="mr-1.5" />
                                Cleaning...
                              </>
                            ) : (
                              <>
                                <Trash2 className="h-4 w-4 mr-1.5" />
                                Clean up VPS
                              </>
                            )}
                          </Button>
                        </div>
                        {vpsCleanupMessage && (
                          <p className="text-xs text-slate-500">{vpsCleanupMessage}</p>
                        )}
                      </>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {/* Divider when bundles exist */}
      {!bundleArtifact && !isBuildingBundle && matchingBundles.length > 0 && (
        <div className="flex items-center gap-4">
          <div className="flex-1 border-t border-slate-700" />
          <span className="text-xs text-slate-500 uppercase">or</span>
          <div className="flex-1 border-t border-slate-700" />
        </div>
      )}

      <BuildStatusPanel
        mode="interactive"
        isBuilding={isBuildingBundle}
        bundleArtifact={bundleArtifact}
        bundleError={bundleError}
        canBuild={parsedManifest.ok}
        onBuild={build}
        onClearBundle={() => setBundleArtifact(null)}
        buildButtonTestId={selectors.manifest.bundleBuildButton}
        bundleResultTestId={selectors.manifest.bundleBuildResult}
      />
    </div>
  );
}
