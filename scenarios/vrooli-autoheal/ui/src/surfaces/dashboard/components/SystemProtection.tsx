// System Protection status component
// [REQ:WATCH-DETECT-001] [REQ:WATCH-INSTALL-001] [REQ:UI-HEALTH-001]
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  Shield,
  CheckCircle,
  AlertTriangle,
  XCircle,
  ChevronDown,
  ChevronUp,
  ChevronRight,
  Copy,
  Check,
  RefreshCw,
  ExternalLink,
  Terminal,
  Download,
  Trash2,
  Loader2,
} from "lucide-react";
import {
  fetchWatchdogStatus,
  fetchWatchdogTemplate,
  installWatchdog,
  uninstallWatchdog,
  type ProtectionLevel,
  type WatchdogStatus,
  type InstallOptions,
} from "../../../lib/api";
import { ErrorDisplay } from "../../../shared/components";
import { Button, Card } from "../../../shared/ui/primitives";
import { Notice, NoticeTitle } from "../../../shared/ui/composites";
import { cn } from "../../../lib/utils";
import { selectors } from "../../../consts/selectors";
import { getDocsPath } from "../../../lib/docs";

const PROTECTION_CONFIG: Record<
  ProtectionLevel,
  { textClass: string; bgClass: string; label: string; icon: typeof CheckCircle }
> = {
  full: {
    textClass: "text-accent-success",
    bgClass: "bg-accent-success/20",
    label: "Full Protection",
    icon: CheckCircle,
  },
  partial: {
    textClass: "text-accent-warning",
    bgClass: "bg-accent-warning/20",
    label: "Partial Protection",
    icon: AlertTriangle,
  },
  none: {
    textClass: "text-accent-danger",
    bgClass: "bg-accent-danger/20",
    label: "Not Protected",
    icon: XCircle,
  },
};

function StatusIndicator({ active, label }: { active: boolean; label: string }) {
  return (
    <div className="flex items-center justify-between py-1.5">
      <span className="text-sm text-text-muted">{label}</span>
      <span className={cn("flex items-center gap-1.5 text-sm", active ? "text-accent-success" : "text-text-muted/70")}>
        <span className={cn("h-2 w-2 rounded-full", active ? "bg-accent-success" : "bg-border-strong")} />
        {active ? "Active" : "Inactive"}
      </span>
    </div>
  );
}

function LingeringWarning({ username }: { username: string }) {
  const [copied, setCopied] = useState(false);
  const command = `sudo loginctl enable-linger ${username}`;

  const handleCopy = async () => {
    await navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Notice tone="warning" className="mt-3">
      <div className="flex items-start gap-2">
        <Terminal size={16} className="mt-0.5 shrink-0 text-accent-warning" />
        <div className="min-w-0 flex-1">
          <NoticeTitle tone="warning" className="mb-1">
            Headless Boot Required
          </NoticeTitle>
          <p className="mb-2 text-xs text-text-muted">
            Your service won&apos;t start at boot without a login session. Enable lingering to fix this:
          </p>
          <div className="group relative">
            <code className="block overflow-x-auto rounded bg-surface-base p-2 pr-10 font-mono text-xs text-accent-success">
              {command}
            </code>
            <Button
              onClick={handleCopy}
              size="icon"
              variant="outline"
              className="absolute right-1.5 top-1.5 h-6 w-6 p-0"
              title="Copy command"
            >
              {copied ? <Check size={12} className="text-accent-success" /> : <Copy size={12} className="text-text-muted" />}
            </Button>
          </div>
        </div>
      </div>
    </Notice>
  );
}

function OneClickInstall({ onClose, onInstalled }: { onClose: () => void; onInstalled: () => void }) {
  const [copiedOneLiner, setCopiedOneLiner] = useState(false);
  const [copiedTemplate, setCopiedTemplate] = useState(false);
  const [showManual, setShowManual] = useState(false);
  const [installError, setInstallError] = useState<string | null>(null);
  const [installSuccess, setInstallSuccess] = useState<string | null>(null);

  const queryClient = useQueryClient();

  const { data: templateData, isLoading: templateLoading } = useQuery({
    queryKey: ["watchdog-template"],
    queryFn: fetchWatchdogTemplate,
  });

  const installMutation = useMutation({
    mutationFn: (opts: InstallOptions) => installWatchdog(opts),
    onSuccess: (result) => {
      if (result.success) {
        setInstallSuccess(result.message);
        setInstallError(null);
        queryClient.invalidateQueries({ queryKey: ["watchdog"] });
        onInstalled();
      } else {
        setInstallError(result.error || result.message);
        setInstallSuccess(null);
      }
    },
    onError: (error: Error) => {
      setInstallError(error.message);
      setInstallSuccess(null);
    },
  });

  const handleOneClickInstall = (useSystemService: boolean) => {
    setInstallError(null);
    setInstallSuccess(null);
    installMutation.mutate({
      useSystemService,
      enableLingering: true,
    });
  };

  const handleCopyOneLiner = async () => {
    if (!templateData?.oneLiner) return;
    await navigator.clipboard.writeText(templateData.oneLiner);
    setCopiedOneLiner(true);
    setTimeout(() => setCopiedOneLiner(false), 2000);
  };

  const handleCopyTemplate = async () => {
    if (!templateData?.template) return;
    await navigator.clipboard.writeText(templateData.template);
    setCopiedTemplate(true);
    setTimeout(() => setCopiedTemplate(false), 2000);
  };

  const isInstalling = installMutation.isPending;

  return (
    <div className="mt-3 space-y-3 border-t border-border-default/40 pt-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-text-primary">Install Watchdog</h4>
        <Button onClick={onClose} variant="outline" size="sm" disabled={isInstalling}>
          Close
        </Button>
      </div>

      {installSuccess && (
        <Notice tone="success" className="p-2">
          <div className="flex items-center gap-2">
            <CheckCircle size={14} className="text-accent-success" />
            <p className="text-sm text-accent-success">{installSuccess}</p>
          </div>
        </Notice>
      )}

      {installError && (
        <Notice tone="danger" className="p-2">
          <p className="text-sm text-accent-danger">{installError}</p>
        </Notice>
      )}

      <div className="space-y-2">
        <p className="text-xs text-text-muted">Choose installation type:</p>
        <div className="flex gap-2">
          <Button
            onClick={() => handleOneClickInstall(false)}
            disabled={isInstalling}
            variant="outline"
            className="flex-1 border-accent-primary/30 bg-accent-primary/10 text-accent-primary hover:bg-accent-primary/20"
          >
            {isInstalling ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
            <span className="ml-2">User Service</span>
          </Button>
          <Button
            onClick={() => handleOneClickInstall(true)}
            disabled={isInstalling}
            variant="outline"
            className="flex-1 border-accent-warning/30 bg-accent-warning/10 text-accent-warning hover:bg-accent-warning/20"
            title="Requires sudo/admin"
          >
            {isInstalling ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
            <span className="ml-2">System Service</span>
          </Button>
        </div>
        <p className="text-xs text-text-muted/80">User service is recommended. System service requires sudo/admin.</p>
      </div>

      <Button onClick={() => setShowManual(!showManual)} variant="outline" size="sm" className="w-fit">
        {showManual ? "Hide" : "Show"} manual installation
        <ChevronRight size={12} className={cn("ml-1 transition-transform", showManual ? "rotate-90" : "")} />
      </Button>

      {showManual && templateData && (
        <div className="space-y-3 border-l-2 border-border-default/60 pl-2">
          {templateData.oneLiner && (
            <div className="space-y-1">
              <p className="text-xs text-text-muted">
                Or run this command (requires <code className="text-accent-primary">jq</code>):
              </p>
              <div className="group relative">
                <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded bg-surface-base p-2 pr-10 font-mono text-xs text-accent-success">
                  {templateData.oneLiner}
                </pre>
                <Button
                  onClick={handleCopyOneLiner}
                  size="icon"
                  variant="outline"
                  className="absolute right-2 top-2 h-7 w-7 p-0"
                  title="Copy command"
                >
                  {copiedOneLiner ? <Check size={14} className="text-accent-success" /> : <Copy size={14} className="text-text-muted" />}
                </Button>
              </div>
            </div>
          )}

          <div className="space-y-1 text-xs text-text-muted">
            {templateData.instructions.split("\n").map((line, i) => (
              <p key={i}>{line}</p>
            ))}
          </div>

          <div className="relative">
            <pre className="max-h-32 overflow-x-auto rounded bg-surface-base p-2 font-mono text-xs text-text-muted">
              {templateData.template.slice(0, 500)}
              {templateData.template.length > 500 ? "..." : ""}
            </pre>
            <Button
              onClick={handleCopyTemplate}
              size="icon"
              variant="outline"
              className="absolute right-2 top-2 h-7 w-7 p-0"
              title="Copy template"
            >
              {copiedTemplate ? <Check size={14} className="text-accent-success" /> : <Copy size={14} className="text-text-muted" />}
            </Button>
          </div>
        </div>
      )}

      {templateLoading && <p className="text-xs text-text-muted/80">Loading template...</p>}
    </div>
  );
}

function UninstallPanel({ onClose, onUninstalled }: { onClose: () => void; onUninstalled: () => void }) {
  const [error, setError] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const uninstallMutation = useMutation({
    mutationFn: uninstallWatchdog,
    onSuccess: (result) => {
      if (result.success) {
        queryClient.invalidateQueries({ queryKey: ["watchdog"] });
        onUninstalled();
        onClose();
      } else {
        setError(result.error || result.message);
      }
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  return (
    <div className="mt-3 space-y-3 border-t border-border-default/40 pt-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-accent-danger">Uninstall Watchdog</h4>
        <Button onClick={onClose} variant="outline" size="sm" disabled={uninstallMutation.isPending}>
          Cancel
        </Button>
      </div>

      <p className="text-xs text-text-muted">
        This will remove boot protection. The autoheal loop will continue running but won&apos;t auto-start after a
        reboot.
      </p>

      {error && (
        <Notice tone="danger" className="p-2">
          <p className="text-sm text-accent-danger">{error}</p>
        </Notice>
      )}

      <Button
        onClick={() => uninstallMutation.mutate()}
        disabled={uninstallMutation.isPending}
        variant="outline"
        className="w-full border-accent-danger/30 bg-accent-danger/10 text-accent-danger hover:bg-accent-danger/20"
      >
        {uninstallMutation.isPending ? <Loader2 size={14} className="animate-spin" /> : <Trash2 size={14} />}
        <span className="ml-2">Confirm Uninstall</span>
      </Button>
    </div>
  );
}

interface SystemProtectionProps {
  compact?: boolean;
}

export function SystemProtection({ compact = false }: SystemProtectionProps) {
  const [showInstall, setShowInstall] = useState(false);
  const [showUninstall, setShowUninstall] = useState(false);

  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["watchdog"],
    queryFn: () => fetchWatchdogStatus(false),
    staleTime: 60000,
    refetchInterval: 120000,
  });

  const handleInstalled = () => {
    setTimeout(() => refetch(), 1000);
  };

  const handleUninstalled = () => {
    setShowUninstall(false);
    setTimeout(() => refetch(), 1000);
  };

  if (isLoading) {
    return (
      <Card className="p-4" data-testid={selectors.systemProtection}>
        <div className="mb-3 flex items-center gap-2">
          <Shield size={18} className="text-accent-primary" />
          <h3 className="text-sm font-medium">System Protection</h3>
        </div>
        <p className="text-sm text-text-muted">Loading...</p>
      </Card>
    );
  }

  if (error || !data) {
    return (
      <Card className="p-4" data-testid={selectors.systemProtection}>
        <div className="mb-3 flex items-center gap-2">
          <Shield size={18} className="text-accent-primary" />
          <h3 className="text-sm font-medium">System Protection</h3>
        </div>
        <ErrorDisplay error={error} onRetry={() => refetch()} compact />
      </Card>
    );
  }

  const config = PROTECTION_CONFIG[data.protectionLevel];
  const Icon = config.icon;

  if (compact) {
    return (
      <div
        className={cn("flex items-center gap-1.5 rounded px-2 py-1", config.bgClass)}
        title={config.label}
        data-testid={selectors.systemProtectionCompact}
      >
        <Icon size={14} className={config.textClass} />
      </div>
    );
  }

  return (
    <Card className="p-4" data-testid={selectors.systemProtection}>
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Shield size={18} className="text-accent-primary" />
          <h3 className="text-sm font-medium">System Protection</h3>
        </div>
        <Button onClick={() => refetch()} disabled={isFetching} variant="outline" size="icon" title="Refresh status">
          <RefreshCw size={14} className={cn("text-text-muted", isFetching ? "animate-spin" : "")} />
        </Button>
      </div>

      <div className={cn("mb-3 flex items-center gap-2 rounded-lg p-2", config.bgClass)}>
        <Icon size={18} className={config.textClass} />
        <span className={cn("text-sm font-medium", config.textClass)}>{config.label}</span>
      </div>

      <div className="space-y-0.5">
        <StatusIndicator active={data.loopRunning} label="Autoheal Loop" />
        <StatusIndicator active={data.watchdogInstalled} label="OS Watchdog" />
        <StatusIndicator active={data.bootProtectionActive} label="Boot Recovery" />
      </div>

      {data.watchdogType === "systemd" &&
        data.watchdogInstalled &&
        data.isUserService &&
        !data.lingeringEnabled &&
        data.username && <LingeringWarning username={data.username} />}

      <a
        href={`#docs?path=${encodeURIComponent(getDocsPath("system-protection"))}`}
        className="mt-3 flex items-center gap-1.5 text-xs text-accent-primary transition-colors hover:text-accent-primary/80"
      >
        <ExternalLink size={12} />
        Learn more about system protection
      </a>

      {data.watchdogType && (
        <p className="mt-2 text-xs text-text-muted">
          Type: <span className="text-text-primary/90">{data.watchdogType}</span>
          {data.servicePath && (
            <span className="block truncate" title={data.servicePath}>
              Path: {data.servicePath}
            </span>
          )}
        </p>
      )}

      {data.lastError && <p className="mt-2 text-xs text-accent-warning">{data.lastError}</p>}

      {data.canInstall && (
        <div className="mt-3 flex gap-2">
          {!data.watchdogInstalled ? (
            <Button
              onClick={() => {
                setShowInstall(!showInstall);
                setShowUninstall(false);
              }}
              variant="outline"
              className="flex-1 justify-between border-accent-primary/30 bg-accent-primary/10 text-accent-primary hover:bg-accent-primary/20"
            >
              <span>Install Watchdog</span>
              {showInstall ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
            </Button>
          ) : (
            <>
              <Button
                onClick={() => {
                  setShowInstall(!showInstall);
                  setShowUninstall(false);
                }}
                variant="outline"
                className="flex-1 justify-between border-accent-success/30 bg-accent-success/10 text-accent-success hover:bg-accent-success/20"
              >
                <span>Reinstall</span>
                {showInstall ? <ChevronUp size={16} /> : <Download size={16} />}
              </Button>
              <Button
                onClick={() => {
                  setShowUninstall(!showUninstall);
                  setShowInstall(false);
                }}
                variant="outline"
                className="border-accent-danger/30 bg-accent-danger/10 px-3 text-accent-danger hover:bg-accent-danger/20"
                title="Uninstall watchdog"
              >
                <Trash2 size={16} />
              </Button>
            </>
          )}
        </div>
      )}

      {showInstall && <OneClickInstall onClose={() => setShowInstall(false)} onInstalled={handleInstalled} />}

      {showUninstall && <UninstallPanel onClose={() => setShowUninstall(false)} onUninstalled={handleUninstalled} />}

      {!data.canInstall && <p className="mt-3 text-xs text-text-muted/80">OS watchdog not available on this platform</p>}
    </Card>
  );
}

export function useProtectionStatus(): { status: WatchdogStatus | undefined; isLoading: boolean } {
  const { data, isLoading } = useQuery({
    queryKey: ["watchdog"],
    queryFn: () => fetchWatchdogStatus(false),
    staleTime: 60000,
  });

  return { status: data, isLoading };
}
