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
import { ErrorState } from "../components/ui/error-state";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { GeneralTab } from "../components/settings/GeneralTab";
import { ExecutionTab } from "../components/settings/ExecutionTab";
import { WorkshopTab } from "../components/settings/WorkshopTab";
import { ReviewTab } from "../components/settings/ReviewTab";
import { AudioTab } from "../components/settings/AudioTab";
import { selectors } from "../consts/selectors";
import { applyTheme, defaultQueryOptions } from "../lib";
import { settingsService } from "../services";
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
          <TabsTrigger value="audio" data-testid={selectors.settings.tabAudio}>Audio</TabsTrigger>
        </TabsList>

        <TabsContent value="general">
          <GeneralTab form={form} patch={patch} onThemeChange={handleThemeChange} />
        </TabsContent>

        <TabsContent value="execution">
          <ExecutionTab form={form} patch={patch} />
        </TabsContent>

        <TabsContent value="workshop">
          <WorkshopTab form={form} patch={patch} />
        </TabsContent>

        <TabsContent value="review">
          <ReviewTab form={form} patch={patch} />
        </TabsContent>

        <TabsContent value="audio">
          <AudioTab />
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
