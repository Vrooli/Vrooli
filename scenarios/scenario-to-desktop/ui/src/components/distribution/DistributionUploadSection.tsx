import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Cloud, CheckCircle, XCircle, RefreshCw, Upload, ExternalLink } from "lucide-react";
import {
  fetchDistributionTargets,
  startDistribution,
  fetchDistributionStatus,
  type DistributionTarget,
  type DistributionStatus,
} from "../../lib/api";
import { Button } from "../ui/button";
import { Checkbox } from "../ui/checkbox";
import { cn } from "../../lib/utils";

interface DistributionUploadSectionProps {
  scenarioName: string;
  artifacts: Record<string, string>;
  version?: string;
}

export function DistributionUploadSection({
  scenarioName,
  artifacts,
  version,
}: DistributionUploadSectionProps) {
  const [distributionId, setDistributionId] = useState<string | null>(null);
  const [selectedTargets, setSelectedTargets] = useState<string[]>([]);
  const [uploadError, setUploadError] = useState<string | null>(null);

  // Fetch available targets
  const { data: targetsData, isLoading: targetsLoading } = useQuery({
    queryKey: ["distribution-targets"],
    queryFn: fetchDistributionTargets,
  });

  // Poll distribution status
  const { data: distributionStatus } = useQuery<DistributionStatus | null>({
    queryKey: ["distribution-status", distributionId],
    queryFn: async () => (distributionId ? fetchDistributionStatus(distributionId) : null),
    enabled: !!distributionId,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === "completed" || status === "partial" || status === "failed" || status === "cancelled") {
        return false;
      }
      return 2000;
    },
  });

  // Start distribution mutation
  const distributeMutation = useMutation({
    mutationFn: () =>
      startDistribution({
        scenario_name: scenarioName,
        version,
        artifacts,
        target_names: selectedTargets.length > 0 ? selectedTargets : undefined,
        parallel: true,
      }),
    onSuccess: (resp) => {
      setDistributionId(resp.distribution_id);
      setUploadError(null);
    },
    onError: (error: Error) => {
      setUploadError(error.message);
    },
  });

  const enabledTargets = (targetsData?.targets || []).filter((t) => t.enabled);
  const hasTargets = enabledTargets.length > 0;
  const hasArtifacts = Object.keys(artifacts).length > 0;

  const toggleTarget = (name: string) => {
    setSelectedTargets((prev) =>
      prev.includes(name) ? prev.filter((n) => n !== name) : [...prev, name]
    );
  };

  const isUploading =
    distributeMutation.isPending ||
    (distributionStatus?.status === "pending" || distributionStatus?.status === "running");

  // Show distribution status if we have one
  if (distributionStatus) {
    return (
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <Cloud className="h-5 w-5 text-blue-400" />
          <span className="font-semibold text-slate-200">Distribution Status</span>
        </div>

        <DistributionStatusDisplay status={distributionStatus} />

        {(distributionStatus.status === "completed" ||
          distributionStatus.status === "partial" ||
          distributionStatus.status === "failed") && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setDistributionId(null);
              setUploadError(null);
            }}
          >
            Start New Upload
          </Button>
        )}
      </div>
    );
  }

  if (!hasArtifacts) {
    return null;
  }

  if (targetsLoading) {
    return (
      <div className="flex items-center gap-2 text-slate-400">
        <RefreshCw className="h-4 w-4 animate-spin" />
        <span className="text-sm">Loading distribution targets...</span>
      </div>
    );
  }

  if (!hasTargets) {
    return (
      <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-4">
        <div className="flex items-center gap-2 mb-2">
          <Cloud className="h-5 w-5 text-slate-500" />
          <span className="font-semibold text-slate-300">Distribution</span>
        </div>
        <p className="text-sm text-slate-400 mb-3">
          No distribution targets configured. Add targets in the Distribution tab to upload installers to cloud storage.
        </p>
        <a
          href="?view=distribution"
          className="text-sm text-blue-300 hover:text-blue-200 flex items-center gap-1"
        >
          Configure Distribution Targets
          <ExternalLink className="h-3 w-3" />
        </a>
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-4 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Cloud className="h-5 w-5 text-blue-400" />
          <span className="font-semibold text-slate-200">Distribute Installers</span>
        </div>
        <span className="text-xs text-slate-400">
          {Object.keys(artifacts).length} artifact{Object.keys(artifacts).length !== 1 ? "s" : ""} ready
        </span>
      </div>

      <div className="space-y-2">
        <p className="text-xs text-slate-400">Select targets to upload to:</p>
        <div className="grid gap-2 md:grid-cols-2">
          {enabledTargets.map((target) => (
            <TargetSelector
              key={target.name}
              target={target}
              selected={selectedTargets.length === 0 || selectedTargets.includes(target.name)}
              onToggle={() => toggleTarget(target.name)}
            />
          ))}
        </div>
        <p className="text-xs text-slate-500">
          {selectedTargets.length === 0
            ? "All enabled targets will be used"
            : `${selectedTargets.length} target${selectedTargets.length !== 1 ? "s" : ""} selected`}
        </p>
      </div>

      {uploadError && (
        <div className="p-2 rounded bg-red-950/30 border border-red-800/30 text-sm text-red-300">
          {uploadError}
        </div>
      )}

      <Button
        onClick={() => distributeMutation.mutate()}
        disabled={isUploading}
        className="w-full"
      >
        {isUploading ? (
          <>
            <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
            Uploading...
          </>
        ) : (
          <>
            <Upload className="h-4 w-4 mr-2" />
            Upload to Cloud Storage
          </>
        )}
      </Button>
    </div>
  );
}

function TargetSelector({
  target,
  selected,
  onToggle,
}: {
  target: DistributionTarget;
  selected: boolean;
  onToggle: () => void;
}) {
  const providerLabel: Record<string, string> = {
    s3: "S3",
    r2: "R2",
    "s3-compatible": "S3",
  };

  return (
    <button
      type="button"
      onClick={onToggle}
      className={cn(
        "flex items-center gap-3 p-2 rounded border text-left transition",
        selected
          ? "border-blue-700 bg-blue-950/30"
          : "border-slate-700 bg-slate-900/30 opacity-60"
      )}
    >
      <Checkbox checked={selected} onChange={onToggle} />
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-slate-200 truncate">{target.name}</div>
        <div className="text-xs text-slate-400">
          {providerLabel[target.provider] || target.provider} • {target.bucket}
        </div>
      </div>
    </button>
  );
}

function DistributionStatusDisplay({ status }: { status: DistributionStatus }) {
  const statusColors: Record<string, string> = {
    pending: "text-slate-400",
    running: "text-blue-400",
    completed: "text-green-400",
    partial: "text-yellow-400",
    failed: "text-red-400",
    cancelled: "text-slate-500",
  };

  const statusIcons: Record<string, React.ReactNode> = {
    pending: <RefreshCw className="h-4 w-4" />,
    running: <RefreshCw className="h-4 w-4 animate-spin" />,
    completed: <CheckCircle className="h-4 w-4" />,
    partial: <CheckCircle className="h-4 w-4" />,
    failed: <XCircle className="h-4 w-4" />,
    cancelled: <XCircle className="h-4 w-4" />,
  };

  return (
    <div className="space-y-3">
      <div className={cn("flex items-center gap-2", statusColors[status.status])}>
        {statusIcons[status.status]}
        <span className="font-medium capitalize">{status.status}</span>
      </div>

      {status.error && (
        <div className="p-2 rounded bg-red-950/30 border border-red-800/30 text-sm text-red-300">
          {status.error}
        </div>
      )}

      <div className="space-y-2">
        {Object.entries(status.targets).map(([targetName, targetDist]) => (
          <div
            key={targetName}
            className="rounded border border-slate-800 bg-slate-900/40 p-3"
          >
            <div className="flex items-center justify-between mb-2">
              <span className="font-medium text-slate-200">{targetName}</span>
              <span className={cn("text-xs capitalize", statusColors[targetDist.status])}>
                {targetDist.status}
              </span>
            </div>

            {Object.entries(targetDist.uploads).map(([platform, upload]) => (
              <div key={platform} className="flex items-center justify-between text-sm py-1">
                <span className="text-slate-400 capitalize">{platform}</span>
                <div className="flex items-center gap-2">
                  {upload.status === "completed" && upload.url ? (
                    <a
                      href={upload.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-300 hover:text-blue-200 flex items-center gap-1"
                    >
                      <span className="truncate max-w-[200px]">{upload.url}</span>
                      <ExternalLink className="h-3 w-3 flex-shrink-0" />
                    </a>
                  ) : (
                    <span className={cn("text-xs capitalize", statusColors[upload.status])}>
                      {upload.status}
                    </span>
                  )}
                </div>
              </div>
            ))}

            {targetDist.error && (
              <p className="text-xs text-red-300 mt-1">{targetDist.error}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
