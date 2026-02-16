/**
 * Settings Page - User preferences and system configuration
 *
 * PURPOSE:
 * Allows users to configure UI preferences and system behavior.
 *
 * CURRENT STATUS: Persistent via filesystem-backed settings API.
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
import { InlineLoadingIndicator, PageLoadingState, PanelLoadingState } from "../components/ui/loading-states";
import { selectors } from "../consts/selectors";
import { applyTheme, defaultQueryOptions } from "../lib";
import { executionPolicyService, settingsService } from "../services";
import type { Settings } from "../types";
import type { ExecutionPolicy } from "../types";

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
  // BrowserRouter doesn't support useBlocker; use history.block when available.
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

export function SettingsPage() {
  const queryClient = useQueryClient();
  const {
    data: settings,
    isLoading,
    isFetching: isSettingsFetching,
    error,
    refetch,
  } = useQuery<Settings, Error>({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });
  const {
    data: executionPolicy,
    isLoading: isPolicyLoading,
    isFetching: isPolicyFetching,
    error: policyError,
    refetch: refetchPolicy,
  } = useQuery<ExecutionPolicy, Error>({
    queryKey: ["execution-policy"],
    queryFn: () => executionPolicyService.get(),
    ...defaultQueryOptions,
  });

  const [form, setForm] = useState<Settings | null>(null);
  const [policyForm, setPolicyForm] = useState<ExecutionPolicy | null>(null);
  const [savedMessage, setSavedMessage] = useState("");
  const settingsRef = useRef<Settings | null>(null);
  const formRef = useRef<Settings | null>(null);
  const isDirtyRef = useRef(false);

  useEffect(() => {
    if (settings) {
      setForm(settings);
    }
  }, [settings]);
  useEffect(() => {
    if (executionPolicy) {
      setPolicyForm(executionPolicy);
    }
  }, [executionPolicy]);

  const updateMutation = useMutation({
    mutationFn: settingsService.update,
    onSuccess: (updated) => {
      queryClient.setQueryData(["settings"], updated);
      setForm(updated);
      setSavedMessage("Settings saved.");
      setTimeout(() => setSavedMessage(""), 3000);
    },
  });
  const updatePolicyMutation = useMutation({
    mutationFn: executionPolicyService.update,
    onSuccess: (updated) => {
      queryClient.setQueryData(["execution-policy"], updated);
      setPolicyForm(updated);
      setSavedMessage("Settings saved.");
      setTimeout(() => setSavedMessage(""), 3000);
    },
  });

  const isDirty = useMemo(() => {
    if (!settings || !form) return false;
    return JSON.stringify(settings) !== JSON.stringify(form);
  }, [settings, form]);
  const isPolicyDirty = useMemo(() => {
    if (!executionPolicy || !policyForm) return false;
    return JSON.stringify(executionPolicy) !== JSON.stringify(policyForm);
  }, [executionPolicy, policyForm]);

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
    isDirty || isPolicyDirty,
    "You have unsaved settings changes. Leave without saving?"
  );

  useEffect(() => {
    if (!isDirty && !isPolicyDirty) return;
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [isDirty, isPolicyDirty]);

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
        ) : (isSettingsFetching || isPolicyFetching) ? (
          <InlineLoadingIndicator
            label="Refreshing settings..."
            testId="settings-refreshing-indicator"
          />
        ) : null}
      </div>
      {policyError && !executionPolicy ? (
        <ErrorState
          error={policyError}
          title="Unable to load execution defaults"
          message="You can still update theme and insight preferences."
          onRetry={() => {
            void refetchPolicy();
          }}
        />
      ) : null}

      <Card data-testid={selectors.settings.themeSettings}>
        <h3 className="text-lg font-medium text-slate-200">Theme</h3>
        <p className="mt-1 text-sm text-slate-400">Choose your preferred color theme</p>
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

      <Card>
        <h3 className="text-lg font-medium text-slate-200">Focus</h3>
        <p className="mt-1 text-sm text-slate-400">Set optional analysis focus context</p>
        <div className="mt-4">
          <label className="block text-sm font-medium text-slate-300">Custom Focus</label>
          <Input
            type="text"
            placeholder="e.g., Focus on test coverage..."
            className="mt-1"
            data-testid={selectors.settings.customFocus}
            value={form.customFocus ?? ""}
            onChange={(e) => setForm({ ...form, customFocus: e.target.value })}
          />
        </div>
      </Card>

      <Card data-testid={selectors.settings.insightsSettings}>
        <h3 className="text-lg font-medium text-slate-200">Insights Engine</h3>
        <p className="mt-1 text-sm text-slate-400">Self-improvement suggestions based on patterns</p>
        <div className="mt-4 space-y-3">
          <label className="flex items-center gap-3">
            <input
              type="checkbox"
              className="h-4 w-4 rounded border-white/20 bg-slate-800 text-cyan-500 focus:ring-cyan-500"
              data-testid={selectors.settings.insightsEnabled}
              checked={form.insightsEnabled}
              onChange={(e) => setForm({ ...form, insightsEnabled: e.target.checked })}
            />
            <span className="text-sm text-slate-300">Enable insights</span>
          </label>
          <label className="flex items-center gap-3">
            <input
              type="checkbox"
              className="h-4 w-4 rounded border-white/20 bg-slate-800 text-cyan-500 focus:ring-cyan-500"
              data-testid={selectors.settings.insightsAutoAnalyze}
              checked={form.insightsAutoAnalyze}
              onChange={(e) => setForm({ ...form, insightsAutoAnalyze: e.target.checked })}
            />
            <span className="text-sm text-slate-300">Auto-analyze on scenario changes</span>
          </label>
        </div>
      </Card>

      <Card>
        <h3 className="text-lg font-medium text-slate-200">Execution Defaults</h3>
        <p className="mt-1 text-sm text-slate-400">Default mode/delay used when queue requests omit explicit values.</p>
        {isPolicyLoading && !policyForm ? (
          <div className="mt-4">
            <PanelLoadingState
              label="Loading execution defaults..."
              testId="settings-policy-loading-state"
            />
          </div>
        ) : policyForm ? (
          <div className="mt-4 space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-300">Default Mode</label>
              <div className="mt-2 grid grid-cols-3 gap-2">
                {(["manual", "scheduled", "yolo"] as const).map((mode) => (
                  <button
                    key={mode}
                    className={`rounded-lg border py-2 text-sm font-medium ${policyForm.defaultMode === mode ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
                    onClick={() => setPolicyForm({ ...policyForm, defaultMode: mode })}
                  >
                    {mode}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-300">Default Schedule Delay (seconds)</label>
              <Input
                type="number"
                min={0}
                className="mt-1"
                value={policyForm.defaultDelaySeconds}
                onChange={(e) =>
                  setPolicyForm({
                    ...policyForm,
                    defaultDelaySeconds: Math.max(0, Number(e.target.value || 0)),
                  })
                }
              />
            </div>
          </div>
        ) : (
          <div className="mt-4">
            <PanelLoadingState label="Execution defaults unavailable." />
          </div>
        )}
      </Card>

      <div className="flex justify-end">
        <Button
          onClick={() => {
            updateMutation.mutate(form);
            if (isPolicyDirty && policyForm) {
              updatePolicyMutation.mutate(policyForm);
            }
          }}
          disabled={(!isDirty && !isPolicyDirty) || updateMutation.isPending || updatePolicyMutation.isPending}
          data-testid={selectors.settings.saveButton}
          className="flex items-center gap-2"
        >
          {updateMutation.isPending || updatePolicyMutation.isPending ? "Saving..." : "Save Settings"}
        </Button>
      </div>

      {(updateMutation.isError || updatePolicyMutation.isError) && (
        <ErrorState
          error={(updateMutation.error ?? updatePolicyMutation.error) as Error}
          title="Failed to save settings"
        />
      )}
    </div>
  );
}
