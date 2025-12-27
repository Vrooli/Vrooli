import { Package, Play, FileArchive, Hash, HardDrive } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { LoadingState } from "../ui/spinner";
import { Card, CardContent } from "../ui/card";
import { selectors } from "../../consts/selectors";
import type { useDeployment } from "../../hooks/useDeployment";

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

export function StepBuild({ deployment }: StepBuildProps) {
  const {
    bundleArtifact,
    bundleError,
    isBuildingBundle,
    build,
    parsedManifest,
  } = deployment;

  return (
    <div className="space-y-6">
      {/* Build Button */}
      <div className="flex items-center gap-3">
        <Button
          data-testid={selectors.manifest.bundleBuildButton}
          onClick={build}
          disabled={isBuildingBundle || !parsedManifest.ok}
        >
          <Play className="h-4 w-4 mr-1.5" />
          {isBuildingBundle ? "Building..." : "Build Bundle"}
        </Button>
        {!bundleArtifact && !isBuildingBundle && (
          <span className="text-sm text-slate-400">
            Creates a deployable tarball with required dependencies
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
          <Alert variant="success" title="Bundle Built Successfully">
            <div className="flex items-center gap-2">
              <Package className="h-4 w-4" />
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
