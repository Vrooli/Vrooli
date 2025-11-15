import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import type { PlatformBuildResult } from "../lib/api";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Badge } from "./ui/badge";
import { Input } from "./ui/input";
import { Button } from "./ui/button";
import { Monitor, Package, Search, Download, CheckCircle, XCircle, Loader2, Zap, Hammer, ExternalLink, AlertCircle, ChevronDown, ChevronUp, Copy, Check, RefreshCw, Trash2, FileDown } from "lucide-react";

interface ScenarioDesktopStatus {
  name: string;
  display_name?: string;
  has_desktop: boolean;
  desktop_path?: string;
  version?: string;
  platforms?: string[];
  built?: boolean;
  dist_path?: string;
  last_modified?: string;
  package_size?: number;
}

interface ScenariosResponse {
  scenarios: ScenarioDesktopStatus[];
  stats: {
    total: number;
    with_desktop: number;
    built: number;
    web_only: number;
  };
}

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

interface GenerateDesktopButtonProps {
  scenarioName: string;
}

function GenerateDesktopButton({ scenarioName }: GenerateDesktopButtonProps) {
  const queryClient = useQueryClient();
  const [buildId, setBuildId] = useState<string | null>(null);

  const generateMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(buildUrl('/desktop/generate/quick'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scenario_name: scenarioName,
          template_type: 'basic'
        })
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to generate desktop app');
      }
      return res.json();
    },
    onSuccess: (data) => {
      setBuildId(data.build_id);
      // Refresh scenarios list to show the new desktop version
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
      }, 3000);
    }
  });

  // Poll build status if we have a buildId
  const { data: buildStatus } = useQuery({
    queryKey: ['build-status', buildId],
    queryFn: async () => {
      if (!buildId) return null;
      const res = await fetch(buildUrl(`/desktop/status/${buildId}`));
      if (!res.ok) throw new Error('Failed to fetch build status');
      return res.json();
    },
    enabled: !!buildId,
    refetchInterval: (data) => {
      // Stop polling if build is complete or failed
      if (data?.status === 'ready' || data?.status === 'failed') {
        return false;
      }
      return 2000; // Poll every 2 seconds while building
    }
  });

  const isBuilding = generateMutation.isPending || buildStatus?.status === 'building';
  const isComplete = buildStatus?.status === 'ready';
  const isFailed = buildStatus?.status === 'failed' || generateMutation.isError;

  if (isComplete) {
    return (
      <div className="ml-4 flex flex-col items-end gap-2">
        <Badge variant="success" className="gap-1">
          <CheckCircle className="h-3 w-3" />
          Generated!
        </Badge>
        <p className="text-xs text-slate-400">
          {buildStatus.output_path}
        </p>
      </div>
    );
  }

  if (isFailed) {
    return (
      <div className="ml-4 flex flex-col items-end gap-2">
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Failed
        </Badge>
        <p className="text-xs text-red-400">
          {buildStatus?.error_log?.[0] || generateMutation.error?.message || 'Unknown error'}
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setBuildId(null);
            generateMutation.reset();
          }}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (isBuilding) {
    return (
      <div className="ml-4 flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-purple-400" />
        <span className="text-sm text-slate-400">Generating...</span>
      </div>
    );
  }

  return (
    <Button
      variant="outline"
      size="sm"
      className="ml-4 gap-2"
      onClick={() => generateMutation.mutate()}
      disabled={isBuilding}
    >
      <Zap className="h-4 w-4" />
      Generate Desktop
    </Button>
  );
}

interface RegenerateButtonProps {
  scenarioName: string;
}

function RegenerateButton({ scenarioName }: RegenerateButtonProps) {
  const queryClient = useQueryClient();
  const [showConfirm, setShowConfirm] = useState(false);
  const [buildId, setBuildId] = useState<string | null>(null);

  const regenerateMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(buildUrl('/desktop/generate/quick'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          scenario_name: scenarioName,
          template_type: 'basic'
        })
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to regenerate desktop app');
      }
      return res.json();
    },
    onSuccess: (data) => {
      setBuildId(data.build_id);
      setShowConfirm(false);
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
      }, 3000);
    }
  });

  // Poll build status if we have a buildId
  const { data: buildStatus } = useQuery({
    queryKey: ['regenerate-status', buildId],
    queryFn: async () => {
      if (!buildId) return null;
      const res = await fetch(buildUrl(`/desktop/status/${buildId}`));
      if (!res.ok) throw new Error('Failed to fetch build status');
      return res.json();
    },
    enabled: !!buildId,
    refetchInterval: (data) => {
      if (data?.status === 'ready' || data?.status === 'failed') {
        return false;
      }
      return 2000;
    },
  });

  const isBuilding = regenerateMutation.isPending || (buildStatus && buildStatus.status === 'building');
  const isComplete = buildStatus?.status === 'ready';
  const isFailed = buildStatus?.status === 'failed';

  if (isComplete) {
    return (
      <Badge variant="success" className="gap-1">
        <CheckCircle className="h-3 w-3" />
        Regenerated!
      </Badge>
    );
  }

  if (isFailed) {
    return (
      <div className="flex gap-2">
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Failed
        </Badge>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setBuildId(null);
            regenerateMutation.reset();
            setShowConfirm(true);
          }}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (isBuilding) {
    return (
      <div className="flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-purple-400" />
        <span className="text-sm text-slate-400">Regenerating...</span>
      </div>
    );
  }

  if (showConfirm) {
    return (
      <div className="flex flex-col gap-2 p-3 bg-yellow-950/20 border border-yellow-800/30 rounded">
        <div className="flex items-start gap-2">
          <AlertCircle className="h-4 w-4 text-yellow-400 flex-shrink-0 mt-0.5" />
          <div className="text-xs text-yellow-200">
            <p className="font-semibold">Regenerate desktop app?</p>
            <p className="text-yellow-300/80 mt-1">This will overwrite existing files. Make sure you've saved any custom changes.</p>
          </div>
        </div>
        <div className="flex gap-2 justify-end">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowConfirm(false)}
            disabled={isBuilding}
          >
            Cancel
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={() => regenerateMutation.mutate()}
            disabled={isBuilding}
          >
            <RefreshCw className="h-3 w-3 mr-1" />
            Confirm Regenerate
          </Button>
        </div>
      </div>
    );
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => setShowConfirm(true)}
      disabled={isBuilding}
      className="gap-1"
    >
      <RefreshCw className="h-4 w-4" />
      Regenerate
    </Button>
  );
}

interface DeleteButtonProps {
  scenarioName: string;
}

function DeleteButton({ scenarioName }: DeleteButtonProps) {
  const queryClient = useQueryClient();
  const [showConfirm, setShowConfirm] = useState(false);
  const [confirmText, setConfirmText] = useState("");

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(buildUrl(`/desktop/delete/${scenarioName}`), {
        method: 'DELETE'
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete desktop app');
      }
      return res.json();
    },
    onSuccess: () => {
      setShowConfirm(false);
      setConfirmText("");
      queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
    }
  });

  const isDeleting = deleteMutation.isPending;
  const isSuccess = deleteMutation.isSuccess;
  const isFailed = deleteMutation.isError;

  if (isSuccess) {
    return (
      <Badge variant="default" className="gap-1">
        <CheckCircle className="h-3 w-3" />
        Deleted
      </Badge>
    );
  }

  if (isFailed) {
    return (
      <div className="flex gap-2">
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Failed
        </Badge>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            deleteMutation.reset();
            setShowConfirm(true);
          }}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (isDeleting) {
    return (
      <div className="flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin text-red-400" />
        <span className="text-sm text-slate-400">Deleting...</span>
      </div>
    );
  }

  if (showConfirm) {
    const isConfirmValid = confirmText === scenarioName;

    return (
      <div className="flex flex-col gap-3 p-4 bg-red-950/20 border border-red-800/30 rounded">
        <div className="flex items-start gap-2">
          <AlertCircle className="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1 text-sm text-red-200">
            <p className="font-semibold text-red-300">⚠️ Permanent Deletion</p>
            <p className="text-red-300/90 mt-1">
              This will <strong>permanently delete</strong> all desktop files for <strong>{scenarioName}</strong>.
            </p>
            <p className="text-red-300/80 mt-2 text-xs">
              This includes generated templates, built packages, and all configuration.
            </p>
            <p className="text-red-300/80 mt-2 text-xs font-semibold">
              This action cannot be undone!
            </p>
          </div>
        </div>

        <div className="flex flex-col gap-2">
          <div>
            <label className="text-xs text-red-300 font-medium">
              Type the scenario name "<span className="font-mono font-bold">{scenarioName}</span>" to confirm:
            </label>
            <Input
              type="text"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder={scenarioName}
              className="mt-1 font-mono text-sm"
              autoFocus
            />
          </div>

          <div className="flex gap-2 justify-end">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setShowConfirm(false);
                setConfirmText("");
              }}
              disabled={isDeleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => deleteMutation.mutate()}
              disabled={isDeleting || !isConfirmValid}
            >
              <Trash2 className="h-3 w-3 mr-1" />
              Confirm Delete
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => setShowConfirm(true)}
      disabled={isDeleting}
      className="gap-1 text-red-400 hover:text-red-300 border-red-800/30 hover:border-red-700/50"
    >
      <Trash2 className="h-4 w-4" />
      Delete
    </Button>
  );
}

interface PlatformChipProps {
  platform: string;
  result?: PlatformBuildResult;
  scenarioName: string;
}

function PlatformChip({ platform, result, scenarioName }: PlatformChipProps) {
  const [showError, setShowError] = useState(false);
  const [copied, setCopied] = useState(false);

  const platformIcons: Record<string, string> = {
    win: '🪟',
    mac: '🍎',
    linux: '🐧'
  };

  const platformNames: Record<string, string> = {
    win: 'Windows',
    mac: 'macOS',
    linux: 'Linux'
  };

  const handleDownload = () => {
    const downloadUrl = buildUrl(`/desktop/download/${scenarioName}/${platform}`);
    window.open(downloadUrl, '_blank');
  };

  const handleCopyErrors = async () => {
    if (!result?.error_log) return;
    const errorText = result.error_log.join('\n\n---\n\n');
    await navigator.clipboard.writeText(errorText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Determine chip style based on status
  let chipClass = "flex items-center gap-2 px-3 py-2 rounded-lg border transition-all";
  let icon = null;
  let statusText = "";

  if (!result || result.status === "pending") {
    chipClass += " bg-slate-800 border-slate-600 text-slate-400";
    icon = <div className="h-2 w-2 rounded-full bg-slate-500" />;
    statusText = "Pending";
  } else if (result.status === "building") {
    chipClass += " bg-blue-950/30 border-blue-700 text-blue-300 animate-pulse";
    icon = <Loader2 className="h-3 w-3 animate-spin" />;
    statusText = "Building";
  } else if (result.status === "ready") {
    chipClass += " bg-green-950/30 border-green-700 text-green-300 hover:border-green-600 cursor-pointer";
    icon = <CheckCircle className="h-3 w-3" />;
    statusText = "Ready";
  } else if (result.status === "failed") {
    chipClass += " bg-red-950/30 border-red-700 text-red-300";
    icon = <XCircle className="h-3 w-3" />;
    statusText = "Failed";
  } else if (result.status === "skipped") {
    chipClass += " bg-yellow-950/30 border-yellow-700 text-yellow-300";
    icon = <AlertCircle className="h-3 w-3" />;
    statusText = "Skipped";
  }

  return (
    <div className="flex flex-col gap-2">
      <div
        className={chipClass}
        onClick={result?.status === "ready" ? handleDownload : undefined}
        title={result?.status === "ready" ? "Click to download" : undefined}
      >
        <span className="text-lg">{platformIcons[platform]}</span>
        {icon}
        <div className="flex flex-col gap-0.5">
          <span className="text-xs font-medium">{platformNames[platform]}</span>
          <span className="text-[10px] opacity-80">{statusText}</span>
        </div>
        {result?.file_size && result.status === "ready" && (
          <span className="text-[10px] ml-auto opacity-70">{formatBytes(result.file_size)}</span>
        )}
        {result?.status === "ready" && (
          <FileDown className="h-3 w-3 ml-auto" />
        )}
      </div>

      {/* Show skip reason for skipped platforms */}
      {result?.status === "skipped" && result.skip_reason && (
        <div className="bg-yellow-950/20 border border-yellow-800/30 rounded p-2 text-xs text-yellow-300">
          <div className="flex items-start gap-2">
            <AlertCircle className="h-3 w-3 flex-shrink-0 mt-0.5" />
            <div className="flex-1">{result.skip_reason}</div>
          </div>
        </div>
      )}

      {/* Show error details for failed platforms */}
      {result?.status === "failed" && result.error_log && result.error_log.length > 0 && (
        <div className="flex flex-col gap-1">
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-xs"
              onClick={() => setShowError(!showError)}
            >
              {showError ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
              {showError ? 'Hide' : 'Show'} Error
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-xs"
              onClick={handleCopyErrors}
            >
              {copied ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
              {copied ? 'Copied' : 'Copy'}
            </Button>
          </div>
          {showError && (
            <div className="bg-red-950/20 border border-red-800/30 rounded p-2 text-[10px] font-mono text-red-300 max-h-32 overflow-y-auto">
              {result.error_log.map((error, idx) => (
                <div key={idx} className="whitespace-pre-wrap break-words mb-1 opacity-90">
                  {error}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

interface BuildDesktopButtonProps {
  scenarioName: string;
}

function BuildDesktopButton({ scenarioName }: BuildDesktopButtonProps) {
  const queryClient = useQueryClient();
  const [buildId, setBuildId] = useState<string | null>(null);
  const [showErrors, setShowErrors] = useState(false);
  const [mutationError, setMutationError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const buildMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch(buildUrl(`/desktop/build/${scenarioName}`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          platforms: ['win', 'mac', 'linux'] // Build all platforms
        })
      });
      if (!res.ok) {
        const error = await res.json().catch(() => ({ message: res.statusText }));
        throw new Error(error.message || 'Failed to start build');
      }
      return res.json();
    },
    onSuccess: (data) => {
      setBuildId(data.build_id);
      setMutationError(null);
    },
    onError: (error: Error) => {
      setMutationError(error.message);
      setShowErrors(true);
    }
  });

  // Poll build status if we have a buildId
  const { data: buildStatus } = useQuery({
    queryKey: ['build-status', buildId],
    queryFn: async () => {
      if (!buildId) return null;
      const res = await fetch(buildUrl(`/desktop/status/${buildId}`));
      if (!res.ok) throw new Error('Failed to fetch build status');
      return res.json();
    },
    enabled: !!buildId,
    refetchInterval: (data) => {
      // Stop polling if build is complete
      if (data?.status === 'ready' || data?.status === 'partial' || data?.status === 'failed') {
        return false;
      }
      return 3000; // Poll every 3 seconds during build
    },
  });

  // Refresh scenarios list when build completes
  if ((buildStatus?.status === 'ready' || buildStatus?.status === 'partial') && buildId) {
    queryClient.invalidateQueries({ queryKey: ['scenarios-desktop-status'] });
  }

  const isBuilding = buildMutation.isPending || (buildStatus && buildStatus.status === 'building');
  const isSuccess = buildStatus?.status === 'ready' || buildStatus?.status === 'partial';
  const isFailed = buildStatus?.status === 'failed';

  // Show platform chips when build has results
  if (buildStatus?.platform_results) {
    const platforms = ['win', 'mac', 'linux'];

    return (
      <div className="flex flex-col gap-3 w-full">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          {platforms.map(platform => (
            <PlatformChip
              key={platform}
              platform={platform}
              result={buildStatus.platform_results?.[platform]}
              scenarioName={scenarioName}
            />
          ))}
        </div>

        {/* Show overall status */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {buildStatus.status === 'ready' && (
              <div className="flex items-center gap-1 text-green-400">
                <CheckCircle className="h-4 w-4" />
                <span className="text-sm">All platforms built successfully</span>
              </div>
            )}
            {buildStatus.status === 'partial' && (
              <div className="flex items-center gap-1 text-yellow-400">
                <AlertCircle className="h-4 w-4" />
                <span className="text-sm">Some platforms built successfully</span>
              </div>
            )}
            {buildStatus.status === 'failed' && (
              <div className="flex items-center gap-1 text-red-400">
                <XCircle className="h-4 w-4" />
                <span className="text-sm">Build failed</span>
              </div>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setBuildId(null);
              setShowErrors(false);
              setMutationError(null);
              setCopied(false);
              buildMutation.mutate();
            }}
          >
            Rebuild All
          </Button>
        </div>
      </div>
    );
  }

  // Show mutation error
  if (mutationError) {
    return (
      <div className="flex flex-col gap-2 w-full">
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1 text-red-400">
            <XCircle className="h-4 w-4" />
            <span className="text-sm">Failed to start build</span>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setMutationError(null);
              buildMutation.mutate();
            }}
          >
            Retry
          </Button>
        </div>
        <div className="bg-red-950/20 border border-red-800/30 rounded p-3 text-xs text-red-300">
          {mutationError}
        </div>
      </div>
    );
  }

  if (isBuilding) {
    // Show platform chips with building status
    if (buildStatus?.platform_results) {
      const platforms = ['win', 'mac', 'linux'];
      return (
        <div className="flex flex-col gap-3 w-full">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            {platforms.map(platform => (
              <PlatformChip
                key={platform}
                platform={platform}
                result={buildStatus.platform_results?.[platform]}
                scenarioName={scenarioName}
              />
            ))}
          </div>
          <div className="flex items-center gap-2 text-blue-400">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span className="text-sm">Building platforms...</span>
          </div>
        </div>
      );
    }

    return (
      <div className="flex items-center gap-2 text-blue-400">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-sm">Starting build...</span>
      </div>
    );
  }

  return (
    <Button
      variant="default"
      size="sm"
      className="gap-2"
      onClick={() => buildMutation.mutate()}
    >
      <Hammer className="h-4 w-4" />
      Build Desktop App
    </Button>
  );
}

interface DownloadButtonsProps {
  scenarioName: string;
  platforms?: string[];
}

function DownloadButtons({ scenarioName, platforms = [] }: DownloadButtonsProps) {
  const handleDownload = (platform: string) => {
    const downloadUrl = buildUrl(`/desktop/download/${scenarioName}/${platform}`);
    window.open(downloadUrl, '_blank');
  };

  const platformIcons: Record<string, string> = {
    win: '🪟',
    mac: '🍎',
    linux: '🐧'
  };

  const platformNames: Record<string, string> = {
    win: 'Windows',
    mac: 'macOS',
    linux: 'Linux'
  };

  if (platforms.length === 0) {
    return null;
  }

  return (
    <div className="flex gap-2">
      {platforms.map((platform) => (
        <Button
          key={platform}
          variant="outline"
          size="sm"
          className="gap-2"
          onClick={() => handleDownload(platform)}
        >
          <Download className="h-3 w-3" />
          <span>{platformIcons[platform]}</span>
          {platformNames[platform]}
        </Button>
      ))}
    </div>
  );
}

export function ScenarioInventory() {
  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState<"all" | "desktop" | "web">("all");

  const { data, isLoading, error } = useQuery<ScenariosResponse>({
    queryKey: ['scenarios-desktop-status'],
    queryFn: async () => {
      const res = await fetch(buildUrl('/scenarios/desktop-status'));
      if (!res.ok) throw new Error('Failed to fetch scenarios');
      return res.json();
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-8 w-8 animate-spin text-purple-400" />
        <span className="ml-3 text-slate-400">Loading scenarios...</span>
      </div>
    );
  }

  if (error) {
    return (
      <Card className="border-red-900 bg-red-950/20">
        <CardContent className="p-6">
          <div className="flex items-center gap-3 text-red-400">
            <XCircle className="h-6 w-6" />
            <div>
              <div className="font-semibold">Failed to load scenarios</div>
              <div className="text-sm text-red-300">{(error as Error).message}</div>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  // Filter scenarios based on search and status
  const filteredScenarios = data?.scenarios.filter(scenario => {
    const matchesSearch = scenario.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         (scenario.display_name?.toLowerCase().includes(searchTerm.toLowerCase()));
    const matchesFilter = filterStatus === "all" ||
                         (filterStatus === "desktop" && scenario.has_desktop) ||
                         (filterStatus === "web" && !scenario.has_desktop);
    return matchesSearch && matchesFilter;
  }) || [];

  return (
    <div className="space-y-6">
      {/* Statistics Cards */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <Card className="border-slate-700 bg-slate-800/50">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <Package className="h-8 w-8 text-blue-400" />
              <div>
                <div className="text-2xl font-bold">{data?.stats.total || 0}</div>
                <div className="text-sm text-slate-400">Total Scenarios</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-green-700 bg-green-950/20">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <Monitor className="h-8 w-8 text-green-400" />
              <div>
                <div className="text-2xl font-bold">{data?.stats.with_desktop || 0}</div>
                <div className="text-sm text-slate-400">With Desktop</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-purple-700 bg-purple-950/20">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <Download className="h-8 w-8 text-purple-400" />
              <div>
                <div className="text-2xl font-bold">{data?.stats.built || 0}</div>
                <div className="text-sm text-slate-400">Built Packages</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-slate-700 bg-slate-800/50">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <Package className="h-8 w-8 text-slate-400" />
              <div>
                <div className="text-2xl font-bold">{data?.stats.web_only || 0}</div>
                <div className="text-sm text-slate-400">Web Only</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search and Filter */}
      <Card className="border-slate-700 bg-slate-800/50">
        <CardContent className="p-4">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <Input
                type="text"
                placeholder="Search scenarios..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-10"
              />
            </div>
            <div className="flex gap-2">
              <Button
                variant={filterStatus === "all" ? "default" : "outline"}
                onClick={() => setFilterStatus("all")}
                size="sm"
              >
                All
              </Button>
              <Button
                variant={filterStatus === "desktop" ? "default" : "outline"}
                onClick={() => setFilterStatus("desktop")}
                size="sm"
              >
                Desktop
              </Button>
              <Button
                variant={filterStatus === "web" ? "default" : "outline"}
                onClick={() => setFilterStatus("web")}
                size="sm"
              >
                Web Only
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Scenarios List */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">
            Scenarios ({filteredScenarios.length})
          </h3>
        </div>

        {filteredScenarios.length === 0 ? (
          <Card className="border-slate-700 bg-slate-800/50">
            <CardContent className="p-8 text-center">
              <Package className="mx-auto h-12 w-12 text-slate-600" />
              <p className="mt-3 text-slate-400">No scenarios found matching your filters</p>
            </CardContent>
          </Card>
        ) : (
          filteredScenarios.map((scenario) => (
            <Card
              key={scenario.name}
              className={`border ${
                scenario.has_desktop
                  ? 'border-green-700 bg-green-950/10'
                  : 'border-slate-700 bg-slate-800/50'
              } transition-all hover:border-purple-600`}
            >
              <CardContent className="p-4">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-700">
                        {scenario.has_desktop ? (
                          <Monitor className="h-5 w-5 text-green-400" />
                        ) : (
                          <Package className="h-5 w-5 text-slate-400" />
                        )}
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h4 className="font-semibold">{scenario.name}</h4>
                          {scenario.has_desktop && (
                            <Badge variant="success" className="text-xs">
                              <CheckCircle className="mr-1 h-3 w-3" />
                              Desktop
                            </Badge>
                          )}
                          {scenario.built && (
                            <Badge variant="default" className="text-xs">
                              <Download className="mr-1 h-3 w-3" />
                              Built
                            </Badge>
                          )}
                        </div>
                        {scenario.display_name && (
                          <p className="text-sm text-slate-400">{scenario.display_name}</p>
                        )}
                      </div>
                    </div>

                    {scenario.has_desktop && (
                      <div className="mt-3 ml-13 grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
                        {scenario.version && (
                          <div>
                            <span className="text-slate-500">Version:</span>{" "}
                            <span className="text-slate-300">{scenario.version}</span>
                          </div>
                        )}
                        {scenario.platforms && scenario.platforms.length > 0 && (
                          <div>
                            <span className="text-slate-500">Platforms:</span>{" "}
                            <span className="text-slate-300">{scenario.platforms.join(", ")}</span>
                          </div>
                        )}
                        {scenario.package_size && scenario.package_size > 0 && (
                          <div>
                            <span className="text-slate-500">Size:</span>{" "}
                            <span className="text-slate-300">{formatBytes(scenario.package_size)}</span>
                          </div>
                        )}
                        {scenario.last_modified && (
                          <div>
                            <span className="text-slate-500">Modified:</span>{" "}
                            <span className="text-slate-300">{scenario.last_modified}</span>
                          </div>
                        )}
                      </div>
                    )}

                    {scenario.desktop_path && (
                      <div className="mt-2 ml-13 text-xs text-slate-500">
                        {scenario.desktop_path}
                      </div>
                    )}

                    {/* Action buttons based on state */}
                    {scenario.has_desktop && scenario.built && scenario.platforms && (
                      <div className="mt-3 ml-13">
                        <DownloadButtons
                          scenarioName={scenario.name}
                          platforms={scenario.platforms}
                        />
                      </div>
                    )}

                    {scenario.has_desktop && !scenario.built && (
                      <div className="mt-3 ml-13 flex flex-wrap gap-2">
                        <BuildDesktopButton scenarioName={scenario.name} />
                        <RegenerateButton scenarioName={scenario.name} />
                        <DeleteButton scenarioName={scenario.name} />
                      </div>
                    )}

                    {scenario.has_desktop && scenario.built && (
                      <div className="mt-3 ml-13 flex flex-wrap gap-2">
                        <RegenerateButton scenarioName={scenario.name} />
                        <DeleteButton scenarioName={scenario.name} />
                      </div>
                    )}
                  </div>

                  {!scenario.has_desktop && (
                    <GenerateDesktopButton scenarioName={scenario.name} />
                  )}
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
}
