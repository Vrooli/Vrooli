/**
 * Settings Page - User preferences and system configuration
 *
 * PURPOSE:
 * Allows users to configure UI preferences, execution defaults,
 * workshop behavior, agent settings, review thresholds, and UI behavior.
 *
 * CURRENT STATUS: Persistent via filesystem-backed unified settings API.
 *
 * Related PRD targets: OT-P1-010
 */

import { useContext, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Info } from "lucide-react";
import { UNSAFE_NavigationContext } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { Input } from "../components/ui/input";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { selectors } from "../consts/selectors";
import { applyTheme, defaultQueryOptions } from "../lib";
import { settingsService } from "../services";
import { DEFAULT_SETTINGS } from "../services/settings-service";
import type { Settings } from "../types";

type NavigationContextValue = React.ContextType<typeof UNSAFE_NavigationContext>;
type NavigationBlockerTransaction = { retry: () => void };
type NavigationBlocker = (tx: NavigationBlockerTransaction) => void;
type NavigatorWithBlock = NavigationContextValue["navigator"] & {
  block: (blocker: NavigationBlocker) => () => void;
};

function supportsNavigationBlock(
  navigator: NavigationContextValue["navigator"]
): navigator is NavigatorWithBlock {
  return typeof (navigator as { block?: unknown }).block === "function";
}

function useNavigationBlocker(when: boolean, message: string) {
  const { navigator } = useContext(UNSAFE_NavigationContext);
  const whenRef = useRef(when);
  const messageRef = useRef(message);

  useEffect(() => {
    whenRef.current = when;
  }, [when]);

  useEffect(() => {
    messageRef.current = message;
  }, [message]);

  useEffect(() => {
    if (!supportsNavigationBlock(navigator)) {
      return;
    }

    const unblock = navigator.block((tx) => {
      if (!whenRef.current) {
        tx.retry();
        return;
      }

      if (window.confirm(messageRef.current)) {
        unblock();
        tx.retry();
      }
    });

    return unblock;
  }, [navigator]);
}

function ToggleButtons<T extends string | boolean>({
  value,
  options,
  onChange,
}: {
  value: T;
  options: { value: T; label: string }[];
  onChange: (value: T) => void;
}) {
  return (
    <div className={`mt-2 grid gap-2`} style={{ gridTemplateColumns: `repeat(${options.length}, 1fr)` }}>
      {options.map((opt) => (
        <button
          key={String(opt.value)}
          className={`rounded-lg border py-2 text-sm font-medium ${
            value === opt.value
              ? "border-cyan-500 bg-slate-900 text-cyan-400"
              : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"
          }`}
          onClick={() => onChange(opt.value)}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}

export function SettingsPage() {
  const queryClient = useQueryClient();
  const {
    data: settings,
    isLoading,
    isFetching,
    error,
    refetch,
  } = useQuery<Settings, Error>({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  const [form, setForm] = useState<Settings | null>(null);
  const [savedMessage, setSavedMessage] = useState("");
  const settingsRef = useRef<Settings | null>(null);
  const formRef = useRef<Settings | null>(null);
  const isDirtyRef = useRef(false);

  useEffect(() => {
    if (settings) {
      setForm(settings);
    }
  }, [settings]);

  const updateMutation = useMutation({
    mutationFn: settingsService.update,
    onSuccess: (updated) => {
      queryClient.setQueryData(["settings"], updated);
      setForm(updated);
      setSavedMessage("Settings saved.");
      setTimeout(() => setSavedMessage(""), 3000);
    },
  });

  const isDirty = useMemo(() => {
    if (!settings || !form) return false;
    return JSON.stringify(settings) !== JSON.stringify(form);
  }, [settings, form]);

  useEffect(() => {
    settingsRef.current = settings ?? null;
  }, [settings]);

  useEffect(() => {
    formRef.current = form;
  }, [form]);

  useEffect(() => {
    isDirtyRef.current = isDirty;
  }, [isDirty]);

  useNavigationBlocker(
    isDirty,
    "You have unsaved settings changes. Leave without saving?"
  );

  useEffect(() => {
    if (!isDirty) return;
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [isDirty]);

  useEffect(() => {
    return () => {
      const saved = settingsRef.current;
      const draft = formRef.current;
      if (!isDirtyRef.current || !saved || !draft) return;
      if (draft.theme !== saved.theme) {
        applyTheme(saved.theme);
      }
    };
  }, []);

  const handleThemeChange = (theme: Settings["theme"]) => {
    if (!form) return;
    setForm({ ...form, theme });
    applyTheme(theme);
  };

  const patch = (updates: Partial<Settings>) => {
    if (!form) return;
    setForm({ ...form, ...updates });
  };

  if (isLoading && !settings) {
    return (
      <div className="space-y-6" data-testid={selectors.settings.page}>
        <PageLoadingState
          label="Loading settings..."
          variant="settings"
          testId="settings-loading-state"
        />
      </div>
    );
  }

  if (error && !settings) {
    return (
      <div className="space-y-6" data-testid={selectors.settings.page}>
        <ErrorState
          error={error}
          title="Unable to load settings"
          onRetry={() => {
            void refetch();
          }}
        />
      </div>
    );
  }

  if (!form) return null;

  return (
    <div className="space-y-6" data-testid={selectors.settings.page}>
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Settings</h2>
        {savedMessage ? (
          <div className="flex items-center gap-2 text-sm text-emerald-400">
            <Info className="h-4 w-4" />
            {savedMessage}
          </div>
        ) : isFetching ? (
          <InlineLoadingIndicator
            label="Refreshing settings..."
            testId="settings-refreshing-indicator"
          />
        ) : null}
      </div>

      <Tabs defaultValue="general" data-testid={selectors.settings.settingsTabs}>
        <TabsList className="w-full">
          <TabsTrigger value="general" data-testid={selectors.settings.tabGeneral}>General</TabsTrigger>
          <TabsTrigger value="execution" data-testid={selectors.settings.tabExecution}>Execution</TabsTrigger>
          <TabsTrigger value="workshop" data-testid={selectors.settings.tabWorkshop}>Workshop</TabsTrigger>
          <TabsTrigger value="review" data-testid={selectors.settings.tabReview}>Review</TabsTrigger>
        </TabsList>

        {/* General Tab */}
        <TabsContent value="general" className="space-y-6">
          {/* Theme */}
          <Card data-testid={selectors.settings.themeSettings}>
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-medium text-slate-200">Theme</h3>
                <p className="mt-1 text-sm text-slate-400">Choose your preferred color theme</p>
              </div>
              <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({ theme: DEFAULT_SETTINGS.theme })}>Reset</button>
            </div>
            <div className="mt-4 flex gap-2">
              <button
                className={`flex-1 rounded-lg border py-3 text-sm font-medium ${form.theme === "dark" ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
                data-testid={selectors.settings.themeDark}
                onClick={() => handleThemeChange("dark")}
              >
                Dark
              </button>
              <button
                className={`flex-1 rounded-lg border py-3 text-sm font-medium ${form.theme === "light" ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
                data-testid={selectors.settings.themeLight}
                onClick={() => handleThemeChange("light")}
              >
                Light
              </button>
              <button
                className={`flex-1 rounded-lg border py-3 text-sm font-medium ${form.theme === "system" ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
                data-testid={selectors.settings.themeSystem}
                onClick={() => handleThemeChange("system")}
              >
                System
              </button>
            </div>
          </Card>

          {/* UI Preferences */}
          <Card data-testid={selectors.settings.uiPreferences}>
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-medium text-slate-200">UI Preferences</h3>
                <p className="mt-1 text-sm text-slate-400">Customize the interface behavior.</p>
              </div>
              <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
                searchDebounceMs: DEFAULT_SETTINGS.searchDebounceMs,
                toastDurationMs: DEFAULT_SETTINGS.toastDurationMs,
                confirmDestructiveActions: DEFAULT_SETTINGS.confirmDestructiveActions,
              })}>Reset</button>
            </div>
            <div className="mt-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300">Search Debounce (ms)</label>
                <p className="mt-1 text-xs text-slate-400">Delay before search requests fire while typing (100-2000).</p>
                <Input
                  type="number"
                  min={100}
                  max={2000}
                  className="mt-1"
                  value={form.searchDebounceMs}
                  onChange={(e) => patch({ searchDebounceMs: Math.max(100, Math.min(2000, Number(e.target.value || 100))) })}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300">Toast Duration (seconds)</label>
                <p className="mt-1 text-xs text-slate-400">How long notification messages remain visible (1-30).</p>
                <Input
                  type="number"
                  min={1}
                  max={30}
                  className="mt-1"
                  value={Math.round(form.toastDurationMs / 1000)}
                  onChange={(e) => patch({ toastDurationMs: Math.max(1000, Math.min(30000, Number(e.target.value || 1) * 1000)) })}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Confirm Destructive Actions</label>
                <p className="mt-1 text-xs text-slate-400">Show confirmation dialogs before irreversible operations.</p>
                <ToggleButtons
                  value={form.confirmDestructiveActions}
                  options={[
                    { value: false as const, label: "Disabled" },
                    { value: true as const, label: "Enabled" },
                  ]}
                  onChange={(v) => patch({ confirmDestructiveActions: v })}
                />
              </div>
            </div>
          </Card>
        </TabsContent>

        {/* Execution Tab */}
        <TabsContent value="execution" className="space-y-6">
          {/* Execution Defaults */}
          <Card data-testid={selectors.settings.executionDefaults}>
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-medium text-slate-200">Execution Defaults</h3>
                <p className="mt-1 text-sm text-slate-400">Default mode used when queue requests omit explicit values.</p>
              </div>
              <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
                defaultMode: DEFAULT_SETTINGS.defaultMode,
                autoFixup: DEFAULT_SETTINGS.autoFixup,
                maxFixupAttempts: DEFAULT_SETTINGS.maxFixupAttempts,
              })}>Reset</button>
            </div>
            <div className="mt-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300">Default Mode</label>
                <ToggleButtons
                  value={form.defaultMode}
                  options={[
                    { value: "manual" as const, label: "manual" },
                    { value: "yolo" as const, label: "yolo" },
                  ]}
                  onChange={(v) => patch({ defaultMode: v })}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Auto-Fixup</label>
                <p className="mt-1 text-xs text-slate-400">When enabled, automatically re-runs execution when review finds issues.</p>
                <ToggleButtons
                  value={form.autoFixup}
                  options={[
                    { value: false as const, label: "Disabled" },
                    { value: true as const, label: "Enabled" },
                  ]}
                  onChange={(v) => patch({ autoFixup: v })}
                />
              </div>
              {form.autoFixup && (
                <div>
                  <label className="block text-sm font-medium text-slate-300">Max Fixup Attempts</label>
                  <p className="mt-1 text-xs text-slate-400">Maximum number of automatic fix-up attempts (0-5).</p>
                  <Input
                    type="number"
                    min={0}
                    max={5}
                    className="mt-1"
                    value={form.maxFixupAttempts}
                    onChange={(e) => patch({ maxFixupAttempts: Math.max(0, Math.min(5, Number(e.target.value || 0))) })}
                  />
                </div>
              )}
            </div>
          </Card>

          {/* Governance */}
          <Card>
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-medium text-slate-200">Governance</h3>
                <p className="mt-1 text-sm text-slate-400">Concurrency limits, queue depth, circuit breaker, and cost controls.</p>
              </div>
              <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
                maxConcurrentExecutions: DEFAULT_SETTINGS.maxConcurrentExecutions,
                maxQueueDepth: DEFAULT_SETTINGS.maxQueueDepth,
                circuitBreakerThreshold: DEFAULT_SETTINGS.circuitBreakerThreshold,
                circuitBreakerCooldownMinutes: DEFAULT_SETTINGS.circuitBreakerCooldownMinutes,
                executionCostCapPerRun: DEFAULT_SETTINGS.executionCostCapPerRun,
                costPerTurnEstimate: DEFAULT_SETTINGS.costPerTurnEstimate,
              })}>Reset</button>
            </div>
            <div className="mt-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300">Max Concurrent Executions</label>
                <p className="mt-1 text-xs text-slate-400">Maximum simultaneous agent runs (1-20).</p>
                <Input
                  type="number"
                  min={1}
                  max={20}
                  className="mt-1"
                  value={form.maxConcurrentExecutions}
                  onChange={(e) => patch({ maxConcurrentExecutions: Math.max(1, Math.min(20, Number(e.target.value || 1))) })}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Max Queue Depth</label>
                <p className="mt-1 text-xs text-slate-400">Maximum pending items in queue (0 = unlimited, max 100).</p>
                <Input
                  type="number"
                  min={0}
                  max={100}
                  className="mt-1"
                  value={form.maxQueueDepth}
                  onChange={(e) => patch({ maxQueueDepth: Math.max(0, Math.min(100, Number(e.target.value || 0))) })}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Circuit Breaker Threshold</label>
                <p className="mt-1 text-xs text-slate-400">Consecutive failures before circuit trips (1-10).</p>
                <Input
                  type="number"
                  min={1}
                  max={10}
                  className="mt-1"
                  value={form.circuitBreakerThreshold}
                  onChange={(e) => patch({ circuitBreakerThreshold: Math.max(1, Math.min(10, Number(e.target.value || 1))) })}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Circuit Breaker Cooldown (minutes)</label>
                <p className="mt-1 text-xs text-slate-400">Time before auto-reset after trip (5-1440).</p>
                <Input
                  type="number"
                  min={5}
                  max={1440}
                  className="mt-1"
                  value={form.circuitBreakerCooldownMinutes}
                  onChange={(e) => patch({ circuitBreakerCooldownMinutes: Math.max(5, Math.min(1440, Number(e.target.value || 5))) })}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Cost Cap Per Run ($)</label>
                <p className="mt-1 text-xs text-slate-400">Estimated cost cap per execution (0 = unlimited).</p>
                <Input
                  type="number"
                  min={0}
                  step={0.5}
                  className="mt-1"
                  value={form.executionCostCapPerRun}
                  onChange={(e) => patch({ executionCostCapPerRun: Math.max(0, Number(e.target.value || 0)) })}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Cost Per Turn Estimate ($)</label>
                <p className="mt-1 text-xs text-slate-400">Estimated cost per agent turn for cap calculation (0.00-5.00).</p>
                <Input
                  type="number"
                  min={0}
                  max={5}
                  step={0.01}
                  className="mt-1"
                  value={form.costPerTurnEstimate}
                  onChange={(e) => patch({ costPerTurnEstimate: Math.max(0, Math.min(5, Number(e.target.value || 0))) })}
                />
              </div>
            </div>
          </Card>

          {/* Agent Behavior */}
          <Card data-testid={selectors.settings.agentSettings}>
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-medium text-slate-200">Agent Behavior</h3>
                <p className="mt-1 text-sm text-slate-400">Controls for spawned agent runs.</p>
              </div>
              <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
                agentMaxTurns: DEFAULT_SETTINGS.agentMaxTurns,
                agentTimeoutSeconds: DEFAULT_SETTINGS.agentTimeoutSeconds,
                agentRequiresApproval: DEFAULT_SETTINGS.agentRequiresApproval,
              })}>Reset</button>
            </div>
            <div className="mt-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300">Max Turns</label>
                <p className="mt-1 text-xs text-slate-400">Maximum conversation turns per agent run (5-200).</p>
                <Input
                  type="number"
                  min={5}
                  max={200}
                  className="mt-1"
                  value={form.agentMaxTurns}
                  onChange={(e) => patch({ agentMaxTurns: Math.max(5, Math.min(200, Number(e.target.value || 5))) })}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300">Timeout (minutes)</label>
                <p className="mt-1 text-xs text-slate-400">Maximum wall-clock time before automatic cancellation (1-60).</p>
                <Input
                  type="number"
                  min={1}
                  max={60}
                  className="mt-1"
                  value={Math.round(form.agentTimeoutSeconds / 60)}
                  onChange={(e) => patch({ agentTimeoutSeconds: Math.max(60, Math.min(3600, Number(e.target.value || 1) * 60)) })}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Require Approval</label>
                <p className="mt-1 text-xs text-slate-400">Pause agent runs for human approval before execution.</p>
                <ToggleButtons
                  value={form.agentRequiresApproval}
                  options={[
                    { value: false as const, label: "Disabled" },
                    { value: true as const, label: "Enabled" },
                  ]}
                  onChange={(v) => patch({ agentRequiresApproval: v })}
                />
              </div>
            </div>
          </Card>
        </TabsContent>

        {/* Workshop Tab */}
        <TabsContent value="workshop" className="space-y-6">
          <Card data-testid={selectors.settings.workshopSettings}>
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-medium text-slate-200">Workshop</h3>
                <p className="mt-1 text-sm text-slate-400">Controls for the workshop refinement system.</p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  className="rounded border border-white/10 px-2 py-1 text-xs text-slate-400 hover:border-white/20 hover:text-slate-300"
                  onClick={() => patch({
                    autoInitializeWorkshop: false,
                    autoAdvanceWorkshop: false,
                    autoCascadeWorkshop: false,
                  })}
                >
                  Disable All
                </button>
                <button
                  className="rounded border border-white/10 px-2 py-1 text-xs text-slate-400 hover:border-white/20 hover:text-slate-300"
                  onClick={() => patch({
                    autoInitializeWorkshop: true,
                    autoAdvanceWorkshop: true,
                    autoCascadeWorkshop: true,
                  })}
                >
                  Enable All
                </button>
                <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
                  autoInitializeWorkshop: DEFAULT_SETTINGS.autoInitializeWorkshop,
                  autoAdvanceWorkshop: DEFAULT_SETTINGS.autoAdvanceWorkshop,
                  autoCascadeWorkshop: DEFAULT_SETTINGS.autoCascadeWorkshop,
                  maxAutoRounds: DEFAULT_SETTINGS.maxAutoRounds,
                })}>Reset</button>
              </div>
            </div>
            <div className="mt-4 space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300">Auto-Initialize Workshop</label>
                <p className="mt-1 text-xs text-slate-400">Automatically spawn the first workshop round when a backlog item is created.</p>
                <ToggleButtons
                  value={form.autoInitializeWorkshop}
                  options={[
                    { value: false as const, label: "Disabled" },
                    { value: true as const, label: "Enabled" },
                  ]}
                  onChange={(v) => patch({ autoInitializeWorkshop: v })}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Auto-Advance Workshop</label>
                <p className="mt-1 text-xs text-slate-400">Automatically continue workshop saves into either the next round or a final synthesis pass.</p>
                <ToggleButtons
                  value={form.autoAdvanceWorkshop}
                  options={[
                    { value: false as const, label: "Disabled" },
                    { value: true as const, label: "Enabled" },
                  ]}
                  onChange={(v) => patch({ autoAdvanceWorkshop: v })}
                />
              </div>
              <div
                className="border-t border-white/5 pt-4"
                style={{
                  opacity: form.autoAdvanceWorkshop ? 1 : 0.5,
                  pointerEvents: form.autoAdvanceWorkshop ? "auto" : "none",
                }}
              >
                <label className="block text-sm font-medium text-slate-300">Max Auto Rounds</label>
                <p className="mt-1 text-xs text-slate-400">Maximum workshop rounds before auto-advancement stops (0-50).</p>
                <Input
                  type="number"
                  min={0}
                  max={50}
                  className="mt-1"
                  value={form.maxAutoRounds}
                  onChange={(e) => patch({ maxAutoRounds: Math.max(0, Math.min(50, Number(e.target.value || 0))) })}
                  disabled={!form.autoAdvanceWorkshop}
                />
              </div>
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Auto-Cascade Workshop</label>
                <p className="mt-1 text-xs text-slate-400">Automatically trigger dependent item workshops when a dependency becomes ready.</p>
                <ToggleButtons
                  value={form.autoCascadeWorkshop}
                  options={[
                    { value: false as const, label: "Disabled" },
                    { value: true as const, label: "Enabled" },
                  ]}
                  onChange={(v) => patch({ autoCascadeWorkshop: v })}
                />
              </div>
            </div>
          </Card>
        </TabsContent>

        {/* Review Tab */}
        <TabsContent value="review" className="space-y-6">
          <Card data-testid={selectors.settings.reviewSettings}>
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-lg font-medium text-slate-200">Review Thresholds</h3>
                <p className="mt-1 text-sm text-slate-400">Configure what Git Control Tower considers passing, warning, or failing.</p>
              </div>
              <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
                reviewCodeQualityMinScore: DEFAULT_SETTINGS.reviewCodeQualityMinScore,
                reviewTestMinPassRate: DEFAULT_SETTINGS.reviewTestMinPassRate,
                reviewMaxBlockingViolations: DEFAULT_SETTINGS.reviewMaxBlockingViolations,
                reviewMaxWarnings: DEFAULT_SETTINGS.reviewMaxWarnings,
                reviewRequireScreenshots: DEFAULT_SETTINGS.reviewRequireScreenshots,
                reviewRequireTests: DEFAULT_SETTINGS.reviewRequireTests,
              })}>Reset</button>
            </div>
            <div className="mt-4 space-y-4">
              {/* Code Quality */}
              <div>
                <label className="block text-sm font-medium text-slate-300">Minimum Code Quality Score</label>
                <p className="mt-1 text-xs text-slate-400">Minimum tidiness score required for green status (0-100).</p>
                <Input
                  type="number"
                  min={0}
                  max={100}
                  className="mt-1"
                  value={form.reviewCodeQualityMinScore}
                  onChange={(e) => patch({ reviewCodeQualityMinScore: Math.max(0, Math.min(100, Number(e.target.value || 0))) })}
                />
              </div>

              {/* Tests */}
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Require Tests</label>
                <p className="mt-1 text-xs text-slate-400">Whether tests must exist and pass for green status.</p>
                <ToggleButtons
                  value={form.reviewRequireTests}
                  options={[
                    { value: false as const, label: "Not Required" },
                    { value: true as const, label: "Required" },
                  ]}
                  onChange={(v) => patch({ reviewRequireTests: v })}
                />
              </div>
              {form.reviewRequireTests && (
                <div>
                  <label className="block text-sm font-medium text-slate-300">Minimum Test Pass Rate (%)</label>
                  <p className="mt-1 text-xs text-slate-400">Minimum percentage of tests that must pass for green status (0-100).</p>
                  <Input
                    type="number"
                    min={0}
                    max={100}
                    className="mt-1"
                    value={Math.round(form.reviewTestMinPassRate * 100)}
                    onChange={(e) => patch({ reviewTestMinPassRate: Math.max(0, Math.min(100, Number(e.target.value || 0))) / 100 })}
                  />
                </div>
              )}

              {/* Standards */}
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Max Blocking Violations</label>
                <p className="mt-1 text-xs text-slate-400">Maximum critical/error violations allowed for green status.</p>
                <Input
                  type="number"
                  min={0}
                  className="mt-1"
                  value={form.reviewMaxBlockingViolations}
                  onChange={(e) => patch({ reviewMaxBlockingViolations: Math.max(0, Number(e.target.value || 0)) })}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300">Max Warnings</label>
                <p className="mt-1 text-xs text-slate-400">Maximum warnings before yellow status. Use -1 for unlimited.</p>
                <Input
                  type="number"
                  min={-1}
                  className="mt-1"
                  value={form.reviewMaxWarnings}
                  onChange={(e) => patch({ reviewMaxWarnings: Math.max(-1, Number(e.target.value || -1)) })}
                />
              </div>

              {/* Visual */}
              <div className="border-t border-white/5 pt-4">
                <label className="block text-sm font-medium text-slate-300">Require Screenshots</label>
                <p className="mt-1 text-xs text-slate-400">Whether screenshots are required for green status.</p>
                <ToggleButtons
                  value={form.reviewRequireScreenshots}
                  options={[
                    { value: false as const, label: "Not Required" },
                    { value: true as const, label: "Required" },
                  ]}
                  onChange={(v) => patch({ reviewRequireScreenshots: v })}
                />
              </div>
            </div>
          </Card>
        </TabsContent>
      </Tabs>

      <div className="flex justify-end">
        <Button
          onClick={() => updateMutation.mutate(form)}
          disabled={!isDirty || updateMutation.isPending}
          data-testid={selectors.settings.saveButton}
          className="flex items-center gap-2"
        >
          {updateMutation.isPending ? "Saving..." : "Save Settings"}
        </Button>
      </div>

      {updateMutation.isError && (
        <ErrorState
          error={updateMutation.error as Error}
          title="Failed to save settings"
        />
      )}
    </div>
  );
}
