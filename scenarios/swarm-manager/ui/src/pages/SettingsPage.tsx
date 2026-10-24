/**
 * Settings Page - User preferences and system configuration
 *
 * PURPOSE:
 * Allows users to configure UI preferences, execution defaults,
 * Plan Workshop guidance, agent settings, review thresholds, and UI behavior.
 *
 * CURRENT STATUS: Persistent via filesystem-backed unified settings API.
 *
 * Related PRD targets: OT-P1-010
 */

import { useContext, useEffect, useMemo, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useActionMutation } from "../hooks/useActionMutation";
import { Info } from "lucide-react";
import { UNSAFE_NavigationContext } from "react-router-dom";
import { Button } from "../components/ui/button";
import { ErrorState } from "../components/ui/error-state";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { GeneralTab } from "../components/settings/GeneralTab";
import { ExecutionTab } from "../components/settings/ExecutionTab";
import { WorkshopTab } from "../components/settings/WorkshopTab";
import { ReviewTab } from "../components/settings/ReviewTab";
import { AudioTab } from "../components/settings/AudioTab";
import { AutonomyTab } from "../components/settings/AutonomyTab";
import { selectors } from "../consts/selectors";
import { applyTheme, defaultQueryOptions } from "../lib";
import { integrationStatusService, settingsService, statsService, transitionService } from "../services";
import type { StatsResponse } from "../types/stats";
import type { IntegrationStatusResponse } from "../services";
import type { Settings, SettingsPolicyProjection } from "../types";

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

export function SettingsPage() {
  const queryClient = useQueryClient();
  const {
    data: settings,
    isLoading,
    isFetching,
    error,
    refetch,
  } = useQuery<Settings>({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  // Policy-controls projection: which settings are policy-level (consumed by
  // the operation runner's transition policies) and their effective values.
  // Advisory metadata only — failures degrade to static labeling.
  const { data: policyProjection } = useQuery<SettingsPolicyProjection | null>({
    queryKey: ["settings", "policy-projection"],
    queryFn: () => settingsService.getPolicyProjection(),
    ...defaultQueryOptions,
  });

  const { data: transitionCatalog } = useQuery({
    queryKey: ["transition-catalog"],
    queryFn: () => transitionService.list(),
    ...defaultQueryOptions,
  });

  const { data: stats } = useQuery<StatsResponse>({
    queryKey: ["stats", "autonomy-gates"],
    queryFn: () => statsService.getStats(),
    ...defaultQueryOptions,
  });

  const {
    data: integrationStatus,
    isLoading: integrationsLoading,
    error: integrationsError,
  } = useQuery<IntegrationStatusResponse>({
    queryKey: ["integrations"],
    queryFn: () => integrationStatusService.get(),
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

  const updateMutation = useActionMutation({
    mutationFn: settingsService.update,
    errorMessage: "Couldn't save settings",
    // The page prints its own "Settings saved." line; a toast on top of it
    // would say the same thing twice. Failures still toast.
    source: "SettingsPage.update",
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

      <section
        className="rounded-lg border border-slate-700 bg-slate-900/50 p-4"
        data-testid={selectors.settings.integrations}
        aria-labelledby="integration-status-heading"
      >
        <div className="flex items-baseline justify-between gap-4">
          <div>
            <h3 id="integration-status-heading" className="font-medium text-slate-100">
              Integration status
            </h3>
            <p className="mt-1 text-sm text-slate-400">
              Dependencies used to start and observe Swarm Manager workflows.
            </p>
          </div>
          {integrationsLoading && <span className="text-sm text-slate-400">Checking…</span>}
        </div>

        {integrationsError ? (
          <p className="mt-3 text-sm text-amber-300">
            Integration status is currently unavailable. Workflow starts will still perform their
            required preflight checks.
          </p>
        ) : integrationStatus ? (
          <ul className="mt-3 divide-y divide-slate-800">
            {integrationStatus.integrations.map((integration) => (
              <li key={integration.id} className="flex items-start justify-between gap-4 py-3 first:pt-0 last:pb-0">
                <div>
                  <p className="font-medium text-slate-200">{integration.id}</p>
                  <p className="mt-1 text-sm text-slate-400">{integration.degradedBehavior}</p>
                  {integration.affectedTransitions.length > 0 && (
                    <p className="mt-1 text-xs text-slate-500">
                      Affects: {integration.affectedTransitions.join(", ")}
                    </p>
                  )}
                </div>
                <span
                  className={
                    integration.availability === "available"
                      ? "rounded-full bg-emerald-950 px-2 py-1 text-xs font-medium text-emerald-300"
                      : "rounded-full bg-amber-950 px-2 py-1 text-xs font-medium text-amber-300"
                  }
                >
                  {integration.availability}
                </span>
              </li>
            ))}
          </ul>
        ) : null}
      </section>

      <Tabs defaultValue="general" data-testid={selectors.settings.settingsTabs}>
        <TabsList className="w-full">
          <TabsTrigger value="general" data-testid={selectors.settings.tabGeneral}>General</TabsTrigger>
          <TabsTrigger value="execution" data-testid={selectors.settings.tabExecution}>Execution</TabsTrigger>
          <TabsTrigger value="workshop" data-testid={selectors.settings.tabWorkshop}>Plan Workshop</TabsTrigger>
          <TabsTrigger value="review" data-testid={selectors.settings.tabReview}>Review</TabsTrigger>
          <TabsTrigger value="audio" data-testid={selectors.settings.tabAudio}>Audio</TabsTrigger>
          <TabsTrigger value="autonomy" data-testid="settings-tab-autonomy">Autonomy</TabsTrigger>
        </TabsList>

        <TabsContent value="general">
          <GeneralTab form={form} patch={patch} onThemeChange={handleThemeChange} />
        </TabsContent>

        <TabsContent value="execution">
          <ExecutionTab form={form} patch={patch} policyProjection={policyProjection} />
        </TabsContent>

        <TabsContent value="workshop">
          <WorkshopTab />
        </TabsContent>

        <TabsContent value="review">
          <ReviewTab form={form} patch={patch} policyProjection={policyProjection} />
        </TabsContent>

        <TabsContent value="audio">
          <AudioTab />
        </TabsContent>

        <TabsContent value="autonomy">
          <AutonomyTab form={form} patch={patch} transitions={transitionCatalog} stats={stats} />
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
          error={updateMutation.error}
          title="Failed to save settings"
        />
      )}
    </div>
  );
}
