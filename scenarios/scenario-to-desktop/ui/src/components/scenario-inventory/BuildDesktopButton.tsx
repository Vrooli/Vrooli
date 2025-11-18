import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { Button } from "../ui/button";
import { Loader2, Hammer, CheckCircle, XCircle, AlertCircle, Check, HelpCircle } from "lucide-react";
import { PlatformChip } from "./PlatformChip";
import { WineInstallDialog } from "../wine";
import { platformIcons, platformNames } from "./utils";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

interface BuildDesktopButtonProps {
  scenarioName: string;
}

interface WineCheckResponse {
  installed: boolean;
  version?: string;
  platform: string;
  required_for: string[];
}

export function BuildDesktopButton({ scenarioName }: BuildDesktopButtonProps) {
  const queryClient = useQueryClient();
  const [buildId, setBuildId] = useState<string | null>(null);
  const [mutationError, setMutationError] = useState<string | null>(null);
  const [showWineDialog, setShowWineDialog] = useState(false);
  const [pendingPlatforms, setPendingPlatforms] = useState<string[]>([]);
  const [platformError, setPlatformError] = useState<string | null>(null);

  const storageKey = useMemo(() => `desktop-platforms-${scenarioName}`, [scenarioName]);
  const defaultPlatforms = ['win', 'mac', 'linux'];
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>(() => {
    if (typeof window === 'undefined') return defaultPlatforms;
    try {
      const stored = window.localStorage.getItem(storageKey);
      if (stored) {
        const parsed = JSON.parse(stored);
        if (Array.isArray(parsed) && parsed.every((item) => typeof item === 'string')) {
          return parsed;
        }
      }
    } catch (error) {
      console.warn('Failed to parse stored platform selection', error);
    }
    return defaultPlatforms;
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;
    window.localStorage.setItem(storageKey, JSON.stringify(selectedPlatforms));
  }, [selectedPlatforms, storageKey]);

  // Check Wine status when building for Windows
  const { data: wineCheck } = useQuery<WineCheckResponse>({
    queryKey: ['wine-check'],
    queryFn: async () => {
      const res = await fetch(buildUrl('/system/wine/check'));
      if (!res.ok) throw new Error('Failed to check Wine status');
      return res.json();
    },
    staleTime: 60000, // Cache for 1 minute
  });

  const buildMutation = useMutation({
    mutationFn: async (platforms: string[]) => {
      const res = await fetch(buildUrl(`/desktop/build/${scenarioName}`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          platforms // Use provided platforms
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
    }
  });

  // Handle build initiation with Wine check
  const handleBuild = (platforms?: string[]) => {
    const targets = platforms && platforms.length ? platforms : selectedPlatforms;
    if (!targets.length) {
      setPlatformError('Select at least one platform to build.');
      return;
    }
    setPlatformError(null);

    // Check if Wine is needed and not installed
    const needsWine = targets.includes('win') && wineCheck?.platform === 'linux';

    if (needsWine && !wineCheck?.installed) {
      // Save platforms for after Wine installation
      setPendingPlatforms(targets);
      setShowWineDialog(true);
      return;
    }

    // Wine not needed or already installed - proceed with build
    buildMutation.mutate(targets);
  };

  // Handle Wine installation complete
  const handleWineInstallComplete = () => {
    setShowWineDialog(false);
    // Invalidate Wine check to get fresh status
    queryClient.invalidateQueries({ queryKey: ['wine-check'] });
    // Proceed with pending build
    if (pendingPlatforms.length > 0) {
      buildMutation.mutate(pendingPlatforms);
      setPendingPlatforms([]);
    }
  };

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

  // Show platform chips when build has results
  if (buildStatus?.platform_results) {
    // Use requested_platforms if available (shows all attempted platforms)
    // Fall back to hardcoded list for backwards compatibility
    const platforms = buildStatus.requested_platforms || ['win', 'mac', 'linux'];

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
              setMutationError(null);
              handleBuild(selectedPlatforms);
            }}
          >
            Rebuild Selected
          </Button>
        </div>
      </div>
    );
  }

  // Show mutation error
  if (mutationError) {
    const copyError = () => {
      navigator.clipboard.writeText(mutationError);
    };

    return (
      <div className="flex flex-col gap-2 w-full">
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1 text-red-400">
            <XCircle className="h-4 w-4" />
            <span className="text-sm">Failed to start build</span>
          </div>
        </div>
        <div className="bg-red-950/20 border border-red-900 rounded p-3">
          <p className="text-xs text-red-300 font-mono whitespace-pre-wrap max-h-32 overflow-y-auto">
            {mutationError}
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={copyError}
            className="gap-1"
          >
            <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            Copy Error
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setMutationError(null);
              handleBuild();
            }}
          >
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (isBuilding) {
    // Show platform chips with building status
    if (buildStatus?.platform_results) {
      // Use requested_platforms if available
      const platforms = buildStatus.requested_platforms || ['win', 'mac', 'linux'];
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

  const togglePlatform = (platform: string) => {
    setSelectedPlatforms((prev) =>
      prev.includes(platform) ? prev.filter((value) => value !== platform) : [...prev, platform]
    );
  };

  const platformOptions = [
    { id: 'win', label: 'Windows (.exe)', helper: 'Most laptops and desktops' },
    { id: 'mac', label: 'macOS (.dmg)', helper: 'MacBook + iMac' },
    { id: 'linux', label: 'Linux (.AppImage)', helper: 'Ubuntu, Fedora, etc.' },
  ];

  const selectionSummary = (() => {
    if (selectedPlatforms.length === platformOptions.length) return 'Building installers for every platform.';
    if (selectedPlatforms.length === 0) return 'Select at least one platform to get started.';
    return `Building ${selectedPlatforms.map((platform) => platformNames[platform]).join(' + ')}.`;
  })();

  return (
    <>
      {showWineDialog && (
        <WineInstallDialog
          onClose={() => {
            setShowWineDialog(false);
            setPendingPlatforms([]);
          }}
          onInstallComplete={handleWineInstallComplete}
        />
      )}
      <div className="space-y-4">
        <div className="space-y-2">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Which installers do you need?</p>
          <p className="text-xs text-slate-400">{selectionSummary}</p>
          <div className="grid gap-2 md:grid-cols-3">
            {platformOptions.map(({ id, label, helper }) => {
              const active = selectedPlatforms.includes(id);
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => togglePlatform(id)}
                  className={`flex flex-col gap-1 rounded-xl border p-3 text-left transition focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                    active ? 'border-blue-400 bg-blue-950/40 shadow-inner' : 'border-slate-700 bg-slate-900/40'
                  }`}
                  aria-pressed={active}
                >
                  <div className="flex items-center justify-between text-sm font-semibold text-slate-100">
                    <div className="flex items-center gap-2">
                      <span>{platformIcons[id]}</span>
                      {label}
                    </div>
                    {active && <Check className="h-4 w-4 text-green-400" />}
                  </div>
                  <p className="text-[11px] text-slate-400">{helper}</p>
                </button>
              );
            })}
          </div>
          <div className="flex items-center gap-1 text-[11px] text-slate-500">
            <HelpCircle className="h-3 w-3" />
            Need help choosing?{' '}
            <a
              href="https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
              target="_blank"
              rel="noreferrer"
              className="text-blue-300 underline"
            >
              Read the desktop tier guide
            </a>
          </div>
        </div>
        {platformError && <p className="text-xs text-red-300">{platformError}</p>}
        <Button
          variant="default"
          size="sm"
          className="gap-2"
          onClick={() => handleBuild(selectedPlatforms)}
        >
          <Hammer className="h-4 w-4" />
          Build selected installers
        </Button>
      </div>
    </>
  );
}
