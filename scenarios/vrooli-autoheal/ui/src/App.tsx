// Vrooli Autoheal Dashboard
// [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002] [REQ:UI-EVENTS-001] [REQ:UI-REFRESH-001] [REQ:UI-RESPONSIVE-001]
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Suspense, lazy, useCallback, useEffect, useMemo, useState } from "react";
import { AlertCircle, CheckCircle2, Loader2, RefreshCw } from "lucide-react";
import { Card } from "./shared/ui/primitives";
import { AppShell, Notice, NoticeTitle } from "./shared/ui/composites";
import { APIError, fetchStatus, groupChecksByStatus, runTick, sortChecksForDisplay, statusToEmoji } from "./lib/api";
import { CheckDetailModal, ErrorDisplay, ReactErrorBoundary, SettingsDialog } from "./shared/components";
import { useCheckMetadata } from "./shared/contexts/CheckMetadataContext";
import { DashboardSurface, type CollapsedGroups, type EnrichedCheck } from "./surfaces/dashboard";
import { useActiveTab } from "./hooks/useActiveTab";
import { useTickNotice, type TickNoticeTone } from "./hooks/useTickNotice";

const TrendsSurface = lazy(async () => import("./surfaces/trends").then((module) => ({ default: module.TrendsSurface })));
const IncidentsSurface = lazy(async () => import("./surfaces/incidents").then((module) => ({ default: module.IncidentsSurface })));
const TimelineSurface = lazy(async () => import("./surfaces/timeline").then((module) => ({ default: module.TimelineSurface })));
const DocsSurface = lazy(async () => import("./surfaces/docs").then((module) => ({ default: module.DocsSurface })));

const AUTO_REFRESH_INTERVAL = 30000;
const EMPTY_METADATA_MAP = new Map<string, never>();

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
        "ok" in parsed &&
        "notApplicable" in parsed
      ) {
        const candidate = parsed as Record<string, unknown>;
        if (
          typeof candidate.critical === "boolean" &&
          typeof candidate.warning === "boolean" &&
          typeof candidate.ok === "boolean" &&
          typeof candidate.notApplicable === "boolean"
        ) {
          return {
            critical: candidate.critical,
            warning: candidate.warning,
            ok: candidate.ok,
            notApplicable: candidate.notApplicable,
          };
        }
      }
    }
  } catch {
    // Ignore parse errors
  }

  return { critical: false, warning: false, ok: true, notApplicable: true };
}

export default function App() {
  const queryClient = useQueryClient();
  const [autoRefresh, setAutoRefresh] = useState(true);
  const { activeTab, handleTabChange } = useActiveTab();
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [collapsedGroups, setCollapsedGroups] = useState<CollapsedGroups>(loadCollapsedState);
  const [selectedCheckId, setSelectedCheckId] = useState<string | null>(null);
  const { tickNotice, setTickNotice } = useTickNotice();
  const metadataContext = useCheckMetadata();
  const checksMetadata = metadataContext.checks ?? [];
  const checksMetadataMap = metadataContext.metadataMap ?? EMPTY_METADATA_MAP;

  const toggleGroup = useCallback((group: keyof CollapsedGroups) => {
    setCollapsedGroups((prev) => {
      const next = { ...prev, [group]: !prev[group] };
      localStorage.setItem("autoheal-collapsed-groups", JSON.stringify(next));
      return next;
    });
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

  const enrichedChecks: EnrichedCheck[] = useMemo(() => {
    const checks = data?.checks || [];
    const autoHealIssues = data?.autoHealIssues ?? {};
    return checks.map((check) => {
      const metadata = checksMetadataMap.get(check.checkId);
      return {
        ...check,
        title: metadata?.title,
        description: metadata?.description,
        importance: metadata?.importance,
        category: metadata?.category,
        intervalSeconds: metadata?.intervalSeconds,
        autoHealIssue: autoHealIssues[check.checkId],
      };
    });
  }, [data?.checks, data?.autoHealIssues, checksMetadataMap]);

  const sortedEnrichedChecks = useMemo(() => sortChecksForDisplay(enrichedChecks), [enrichedChecks]);
  const groupedChecks = useMemo(() => groupChecksByStatus(sortedEnrichedChecks), [sortedEnrichedChecks]);
  const isTickRunning = tickMutation.isPending || Boolean(data?.tickRunning);
  const tabLoadingFallback = (
    <Card className="flex min-h-[16rem] items-center justify-center p-6">
      <div className="text-center">
        <Loader2 className="mx-auto mb-3 h-5 w-5 animate-spin text-accent-primary" />
        <p className="text-sm text-text-muted">Loading tab content...</p>
      </div>
    </Card>
  );

  useEffect(() => {
    if (data) {
      const emoji = statusToEmoji(data.status);
      document.title = `${emoji} Autoheal - ${data.status.toUpperCase()}`;
    }
  }, [data]);

  if (isLoading) {
    return (
      <div className="flex min-h-full items-center justify-center bg-surface-base text-text-primary">
        <div className="text-center">
          <RefreshCw className="mx-auto mb-4 animate-spin" size={32} />
          <p className="text-text-muted">Loading health status...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex min-h-full items-center justify-center bg-surface-base p-6 text-text-primary">
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
    <AppShell
      activeTab={activeTab}
      isTickRunning={isTickRunning}
      onOpenSettings={() => setSettingsOpen(true)}
      onRunTick={() => tickMutation.mutate()}
      onTabChange={handleTabChange}
    >
        {isTickRunning ? (
          <Notice tone="info" className="mb-3 sm:mb-4">
            <div className="flex items-start gap-2 text-sm">
              <Loader2 className="mt-0.5 h-4 w-4 animate-spin text-accent-primary" />
              <div>
                <NoticeTitle tone="neutral">Health check cycle is currently running.</NoticeTitle>
                <p className="text-text-muted">
                  This may be from the loop, another user action, or a manual API/CLI tick.
                </p>
              </div>
            </div>
          </Notice>
        ) : null}

        {tickNotice ? (
          <Notice tone={tickNotice.tone} className="mb-3 sm:mb-4">
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
                <NoticeTitle tone={noticeTitleTone(tickNotice.tone)}>{tickNotice.message}</NoticeTitle>
                {tickNotice.detail ? <p className="text-text-muted">{tickNotice.detail}</p> : null}
              </div>
            </div>
          </Notice>
        ) : null}

		{data?.autoHealSkips?.length ? (
			<Notice tone="warning" className="mb-3 sm:mb-4" role="status">
				<div className="text-sm">
					<NoticeTitle tone="warning">
						Auto-heal decisions recorded without execution ({data.autoHealSkips.length})
					</NoticeTitle>
					<p className="text-text-muted">
						{data.autoHealSkips[0]?.message || "Review the action history for the recorded policy or cooldown reason."}
					</p>
				</div>
			</Notice>
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
            <Suspense fallback={tabLoadingFallback}>
              <TrendsSurface />
            </Suspense>
          </ReactErrorBoundary>
        ) : activeTab === "timeline" ? (
          <ReactErrorBoundary sectionName="Timeline">
            <Suspense fallback={tabLoadingFallback}>
              <TimelineSurface />
            </Suspense>
          </ReactErrorBoundary>
        ) : activeTab === "incidents" ? (
          <ReactErrorBoundary sectionName="Incidents">
            <Suspense fallback={tabLoadingFallback}>
              <IncidentsSurface />
            </Suspense>
          </ReactErrorBoundary>
        ) : (
          <ReactErrorBoundary sectionName="Docs">
            <Suspense fallback={tabLoadingFallback}>
              <DocsSurface />
            </Suspense>
          </ReactErrorBoundary>
        )}

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
    </AppShell>
  );
}

function noticeTitleTone(tone: TickNoticeTone) {
  return tone === "info" ? "neutral" : tone;
}
