/**
 * Settings Page - User preferences and system configuration
 *
 * PURPOSE:
 * Allows users to configure UI preferences and system behavior.
 * Controls the recommendation engine mode, insights engine, and visual theme.
 *
 * CURRENT STATUS: Persistent via filesystem-backed settings API.
 *
 * Related PRD targets: OT-P1-010
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { HelpCircle, Info, RefreshCw } from "lucide-react";
import { useBlocker } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { Input } from "../components/ui/input";
import { selectors } from "../consts/selectors";
import { applyTheme, defaultQueryOptions } from "../lib";
import { settingsService } from "../services";
import type { RecommendationMode, Settings } from "../types";

/** Contextual hints for recommendation modes */
const REC_MODE_HINTS = {
  off: "No recommendations will be generated. Use this if you want full manual control.",
  suggestions: "System analyzes scenarios and suggests improvements. You review and approve each one.",
  yolo: "Low-risk recommendations are auto-approved after a brief delay. High-risk changes still require approval.",
} as const;

export function SettingsPage() {
  const queryClient = useQueryClient();
  const { data: settings, isLoading, error, refetch } = useQuery({
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

  const blocker = useBlocker(isDirty);

  useEffect(() => {
    if (blocker.state !== "blocked") return;
    const confirmLeave = window.confirm("You have unsaved settings changes. Leave without saving?");
    if (confirmLeave) {
      blocker.proceed();
    } else {
      blocker.reset();
    }
  }, [blocker]);

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

  const handleModeChange = (mode: RecommendationMode) => {
    if (!form) return;
    setForm({ ...form, recommendationMode: mode });
  };

  const handleThemeChange = (theme: Settings["theme"]) => {
    if (!form) return;
    setForm({ ...form, theme });
    applyTheme(theme);
  };

  if (isLoading) {
    return (
      <div className="space-y-6" data-testid={selectors.settings.page}>
        <Card padding="lg" centered>
          <p className="text-slate-400">Loading settings...</p>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6" data-testid={selectors.settings.page}>
        <ErrorState error={error as Error} title="Unable to load settings" onRetry={() => refetch()} />
      </div>
    );
  }

  if (!form) return null;

  const recommendationSourceKeys: Array<keyof Settings["recommendationSources"]> = [
    "problems",
    "completeness",
    "tests",
    "coverage",
    "customFocus",
    "scenarioNotes",
  ];
  const recommendationSourceEntries = recommendationSourceKeys.map((key) => ({
    key,
    value: form.recommendationSources[key],
  }));

  return (
    <div className="space-y-6" data-testid={selectors.settings.page}>
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Settings</h2>
        {savedMessage && (
          <div className="flex items-center gap-2 text-sm text-emerald-400">
            <Info className="h-4 w-4" />
            {savedMessage}
          </div>
        )}
      </div>

      {/* Theme Settings */}
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

      {/* Recommendation Engine Settings */}
      <Card data-testid={selectors.settings.recommendationSettings}>
        <h3 className="text-lg font-medium text-slate-200">Recommendation Engine</h3>
        <p className="mt-1 text-sm text-slate-400">Control how the system suggests improvements</p>
        <div className="mt-4 space-y-3">
          <div className="flex gap-2">
            <button
              className={`flex-1 rounded-lg border py-3 text-sm font-medium ${form.recommendationMode === "off" ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
              data-testid={selectors.settings.recModeOff}
              onClick={() => handleModeChange("off")}
            >
              Off
            </button>
            <button
              className={`flex-1 rounded-lg border py-3 text-sm font-medium ${form.recommendationMode === "suggestions" ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
              data-testid={selectors.settings.recModeSuggestions}
              onClick={() => handleModeChange("suggestions")}
            >
              Suggestions
            </button>
            <button
              className={`flex-1 rounded-lg border py-3 text-sm font-medium ${form.recommendationMode === "yolo" ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
              data-testid={selectors.settings.recModeYolo}
              onClick={() => handleModeChange("yolo")}
            >
              YOLO
            </button>
          </div>
          <div className="flex items-start gap-2 rounded-lg bg-slate-800/50 px-3 py-2" data-testid={selectors.settings.recModeHint}>
            <HelpCircle className="h-4 w-4 text-slate-500 mt-0.5 flex-shrink-0" />
            <p className="text-xs text-slate-400">{REC_MODE_HINTS[form.recommendationMode]}</p>
          </div>
        </div>
        <div className="mt-4 space-y-3">
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
        <div className="mt-6 space-y-3">
          <p className="text-sm font-medium text-slate-300">Recommendation sources</p>
          {recommendationSourceEntries.map(({ key, value }) => (
            <label key={key} className="flex items-center gap-3 text-sm text-slate-300">
              <input
                type="checkbox"
                checked={value}
                onChange={(e) =>
                  setForm({
                    ...form,
                    recommendationSources: { ...form.recommendationSources, [key]: e.target.checked },
                  })
                }
                className="h-4 w-4 rounded border-white/20 bg-slate-800 text-cyan-500 focus:ring-cyan-500"
              />
              <span className="capitalize">{key.replace(/([A-Z])/g, " $1")}</span>
            </label>
          ))}
        </div>
        <div className="mt-6 space-y-3">
          <p className="text-sm font-medium text-slate-300">Auto-refresh</p>
          <label className="flex items-center gap-3 text-sm text-slate-300">
            <input
              type="checkbox"
              checked={form.recommendationAutoSync.enabled}
              onChange={(e) =>
                setForm({
                  ...form,
                  recommendationAutoSync: { ...form.recommendationAutoSync, enabled: e.target.checked },
                })
              }
              className="h-4 w-4 rounded border-white/20 bg-slate-800 text-cyan-500 focus:ring-cyan-500"
            />
            <span>Enable scheduled refresh</span>
          </label>
          <Input
            type="text"
            value={form.recommendationAutoSync.interval}
            onChange={(e) =>
              setForm({
                ...form,
                recommendationAutoSync: { ...form.recommendationAutoSync, interval: e.target.value },
              })
            }
            className="max-w-xs"
            placeholder="1h"
          />
          <p className="text-xs text-slate-500">Use a duration like 15m, 1h, 6h.</p>
        </div>
      </Card>

      {/* Insights Settings */}
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

      {/* Save Button */}
      <div className="flex items-center justify-between">
        <div className="text-xs text-slate-500 flex items-center gap-2">
          <RefreshCw className="h-3.5 w-3.5" />
          Theme previews apply immediately. Save to persist changes.
        </div>
        <Button
          disabled={!isDirty || updateMutation.isPending}
          data-testid={selectors.settings.saveButton}
          onClick={() => updateMutation.mutate(form)}
        >
          {updateMutation.isPending ? "Saving..." : "Save Settings"}
        </Button>
      </div>
      {updateMutation.isError && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
          Failed to save settings. Please try again.
        </div>
      )}
    </div>
  );
}
