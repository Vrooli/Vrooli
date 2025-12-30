import { useEffect, useState } from "react";
import { Package, Play, FileArchive, Hash, HardDrive, Clock, CheckCircle2, History, Trash2 } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { LoadingState, Spinner } from "../ui/spinner";
import { Card, CardContent } from "../ui/card";
import { selectors } from "../../consts/selectors";
import type { useDeployment } from "../../hooks/useDeployment";
import { listBundles, cleanupBundles, type BundleInfo } from "../../lib/api";

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

  // Get the scenario ID from the manifest
  const scenarioId = parsedManifest.ok ? parsedManifest.value?.scenario?.id : null;

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

  // Calculate total storage used
  const totalStorageBytes = existingBundles.reduce((sum, b) => sum + b.size_bytes, 0);

  return (
    <div className="space-y-6">
      {/* Existing Bundles Section */}
      {!bundleArtifact && !isBuildingBundle && (
        <div className="space-y-3">
          <div className="flex items-center gap-2 text-sm text-slate-400">
            <History className="h-4 w-4" />
            <span>Existing Builds{scenarioId ? ` for ${scenarioId}` : ""}</span>
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
                      <Button variant="outline" size="sm">
                        Use This Build
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

          {/* Storage Summary */}
          {!isLoadingBundles && existingBundles.length > 0 && (
            <Card className="bg-slate-800/30 border-slate-700/50">
              <CardContent className="py-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-sm text-slate-400">
                    <HardDrive className="h-4 w-4" />
                    <span>
                      {existingBundles.length} bundle{existingBundles.length !== 1 ? "s" : ""} using{" "}
                      {formatBytes(totalStorageBytes)}
                    </span>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleCleanup}
                    disabled={isCleaningUp}
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
                {cleanupMessage && (
                  <p className="text-xs text-slate-500 mt-2">{cleanupMessage}</p>
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

      {/* Build Button */}
      <div className="flex items-center gap-3">
        <Button
          data-testid={selectors.manifest.bundleBuildButton}
          onClick={build}
          disabled={isBuildingBundle || !parsedManifest.ok}
        >
          <Play className="h-4 w-4 mr-1.5" />
          {isBuildingBundle ? "Building..." : "Build New Bundle"}
        </Button>
        {!bundleArtifact && !isBuildingBundle && (
          <span className="text-sm text-slate-400">
            Creates a fresh deployable tarball with required dependencies
          </span>
        )}
      </div>

      {/* Loading State */}
      {isBuildingBundle && (
        <LoadingState message="Building mini-Vrooli bundle..." />
      )}

      {/* Error */}
      {bundleError && (
        <Alert variant="error" title="Bundle Build Failed">
          {bundleError}
        </Alert>
      )}

      {/* Bundle Artifact */}
      {bundleArtifact && !isBuildingBundle && (
        <div data-testid={selectors.manifest.bundleBuildResult} className="space-y-4">
          <Alert variant="success" title="Bundle Ready">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4" />
              Your deployment bundle is ready for upload to the target server.
            </div>
          </Alert>

          <Card>
            <CardContent className="py-4">
              <h4 className="text-sm font-medium text-slate-300 mb-4">Bundle Artifact</h4>

              <div className="space-y-4">
                {/* Path */}
                <div className="flex items-start gap-3">
                  <FileArchive className="h-5 w-5 text-slate-400 flex-shrink-0 mt-0.5" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-slate-500 mb-1">File Path</p>
                    <code className="block text-sm text-slate-200 font-mono break-all bg-slate-800/50 rounded px-2 py-1">
                      {bundleArtifact.path}
                    </code>
                  </div>
                </div>

                {/* SHA256 */}
                <div className="flex items-start gap-3">
                  <Hash className="h-5 w-5 text-slate-400 flex-shrink-0 mt-0.5" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-slate-500 mb-1">SHA256 Checksum</p>
                    <code className="block text-sm text-slate-200 font-mono break-all bg-slate-800/50 rounded px-2 py-1">
                      {bundleArtifact.sha256}
                    </code>
                  </div>
                </div>

                {/* Size */}
                <div className="flex items-start gap-3">
                  <HardDrive className="h-5 w-5 text-slate-400 flex-shrink-0 mt-0.5" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-slate-500 mb-1">Bundle Size</p>
                    <p className="text-sm text-slate-200">
                      {formatBytes(bundleArtifact.size_bytes)}
                      <span className="text-slate-500 ml-2">
                        ({bundleArtifact.size_bytes.toLocaleString()} bytes)
                      </span>
                    </p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
