import { useEffect, useMemo, useRef, useState } from "react";
import type { ComponentType } from "react";
import { useTranslation } from "react-i18next";
import { ChevronLeft, Search } from "lucide-react";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { SessionInfo } from "../api/sessions";
import { cn } from "../lib/classnames";
import { strings } from "../consts/strings";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Button } from "@vrooli/react-component-library/Button/2";
import { SearchField } from "./SearchField";
import { PaneEmptyState } from "./PaneEmptyState";
import GroupTemplatesPanel from "./settings/GroupTemplatesPanel";
import HandoffRulesPanel from "./settings/HandoffRulesPanel";
import IntegrationsSection from "./settings/IntegrationsSection";
import NewPaneDefaultsSection from "./settings/NewPaneDefaultsSection";
import NotificationsSection from "./settings/NotificationsSection";
import SessionManagementSection from "./settings/SessionManagementSection";
import ShortcutProfilesSection from "./settings/ShortcutProfilesSection";
import SnippetsPanel from "./settings/SnippetsPanel";
import TtsSettingsSection from "./settings/TtsSettingsSection";
import VoiceInputSection from "./settings/VoiceInputSection";
import WorkspaceSection from "./settings/WorkspaceSection";
import AccountSection from "./settings/AccountSection";
import DevicesSection from "./settings/DevicesSection";
import DeploymentsSection from "./settings/DeploymentsSection";
import type { InstallOutcome } from "../api/capabilities";
import type { Machine } from "../api/machines";
import {
  DEFAULT_SETTINGS_TAB,
  SETTINGS_TAB_IDS,
  useSettingsTabs,
  type SettingsTabId,
} from "./settings/tabs";

const TAB_STORAGE_KEY = "wc-settings-active-tab";
const LOCATION_STORAGE_KEY = "wc-settings-location-v2";
type SettingsLocation = SettingsTabId | "categories";

function loadLocation(): SettingsLocation {
  if (typeof window === "undefined") return "categories";
  try {
    const raw = window.localStorage.getItem(LOCATION_STORAGE_KEY);
    if (raw && (SETTINGS_TAB_IDS as readonly string[]).includes(raw)) return raw as SettingsTabId;
  } catch {
    // Storage can be unavailable in private browsing.
  }
  return "categories";
}

function storeLocation(location: SettingsLocation) {
  try { window.localStorage.setItem(LOCATION_STORAGE_KEY, location); } catch { /* Keep the in-memory location. */ }
}

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
  // Only the Devices section consumes these; other sections ignore them.
  onStartSessionForMachine?: (machine: Machine) => void;
  onInstallCapability?: (capabilityID: string, target: Machine["target"]) => Promise<InstallOutcome>;
}>;

const SECTION_COMPONENTS: Record<SettingsTabId, SettingsSectionComponent> = {
  account: AccountSection as SettingsSectionComponent,
  devices: DevicesSection as SettingsSectionComponent,
  sessions: SessionManagementSection,
  workspace: WorkspaceSection as SettingsSectionComponent,
  "voice-input": VoiceInputSection as SettingsSectionComponent,
  "voice-output": TtsSettingsSection as SettingsSectionComponent,
  shortcuts: ShortcutProfilesSection as SettingsSectionComponent,
  "new-pane-defaults": NewPaneDefaultsSection as SettingsSectionComponent,
  templates: GroupTemplatesPanel as SettingsSectionComponent,
  snippets: SnippetsPanel as SettingsSectionComponent,
  "handoff-rules": HandoffRulesPanel as SettingsSectionComponent,
  integrations: IntegrationsSection as SettingsSectionComponent,
  notifications: NotificationsSection as SettingsSectionComponent,
  deployments: DeploymentsSection as SettingsSectionComponent,
};

/** The library keeps `settings-tab-<id>` reachable so existing flows still address a tab. */
const settingsTabTestId = (tab: string) => `settings-tab-${tab}`;

interface SettingsModalProps {
  sessions: Array<{ session: SessionInfo }>;
  onDeleteSession: (id: string) => void;
  /** Wiring for the Devices tab's "start session on this machine" action. */
  onStartSessionForMachine?: (machine: Machine) => void;
  onInstallCapability?: (capabilityID: string, target: Machine["target"]) => Promise<InstallOutcome>;
}

export default function SettingsModal({
  sessions,
  onDeleteSession,
  onStartSessionForMachine,
  onInstallCapability,
}: SettingsModalProps) {
  const { t } = useTranslation();
  const settingsModalOpen = useWorkspaceStore((state) => state.settingsModalOpen);
  const setSettingsModalOpen = useWorkspaceStore((state) => state.setSettingsModalOpen);
  const settingsInitialTab = useWorkspaceStore((state) => state.settingsInitialTab);
  const setSettingsInitialTab = useWorkspaceStore((state) => state.setSettingsInitialTab);
  const isMobile = useMediaQuery("(max-width: 767px)");
  const [activeTab, setActiveTab] = useState<SettingsTabId>(() => {
    const location = loadLocation();
    return location === "categories" ? loadStoredTab() : location;
  });
  const [showCategories, setShowCategories] = useState(() => loadLocation() === "categories");
  const [settingsQuery, setSettingsQuery] = useState("");
  const categoryNavigation = useRef(false);
  const settingsTabs = useSettingsTabs();

  const filteredTabs = useMemo(() => {
    const needle = settingsQuery.trim().toLowerCase();
    if (!needle) return settingsTabs;
    return settingsTabs.filter((tab) => (
      `${tab.label} ${tab.shortLabel} ${tab.description} ${tab.keywords} ${tab.id}`.toLowerCase().includes(needle)
    ));
  }, [settingsQuery, settingsTabs]);

  const chooseTab = (tab: SettingsTabId) => {
    storeLocation(tab);
    setActiveTab(tab);
    setShowCategories(false);
    setSettingsQuery("");
  };

  useEffect(() => {
    storeTab(activeTab);
  }, [activeTab]);

  // The previous focused control is removed by drill-in; move focus to the
  // corresponding destination so keyboard and screen-reader users retain context.
  useEffect(() => {
    if (!categoryNavigation.current || !settingsModalOpen || !isMobile) return;
    const testId = showCategories ? settingsTabTestId(activeTab) : "settings-categories-back";
    document.querySelector<HTMLElement>(`[data-testid="${testId}"]`)?.focus();
    categoryNavigation.current = false;
  }, [showCategories, activeTab, settingsModalOpen, isMobile]);

  // Opening a voice tab is audio intent: the reader is here to
  // configure the feature, so an unreachable backend is finally
  // news. Latched here rather than left to the panel so both
  // tabs are covered by one rule.
  const markAudioIntent = useWorkspaceStore((state) => state.markAudioIntent);
  useEffect(() => {
    if (!settingsModalOpen) return;
    if ((!isMobile || !showCategories) && (activeTab === "voice-input" || activeTab === "voice-output")) markAudioIntent();
  }, [settingsModalOpen, activeTab, markAudioIntent, isMobile, showCategories]);

  // Consume a one-shot deep-link request (e.g. "Manage defaults" in the
  // appearance modal) — jump to the requested tab, then clear the request.
  useEffect(() => {
    if (!settingsInitialTab) return;
    if ((SETTINGS_TAB_IDS as readonly string[]).includes(settingsInitialTab)) {
      setActiveTab(settingsInitialTab as SettingsTabId);
      setShowCategories(false);
      storeLocation(settingsInitialTab as SettingsTabId);
    }
    setSettingsInitialTab(null);
  }, [settingsInitialTab, setSettingsInitialTab]);

  const activeDefinition = useMemo(
    () => settingsTabs.find((tab) => tab.id === activeTab) ?? settingsTabs[0],
    [activeTab, settingsTabs],
  );
  const Section = SECTION_COMPONENTS[activeTab];

  const close = () => {
    setSettingsModalOpen(false);
  };

  const title = !isMobile || showCategories
    ? t(strings.settings.title)
    : activeDefinition?.label ?? t(strings.settings.title);

  return (
    <FullPageDrawer
      avoidKeyboard
      open={settingsModalOpen}
      onClose={close}
      closeLabel={t(strings.settings.closeAriaLabel)}
      ariaLabel={t(strings.settings.title)}
      title={title}
      centerTitle={isMobile}
      headerLeading={isMobile && !showCategories ? (
        <IconButton
          data-testid="settings-categories-back"
          aria-label={t("settings.backToCategories", { defaultValue: "Back to settings" })}
          size="lg"
          onClick={() => { categoryNavigation.current = true; storeLocation("categories"); setShowCategories(true); }}
        >
          <ChevronLeft aria-hidden="true" className="rtl:rotate-180" />
        </IconButton>
      ) : undefined}
      testId="settings-modal"
    >
      {isMobile ? (
        <div className="px-4 py-4">
          {showCategories ? (
            <>
              <SearchField
                testId="settings-search"
                value={settingsQuery}
                onValueChange={setSettingsQuery}
                placeholder={t("settings.searchPlaceholder", { defaultValue: "Search settings" })}
                label={t("settings.searchPlaceholder", { defaultValue: "Search settings" })}
                clearLabel={t("settings.clearSearch", { defaultValue: "Clear search" })}
                className="mb-3 w-full"
                onKeyDown={(keyEvent) => {
                  if (keyEvent.key !== "Enter" || filteredTabs.length === 0) return;
                  keyEvent.preventDefault();
                  const first = filteredTabs[0];
                  if (!first) return;
                  categoryNavigation.current = true;
                  chooseTab(first.id);
                }}
              />
              {filteredTabs.length === 0 ? (
                <PaneEmptyState
                  testId="settings-search-empty"
                  className="min-h-[18rem]"
                  icon={<Search />}
                  title={t("settings.searchEmptyTitle", { defaultValue: "No settings match" })}
                  description={t("settings.searchEmptyBody", { defaultValue: "Try a shorter word or clear the search to see every category." })}
                  actions={(
                    <Button type="button" variant="secondary" size="sm" onClick={() => { setSettingsQuery(""); }}>
                      Clear search
                    </Button>
                  )}
                />
              ) : (
                <SettingsList variant="flush" data-testid="settings-categories">
                  <SettingsList.Group>
                    {filteredTabs.map((tab) => {
                      const Icon = tab.icon;
                      return <SettingsList.Row key={tab.id} testId={settingsTabTestId(tab.id)} label={tab.label} hint={tab.description} leading={<Icon aria-hidden="true" />} onSelect={() => { categoryNavigation.current = true; chooseTab(tab.id); }} />;
                    })}
                  </SettingsList.Group>
                </SettingsList>
              )}
            </>
          ) : <Section
            sessions={sessions}
            onDeleteSession={onDeleteSession}
            onRequestClose={close}
            open={settingsModalOpen}
            onStartSessionForMachine={onStartSessionForMachine}
            onInstallCapability={onInstallCapability}
          />}
        </div>
      ) : (
        <div className="flex h-full min-h-0 overflow-hidden">
          <aside
            data-testid="settings-sidebar"
            className="w-[240px] shrink-0 overflow-y-auto border-r border-wc-default bg-wc-surface-base/50 p-3"
          >
            <SearchField
              testId="settings-search"
              value={settingsQuery}
              onValueChange={setSettingsQuery}
              placeholder={t("settings.searchPlaceholder", { defaultValue: "Search settings" })}
              label={t("settings.searchPlaceholder", { defaultValue: "Search settings" })}
              clearLabel={t("settings.clearSearch", { defaultValue: "Clear search" })}
              className="mb-3 w-full"
              onKeyDown={(keyEvent) => {
                if (keyEvent.key !== "Enter" || filteredTabs.length === 0) return;
                keyEvent.preventDefault();
                const first = filteredTabs[0];
                if (first) chooseTab(first.id);
              }}
            />
            {filteredTabs.length === 0 ? (
              <div data-testid="settings-search-empty" className="px-2 py-6 text-center">
                <p className="text-xs font-medium text-wc-text-secondary">{t("settings.searchEmptyTitle", { defaultValue: "No settings match" })}</p>
                <p className="mt-1 text-[11px] leading-relaxed text-wc-text-faint">{t("settings.searchEmptyBody", { defaultValue: "Try a shorter word or clear the search to see every category." })}</p>
                <Button type="button" variant="ghost" size="sm" className="mt-2" onClick={() => { setSettingsQuery(""); }}>
                  Clear search
                </Button>
              </div>
            ) : (
              <nav className="space-y-1" role="tablist" aria-label={t(strings.settings.sidebarAria)}>
                {filteredTabs.map((tab) => {
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
                        "flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-start transition-colors",
                        isActive
                          ? "bg-wc-surface-input text-wc-text-primary shadow-sm"
                          : "text-wc-text-muted hover:bg-wc-surface-input/60 hover:text-wc-text-secondary",
                      )}
                      onClick={() => { chooseTab(tab.id); }}
                    >
                      <Icon className={cn("mt-0.5 h-4 w-4 shrink-0", isActive && "text-wc-accent")} />
                      <span className="min-w-0 text-sm font-medium">{tab.label}</span>
                    </button>
                  );
                })}
              </nav>
            )}
          </aside>

          <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
            <Section
              sessions={sessions}
              onDeleteSession={onDeleteSession}
              onRequestClose={close}
              open={settingsModalOpen}
              onStartSessionForMachine={onStartSessionForMachine}
              onInstallCapability={onInstallCapability}
            />
          </div>
        </div>
      )}
    </FullPageDrawer>
  );
}
