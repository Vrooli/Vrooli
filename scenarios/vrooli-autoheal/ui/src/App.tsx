// Vrooli Autoheal Dashboard
// [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002] [REQ:UI-EVENTS-001] [REQ:UI-REFRESH-001] [REQ:UI-RESPONSIVE-001]
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertCircle, CheckCircle2, BookOpen, LayoutDashboard, Loader2, Play, RefreshCw, Settings, Shield, TrendingUp } from "lucide-react";
import { Badge, Button, Card } from "./shared/ui/primitives";
import { TabTrigger } from "./shared/ui/composites";
import { APIError, fetchChecks, fetchStatus, groupChecksByStatus, runTick, sortChecksForDisplay, statusToEmoji } from "./lib/api";
import type { CheckInfo } from "./lib/api";
import { selectors } from "./consts/selectors";
import { CheckDetailModal, ErrorDisplay, ReactErrorBoundary, SettingsDialog } from "./shared/components";
import { DashboardSurface, type CollapsedGroups, type EnrichedCheck } from "./surfaces/dashboard";
import { TrendsSurface } from "./surfaces/trends";
import { DocsSurface } from "./surfaces/docs";

const AUTO_REFRESH_INTERVAL = 30000;

type TabType = "dashboard" | "trends" | "docs";
type TickNoticeTone = "info" | "success" | "warning" | "danger";

interface TickNotice {
  tone: TickNoticeTone;
  message: string;
  detail?: string;
}

function getTabFromHash(): TabType {
  const hash = window.location.hash.slice(1);
  if (hash === "trends") return "trends";
  if (hash === "docs" || hash.startsWith("docs?")) return "docs";
  return "dashboard";
}

function loadCollapsedState(): CollapsedGroups {
  try {
    const saved = localStorage.getItem("autoheal-collapsed-groups");
    if (saved) {
      const parsed: unknown = JSON.parse(saved);
      if (
        parsed &&
        typeof parsed === "object" &&
        "critical" in parsed &&
        "warning" in parsed &&
        "ok" in parsed
      ) {
        const candidate = parsed as Record<string, unknown>;
        if (
          typeof candidate.critical === "boolean" &&
          typeof candidate.warning === "boolean" &&
          typeof candidate.ok === "boolean"
        ) {
          return {
            critical: candidate.critical,
            warning: candidate.warning,
            ok: candidate.ok,
          };
        }
      }
    }
  } catch {
    // Ignore parse errors
  }

  return { critical: false, warning: false, ok: true };
}

export default function App() {
  const queryClient = useQueryClient();
  const tabSelectors = selectors.tabs ?? {
    dashboard: "autoheal-tab-dashboard",
    trends: "autoheal-tab-trends",
    docs: "autoheal-tab-docs",
  };
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [activeTab, setActiveTab] = useState<TabType>(getTabFromHash);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [collapsedGroups, setCollapsedGroups] = useState<CollapsedGroups>(loadCollapsedState);
  const [selectedCheckId, setSelectedCheckId] = useState<string | null>(null);
  const [tickNotice, setTickNotice] = useState<TickNotice | null>(null);

  const toggleGroup = useCallback((group: keyof CollapsedGroups) => {
    setCollapsedGroups((prev) => {
      const next = { ...prev, [group]: !prev[group] };
      localStorage.setItem("autoheal-collapsed-groups", JSON.stringify(next));
      return next;
    });
  }, []);

  const handleTabChange = useCallback((tab: TabType) => {
    setActiveTab(tab);
    window.location.hash = tab === "dashboard" ? "" : tab;
  }, []);

  useEffect(() => {
    const handleHashChange = () => {
      setActiveTab(getTabFromHash());
    };
    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["status"],
    queryFn: fetchStatus,
    refetchInterval: (query) => {
      const status = query.state.data;
      if (status?.tickRunning) {
        return 2000;
      }
      return autoRefresh ? AUTO_REFRESH_INTERVAL : false;
    },
  });

  const { data: checksMetadata } = useQuery({
    queryKey: ["checks-metadata"],
    queryFn: fetchChecks,
    staleTime: 60000,
  });

  const checksMetadataMap = useMemo(() => {
    const map: Record<string, CheckInfo> = {};
    if (checksMetadata) {
      for (const check of checksMetadata) {
        map[check.id] = check;
      }
    }
    return map;
  }, [checksMetadata]);

  const tickMutation = useMutation({
    mutationFn: () => runTick(true),
    onSuccess: (response) => {
      setTickNotice({
        tone: "success",
        message: `Health check completed (${response.summary.ok}/${response.summary.total} checks healthy)`,
      });
      queryClient.invalidateQueries({ queryKey: ["status"] });
    },
    onError: (mutationError) => {
      if (mutationError instanceof APIError && mutationError.code === "CONFLICT") {
        setTickNotice({
          tone: "warning",
          message: "A health check cycle is already running.",
          detail: mutationError.recovery.hint ?? mutationError.getSuggestedAction(),
        });
        queryClient.invalidateQueries({ queryKey: ["status"] });
        return;
      }

      if (mutationError instanceof APIError) {
        if (mutationError.statusCode === 502) {
          setTickNotice({
            tone: "warning",
            message: "Tick request hit an upstream gateway error (502).",
            detail: "The health cycle may still be running in the background. The dashboard will keep polling status.",
          });
          queryClient.invalidateQueries({ queryKey: ["status"] });
          return;
        }
        setTickNotice({
          tone: "danger",
          message: mutationError.getUserMessage(),
          detail: mutationError.getSuggestedAction(),
        });
        return;
      }

      const detail = mutationError instanceof Error ? mutationError.message : "Unknown error";
      setTickNotice({
        tone: "danger",
        message: "Failed to run health check cycle.",
        detail,
      });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["status"] });
    },
  });

  useEffect(() => {
    if (!tickNotice) {
      return;
    }
    const timer = window.setTimeout(() => setTickNotice(null), 6000);
    return () => window.clearTimeout(timer);
  }, [tickNotice]);

  const enrichedChecks: EnrichedCheck[] = useMemo(() => {
    const checks = data?.checks || [];
    return checks.map((check) => {
      const metadata = checksMetadataMap[check.checkId];
      return {
        ...check,
        title: metadata?.title,
        description: metadata?.description,
        importance: metadata?.importance,
        category: metadata?.category,
        intervalSeconds: metadata?.intervalSeconds,
      };
    });
  }, [data?.checks, checksMetadataMap]);

  const sortedEnrichedChecks = useMemo(() => sortChecksForDisplay(enrichedChecks), [enrichedChecks]);
  const groupedChecks = useMemo(() => groupChecksByStatus(sortedEnrichedChecks), [sortedEnrichedChecks]);
  const isTickRunning = tickMutation.isPending || Boolean(data?.tickRunning);

  useEffect(() => {
    if (data) {
      const emoji = statusToEmoji(data.status);
      document.title = `${emoji} Autoheal - ${data.status.toUpperCase()}`;
    }
  }, [data]);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-surface-base text-text-primary">
        <div className="text-center">
          <RefreshCw className="mx-auto mb-4 animate-spin" size={32} />
          <p className="text-text-muted">Loading health status...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-surface-base p-6 text-text-primary">
        <Card className="max-w-md p-8">
          <ErrorDisplay
            error={error}
            onRetry={() => refetch()}
            title="Connection Error"
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-surface-base text-text-primary" data-testid={selectors.dashboard}>
      <header className="sticky top-0 z-10 border-b border-border-default/70 bg-surface-elevated/70 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-2 px-4 py-3 sm:gap-3">
          <div className="flex min-w-0 items-center gap-2 sm:gap-3">
            <Shield className="shrink-0 text-accent-primary" size={24} />
            <div className="min-w-0">
              <h1 className="truncate text-base font-semibold sm:text-xl">Vrooli Autoheal</h1>
              <p className="hidden text-xs text-text-muted sm:block">Self-healing infrastructure supervisor</p>
            </div>
          </div>

          <div className="flex shrink-0 items-center gap-2 sm:gap-3">
            <Button
              variant="outline"
              size="icon"
              onClick={() => setSettingsOpen(true)}
              data-testid="settings-button"
              title="Open settings"
              aria-label="Open settings"
            >
              <Settings className="h-4 w-4" />
            </Button>

            <Button
              size="sm"
              onClick={() => tickMutation.mutate()}
              disabled={isTickRunning}
              data-testid={selectors.runTickButton}
              className="px-2 sm:px-4"
              aria-label="Run Tick"
            >
              {isTickRunning ? (
                <Loader2 className="h-4 w-4 animate-spin sm:mr-2" />
              ) : (
                <Play className="h-4 w-4 sm:mr-2" />
              )}
              <span className="hidden sm:inline">Run Tick</span>
            </Button>
            {data?.tickRunning ? <Badge tone="info">Tick Running</Badge> : null}
          </div>
        </div>

        <div className="mx-auto max-w-6xl px-4">
          <nav className="flex gap-1 overflow-x-auto">
            <TabTrigger
              onClick={() => handleTabChange("dashboard")}
              active={activeTab === "dashboard"}
              data-testid={tabSelectors.dashboard}
              className="shrink-0"
            >
              <LayoutDashboard size={16} />
              Dashboard
            </TabTrigger>
            <TabTrigger
              onClick={() => handleTabChange("trends")}
              active={activeTab === "trends"}
              data-testid={tabSelectors.trends}
              className="shrink-0"
            >
              <TrendingUp size={16} />
              Trends
            </TabTrigger>
            <TabTrigger
              onClick={() => handleTabChange("docs")}
              active={activeTab === "docs"}
              data-testid={tabSelectors.docs}
              className="shrink-0"
            >
              <BookOpen size={16} />
              Docs
            </TabTrigger>
          </nav>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-4 sm:py-6">
        {isTickRunning ? (
          <Card className="mb-4 border-accent-primary/40 bg-accent-primary/10 p-3">
            <div className="flex items-start gap-2 text-sm">
              <Loader2 className="mt-0.5 h-4 w-4 animate-spin text-accent-primary" />
              <div>
                <p className="font-medium text-text-primary">Health check cycle is currently running.</p>
                <p className="text-text-muted">
                  This may be from the loop, another user action, or a manual API/CLI tick.
                </p>
              </div>
            </div>
          </Card>
        ) : null}

        {tickNotice ? (
          <Card
            className={`mb-4 p-3 ${
              tickNotice.tone === "success"
                ? "border-accent-success/40 bg-accent-success/10"
                : tickNotice.tone === "warning"
                  ? "border-accent-warning/40 bg-accent-warning/10"
                  : tickNotice.tone === "danger"
                    ? "border-accent-danger/40 bg-accent-danger/10"
                    : "border-accent-primary/40 bg-accent-primary/10"
            }`}
          >
            <div className="flex items-start gap-2 text-sm">
              {tickNotice.tone === "success" ? (
                <CheckCircle2 className="mt-0.5 h-4 w-4 text-accent-success" />
              ) : tickNotice.tone === "danger" ? (
                <AlertCircle className="mt-0.5 h-4 w-4 text-accent-danger" />
              ) : tickNotice.tone === "warning" ? (
                <AlertCircle className="mt-0.5 h-4 w-4 text-accent-warning" />
              ) : (
                <Loader2 className="mt-0.5 h-4 w-4 animate-spin text-accent-primary" />
              )}
              <div>
                <p className="font-medium text-text-primary">{tickNotice.message}</p>
                {tickNotice.detail ? <p className="text-text-muted">{tickNotice.detail}</p> : null}
              </div>
            </div>
          </Card>
        ) : null}

        {activeTab === "dashboard" ? (
          <ReactErrorBoundary sectionName="Dashboard">
            <DashboardSurface
              data={data}
              checksMetadataCount={checksMetadata?.length ?? 0}
              enrichedChecks={enrichedChecks}
              groupedChecks={groupedChecks}
              collapsedGroups={collapsedGroups}
              onToggleGroup={toggleGroup}
              autoRefresh={autoRefresh}
              autoRefreshIntervalSeconds={AUTO_REFRESH_INTERVAL / 1000}
              onShowTrends={() => handleTabChange("trends")}
              onSelectCheck={setSelectedCheckId}
            />
          </ReactErrorBoundary>
        ) : activeTab === "trends" ? (
          <ReactErrorBoundary sectionName="Trends">
            <TrendsSurface />
          </ReactErrorBoundary>
        ) : (
          <ReactErrorBoundary sectionName="Docs">
            <DocsSurface />
          </ReactErrorBoundary>
        )}
      </main>

      <SettingsDialog
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        autoRefresh={autoRefresh}
        onAutoRefreshChange={setAutoRefresh}
      />

      {selectedCheckId && (
        <ReactErrorBoundary sectionName="Check details modal">
          <CheckDetailModal
            checkId={selectedCheckId}
            onClose={() => setSelectedCheckId(null)}
          />
        </ReactErrorBoundary>
      )}
    </div>
  );
}
