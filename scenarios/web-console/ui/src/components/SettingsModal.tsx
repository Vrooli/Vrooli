import { useEffect, useMemo, useState } from "react";
import type { ComponentType } from "react";
import { useTranslation } from "react-i18next";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { SessionInfo } from "../api/sessions";
import { cn } from "../lib/classnames";
import { strings } from "../consts/strings";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import GroupTemplatesPanel from "./settings/GroupTemplatesPanel";
import HandoffRulesPanel from "./settings/HandoffRulesPanel";
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
  templates: GroupTemplatesPanel as SettingsSectionComponent,
  "handoff-rules": HandoffRulesPanel as SettingsSectionComponent,
  integrations: IntegrationsSection as SettingsSectionComponent,
};

/** The library keeps `settings-tab-<id>` reachable so existing flows still address a tab. */
const settingsTabTestId = (tab: string) => `settings-tab-${tab}`;

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
  const settingsInitialTab = useWorkspaceStore((state) => state.settingsInitialTab);
  const setSettingsInitialTab = useWorkspaceStore((state) => state.setSettingsInitialTab);
  const isMobile = useMediaQuery("(max-width: 767px)");
  const [activeTab, setActiveTab] = useState<SettingsTabId>(loadStoredTab);
  const settingsTabs = useSettingsTabs();

  useEffect(() => {
    storeTab(activeTab);
  }, [activeTab]);

  // Consume a one-shot deep-link request (e.g. "Manage defaults" in the
  // appearance modal) — jump to the requested tab, then clear the request.
  useEffect(() => {
    if (!settingsInitialTab) return;
    if ((SETTINGS_TAB_IDS as readonly string[]).includes(settingsInitialTab)) {
      setActiveTab(settingsInitialTab as SettingsTabId);
    }
    setSettingsInitialTab(null);
  }, [settingsInitialTab, setSettingsInitialTab]);

  const activeDefinition = useMemo(
    () => settingsTabs.find((tab) => tab.id === activeTab) ?? settingsTabs[0],
    [activeTab, settingsTabs],
  );
  const Section = SECTION_COMPONENTS[activeTab];

  // The tab strip is the library's, so overflow, roving focus, arrow-key
  // navigation, and the selected-tab scroll-into-view come with it rather than
  // being re-implemented per surface.
  const tabItems = useMemo(
    () =>
      settingsTabs.map((tab) => {
        const Icon = tab.icon;
        return {
          id: tab.id,
          label: isMobile ? tab.shortLabel : tab.label,
          icon: <Icon />,
        };
      }),
    [isMobile, settingsTabs],
  );

  const close = () => {
    setSettingsModalOpen(false);
  };

  // On a small viewport the drawer already names the active section in its
  // header and the section repeats its own description in the body, so the
  // eyebrow and the description would be the third and fourth copies of the
  // same words — on the surface with the least room for them.
  const title = isMobile ? (
    <span className="text-base font-semibold">
      {activeDefinition?.label ?? t(strings.settings.title)}
    </span>
  ) : (
    <>
      <span className="me-3 text-[11px] font-semibold uppercase tracking-[0.24em] text-wc-text-muted">
        {t(strings.settings.eyebrow)}
      </span>
      <span className="text-base font-semibold">
        {activeDefinition?.label ?? t(strings.settings.title)}
      </span>
    </>
  );

  return (
    <FullPageDrawer
      avoidKeyboard
      open={settingsModalOpen}
      onClose={close}
      closeLabel={t(strings.settings.closeAriaLabel)}
      title={title}
      headerExtra={
        isMobile ? undefined : (
          <p className="mt-1 text-sm text-wc-text-faint">{activeDefinition?.description}</p>
        )
      }
      // The tab strip belongs above the scroll region and outside the content
      // gutter: a band that scrolls away with the content is not navigation,
      // and a full-bleed strip is what lets seven tabs use the whole width.
      subheader={
        isMobile ? (
          <div data-testid="settings-tabs-row" className="px-1">
            <Tabs
              ariaLabel={t(strings.settings.sidebarAria)}
              items={tabItems}
              active={activeTab}
              onChange={(next) => {
                setActiveTab(next as SettingsTabId);
              }}
              itemTestId={settingsTabTestId}
            />
          </div>
        ) : undefined
      }
      // This app already decided what "mobile" means, once, in `isMobile`.
      // Leaving the drawer on its own `auto` breakpoint would add a second,
      // independent read of the viewport that can disagree with the first —
      // the same class of split-brain the viewport contract exists to close.
      dismissAffordance={isMobile ? "grabber" : "close"}
      testId="settings-modal"
    >
      {isMobile ? (
        <div className="px-3 py-4">
          <Section
            sessions={sessions}
            onDeleteSession={onDeleteSession}
            onRequestClose={close}
            open={settingsModalOpen}
          />
        </div>
      ) : (
        <div className="flex h-full min-h-0 overflow-hidden">
          <aside
            data-testid="settings-sidebar"
            className="w-[260px] shrink-0 overflow-y-auto border-r border-wc-default bg-wc-surface-base/50 p-3"
          >
            <nav className="space-y-1" role="tablist" aria-label={t(strings.settings.sidebarAria)}>
              {settingsTabs.map((tab) => {
                const isActive = tab.id === activeTab;
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.id}
                    data-testid={settingsTabTestId(tab.id)}
                    type="button"
                    role="tab"
                    aria-selected={isActive}
                    className={cn(
                      "flex w-full items-start gap-3 rounded-2xl px-3 py-3 text-start transition-colors",
                      isActive
                        ? "bg-wc-surface-input text-wc-text-primary shadow-sm"
                        : "text-wc-text-muted hover:bg-wc-surface-input/60 hover:text-wc-text-secondary",
                    )}
                    onClick={() => {
                      setActiveTab(tab.id);
                    }}
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
    </FullPageDrawer>
  );
}
