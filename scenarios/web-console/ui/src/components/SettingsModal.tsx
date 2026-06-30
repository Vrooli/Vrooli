import { useEffect, useMemo, useState } from "react";
import type { ComponentType } from "react";
import { GripHorizontal, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { SessionInfo } from "../api/sessions";
import { cn } from "../lib/classnames";
import { strings } from "../consts/strings";
import { Button } from "./ui/button";
import IntegrationsSection from "./settings/IntegrationsSection";
import NewPaneDefaultsSection from "./settings/NewPaneDefaultsSection";
import SessionManagementSection from "./settings/SessionManagementSection";
import ShortcutProfilesSection from "./settings/ShortcutProfilesSection";
import TtsSettingsSection from "./settings/TtsSettingsSection";
import VoiceInputSection from "./settings/VoiceInputSection";
import WorkspaceSection from "./settings/WorkspaceSection";
import {
  DEFAULT_SETTINGS_TAB,
  SETTINGS_TAB_IDS,
  useSettingsTabs,
  type SettingsTabId,
} from "./settings/tabs";

const TAB_STORAGE_KEY = "wc-settings-active-tab";

function loadStoredTab(): SettingsTabId {
  if (typeof window === "undefined") return DEFAULT_SETTINGS_TAB;
  try {
    const raw = window.localStorage.getItem(TAB_STORAGE_KEY);
    if (raw && (SETTINGS_TAB_IDS as readonly string[]).includes(raw)) {
      return raw as SettingsTabId;
    }
  } catch {
    // Ignore storage failures and use the default tab.
  }
  return DEFAULT_SETTINGS_TAB;
}

function storeTab(tab: SettingsTabId) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(TAB_STORAGE_KEY, tab);
  } catch {
    // Ignore storage failures — the tab still works for the current session.
  }
}

type SettingsSectionComponent = ComponentType<{
  sessions: Array<{ session: SessionInfo }>;
  onDeleteSession: (id: string) => void;
  onRequestClose: () => void;
  open: boolean;
}>;

const SECTION_COMPONENTS: Record<SettingsTabId, SettingsSectionComponent> = {
  sessions: SessionManagementSection,
  workspace: WorkspaceSection as SettingsSectionComponent,
  "voice-input": VoiceInputSection as SettingsSectionComponent,
  "voice-output": TtsSettingsSection as SettingsSectionComponent,
  shortcuts: ShortcutProfilesSection as SettingsSectionComponent,
  "new-pane-defaults": NewPaneDefaultsSection as SettingsSectionComponent,
  integrations: IntegrationsSection as SettingsSectionComponent,
};

interface SettingsModalProps {
  sessions: Array<{ session: SessionInfo }>;
  onDeleteSession: (id: string) => void;
}

export default function SettingsModal({
  sessions,
  onDeleteSession,
}: SettingsModalProps) {
  const { t } = useTranslation();
  const settingsModalOpen = useWorkspaceStore((state) => state.settingsModalOpen);
  const setSettingsModalOpen = useWorkspaceStore((state) => state.setSettingsModalOpen);
  const isMobile = useMediaQuery("(max-width: 767px)");
  const [activeTab, setActiveTab] = useState<SettingsTabId>(loadStoredTab);
  const settingsTabs = useSettingsTabs();

  const { elementRef, floatingStyle, pointerHandlers, handleClickCapture } =
    useDraggablePosition({
      isActive: settingsModalOpen && !isMobile,
      storageKey: "wc-settings-pos",
      defaultPosition: () => {
        if (typeof window === "undefined") return { x: 80, y: 60 };
        return {
          x: Math.max(12, (window.innerWidth - 1040) / 2),
          y: Math.max(12, window.innerHeight * 0.08),
        };
      },
    });

  useEffect(() => {
    storeTab(activeTab);
  }, [activeTab]);

  const activeDefinition = useMemo(
    () => settingsTabs.find((tab) => tab.id === activeTab) ?? settingsTabs[0],
    [activeTab, settingsTabs],
  );
  const Section = SECTION_COMPONENTS[activeTab];

  if (!settingsModalOpen) return null;

  const close = () => setSettingsModalOpen(false);

  const shellClassName = isMobile
    ? "wc-stable-theme fixed inset-x-0 bottom-0 top-[max(1rem,var(--wc-safe-top,0px))] z-50 flex flex-col overflow-hidden rounded-t-[28px] border border-wc-default bg-wc-surface-raised shadow-2xl"
    : "wc-stable-theme fixed left-0 top-0 z-50 flex h-[min(84vh,760px)] w-[min(96vw,1040px)] flex-col overflow-hidden rounded-[28px] border border-wc-default bg-wc-surface-raised shadow-2xl";

  const shellStyle = isMobile ? undefined : floatingStyle;

  return (
    <>
      <div
        data-testid="settings-backdrop"
        className="fixed inset-0 z-40 bg-wc-backdrop"
        onClick={close}
      />

      <div
        ref={(node) => {
          elementRef.current = node;
        }}
        data-testid="settings-modal"
        className={shellClassName}
        style={shellStyle}
        onPointerDown={(event) => {
          if (isMobile) return;
          const target = event.target as HTMLElement | null;
          const isOnHandle = Boolean(target?.closest("[data-drag-handle]"));
          const isOnControl = Boolean(target?.closest("button, a, input, textarea, select"));
          if (isOnHandle && !isOnControl) {
            pointerHandlers.onPointerDown(event);
          }
        }}
        onPointerMove={isMobile ? undefined : pointerHandlers.onPointerMove}
        onPointerUp={isMobile ? undefined : pointerHandlers.onPointerUp}
        onPointerCancel={isMobile ? undefined : pointerHandlers.onPointerCancel}
        onClickCapture={isMobile ? undefined : handleClickCapture}
      >
        <div
          data-drag-handle={!isMobile ? true : undefined}
          className={cn(
            "border-b border-wc-default px-4 py-3",
            !isMobile && "cursor-grab active:cursor-grabbing select-none touch-none",
          )}
        >
          {isMobile && (
            <div className="mb-3 flex justify-center">
              <div className="h-1.5 w-12 rounded-full bg-wc-text-muted/35" />
            </div>
          )}

          <div className="flex items-start justify-between gap-4">
            <div className="flex items-start gap-3">
              {!isMobile && <GripHorizontal className="mt-0.5 h-4 w-4 text-wc-text-faint" />}
              <div>
                <div className="text-[11px] font-semibold uppercase tracking-[0.24em] text-wc-text-muted">
                  {t(strings.settings.eyebrow)}
                </div>
                <h2 className="text-lg font-semibold text-wc-text-primary">
                  {activeDefinition?.label ?? t(strings.settings.title)}
                </h2>
                <p className="text-sm text-wc-text-faint">
                  {activeDefinition?.description}
                </p>
              </div>
            </div>
            <Button
              data-testid="settings-close"
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={close}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {isMobile ? (
          <>
            <nav
              data-testid="settings-tabs-row"
              className="flex shrink-0 gap-2 overflow-x-auto border-b border-wc-default px-4 py-3"
              role="tablist"
            >
              {settingsTabs.map((tab) => {
                const isActive = tab.id === activeTab;
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.id}
                    data-testid={`settings-tab-${tab.id}`}
                    type="button"
                    role="tab"
                    aria-selected={isActive}
                    className={cn(
                      "flex shrink-0 items-center gap-2 rounded-full border px-3 py-2 text-sm transition-colors",
                      isActive
                        ? "border-wc-accent bg-wc-surface-input text-wc-text-primary"
                        : "border-wc-default bg-wc-surface-base/70 text-wc-text-muted",
                    )}
                    onClick={() => setActiveTab(tab.id)}
                  >
                    <Icon className="h-4 w-4" />
                    <span>{tab.shortLabel}</span>
                  </button>
                );
              })}
            </nav>

            <div className="flex-1 overflow-y-auto px-4 py-4">
              <Section
                sessions={sessions}
                onDeleteSession={onDeleteSession}
                onRequestClose={close}
                open={settingsModalOpen}
              />
            </div>
          </>
        ) : (
          <div className="flex min-h-0 flex-1 overflow-hidden">
            <aside
              data-testid="settings-sidebar"
              className="w-[260px] shrink-0 border-r border-wc-default bg-wc-surface-base/50 p-3"
            >
              <nav className="space-y-1" role="tablist" aria-label={t(strings.settings.sidebarAria)}>
                {settingsTabs.map((tab) => {
                  const isActive = tab.id === activeTab;
                  const Icon = tab.icon;
                  return (
                    <button
                      key={tab.id}
                      data-testid={`settings-tab-${tab.id}`}
                      type="button"
                      role="tab"
                      aria-selected={isActive}
                      className={cn(
                        "flex w-full items-start gap-3 rounded-2xl px-3 py-3 text-start transition-colors",
                        isActive
                          ? "bg-wc-surface-input text-wc-text-primary shadow-sm"
                          : "text-wc-text-muted hover:bg-wc-surface-input/60 hover:text-wc-text-secondary",
                      )}
                      onClick={() => setActiveTab(tab.id)}
                    >
                      <Icon className={cn("mt-0.5 h-4 w-4 shrink-0", isActive && "text-wc-accent")} />
                      <div className="min-w-0">
                        <div className="text-sm font-medium">{tab.label}</div>
                        <div className="text-[11px] text-wc-text-faint">{tab.description}</div>
                      </div>
                    </button>
                  );
                })}
              </nav>
            </aside>

            <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
              <Section
                sessions={sessions}
                onDeleteSession={onDeleteSession}
                onRequestClose={close}
                open={settingsModalOpen}
              />
            </div>
          </div>
        )}
      </div>
    </>
  );
}
