import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import { strings } from "../consts/strings";
import { DrawerShell } from "./DrawerShell";
import IntegrationsSection from "./settings/IntegrationsSection";
import NewPaneDefaultsSection from "./settings/NewPaneDefaultsSection";
import SessionManagementSection from "./settings/SessionManagementSection";
import ShortcutProfilesSection from "./settings/ShortcutProfilesSection";
import TtsSettingsSection from "./settings/TtsSettingsSection";
import VoiceInputSection from "./settings/VoiceInputSection";
import WorkspaceSection from "./settings/WorkspaceSection";
import { DEFAULT_SETTINGS_TAB, SETTINGS_TAB_IDS, useSettingsTabs, } from "./settings/tabs";
const TAB_STORAGE_KEY = "wc-settings-active-tab";
function loadStoredTab() {
    if (typeof window === "undefined")
        return DEFAULT_SETTINGS_TAB;
    try {
        const raw = window.localStorage.getItem(TAB_STORAGE_KEY);
        if (raw && SETTINGS_TAB_IDS.includes(raw)) {
            return raw;
        }
    }
    catch {
        // Ignore storage failures and use the default tab.
    }
    return DEFAULT_SETTINGS_TAB;
}
function storeTab(tab) {
    if (typeof window === "undefined")
        return;
    try {
        window.localStorage.setItem(TAB_STORAGE_KEY, tab);
    }
    catch {
        // Ignore storage failures — the tab still works for the current session.
    }
}
const SECTION_COMPONENTS = {
    sessions: SessionManagementSection,
    workspace: WorkspaceSection,
    "voice-input": VoiceInputSection,
    "voice-output": TtsSettingsSection,
    shortcuts: ShortcutProfilesSection,
    "new-pane-defaults": NewPaneDefaultsSection,
    integrations: IntegrationsSection,
};
export default function SettingsModal({ sessions, onDeleteSession, }) {
    const { t } = useTranslation();
    const settingsModalOpen = useWorkspaceStore((state) => state.settingsModalOpen);
    const setSettingsModalOpen = useWorkspaceStore((state) => state.setSettingsModalOpen);
    const settingsInitialTab = useWorkspaceStore((state) => state.settingsInitialTab);
    const setSettingsInitialTab = useWorkspaceStore((state) => state.setSettingsInitialTab);
    const isMobile = useMediaQuery("(max-width: 767px)");
    const [activeTab, setActiveTab] = useState(loadStoredTab);
    const settingsTabs = useSettingsTabs();
    useEffect(() => {
        storeTab(activeTab);
    }, [activeTab]);
    // Consume a one-shot deep-link request (e.g. "Manage defaults" in the
    // appearance modal) — jump to the requested tab, then clear the request.
    useEffect(() => {
        if (!settingsInitialTab)
            return;
        if (SETTINGS_TAB_IDS.includes(settingsInitialTab)) {
            setActiveTab(settingsInitialTab);
        }
        setSettingsInitialTab(null);
    }, [settingsInitialTab, setSettingsInitialTab]);
    const activeDefinition = useMemo(() => settingsTabs.find((tab) => tab.id === activeTab) ?? settingsTabs[0], [activeTab, settingsTabs]);
    const Section = SECTION_COMPONENTS[activeTab];
    const close = () => setSettingsModalOpen(false);
    return (_jsx(DrawerShell, { open: settingsModalOpen, onClose: close, closeAriaLabel: t(strings.settings.closeAriaLabel), title: _jsxs(_Fragment, { children: [_jsx("span", { className: "me-3 text-[11px] font-semibold uppercase tracking-[0.24em] text-wc-text-muted", children: t(strings.settings.eyebrow) }), _jsx("span", { className: "text-base font-semibold", children: activeDefinition?.label ?? t(strings.settings.title) })] }), headerExtra: _jsx("p", { className: "mt-1 text-sm text-wc-text-faint", children: activeDefinition?.description }), panelTestId: "settings-modal", children: isMobile ? (_jsxs("div", { className: "flex h-full flex-col", children: [_jsx("nav", { "data-testid": "settings-tabs-row", className: "flex shrink-0 gap-2 overflow-x-auto border-b border-wc-default px-4 py-3", role: "tablist", children: settingsTabs.map((tab) => {
                        const isActive = tab.id === activeTab;
                        const Icon = tab.icon;
                        return (_jsxs("button", { "data-testid": `settings-tab-${tab.id}`, type: "button", role: "tab", "aria-selected": isActive, className: cn("flex shrink-0 items-center gap-2 rounded-full border px-3 py-2 text-sm transition-colors", isActive
                                ? "border-wc-accent bg-wc-surface-input text-wc-text-primary"
                                : "border-wc-default bg-wc-surface-base/70 text-wc-text-muted"), onClick: () => setActiveTab(tab.id), children: [_jsx(Icon, { className: "h-4 w-4" }), _jsx("span", { children: tab.shortLabel })] }, tab.id));
                    }) }), _jsx("div", { className: "min-h-0 flex-1 overflow-y-auto px-4 py-4", children: _jsx(Section, { sessions: sessions, onDeleteSession: onDeleteSession, onRequestClose: close, open: settingsModalOpen }) })] })) : (_jsxs("div", { className: "flex h-full min-h-0 overflow-hidden", children: [_jsx("aside", { "data-testid": "settings-sidebar", className: "w-[260px] shrink-0 overflow-y-auto border-r border-wc-default bg-wc-surface-base/50 p-3", children: _jsx("nav", { className: "space-y-1", role: "tablist", "aria-label": t(strings.settings.sidebarAria), children: settingsTabs.map((tab) => {
                            const isActive = tab.id === activeTab;
                            const Icon = tab.icon;
                            return (_jsxs("button", { "data-testid": `settings-tab-${tab.id}`, type: "button", role: "tab", "aria-selected": isActive, className: cn("flex w-full items-start gap-3 rounded-2xl px-3 py-3 text-start transition-colors", isActive
                                    ? "bg-wc-surface-input text-wc-text-primary shadow-sm"
                                    : "text-wc-text-muted hover:bg-wc-surface-input/60 hover:text-wc-text-secondary"), onClick: () => setActiveTab(tab.id), children: [_jsx(Icon, { className: cn("mt-0.5 h-4 w-4 shrink-0", isActive && "text-wc-accent") }), _jsxs("div", { className: "min-w-0", children: [_jsx("div", { className: "text-sm font-medium", children: tab.label }), _jsx("div", { className: "text-[11px] text-wc-text-faint", children: tab.description })] })] }, tab.id));
                        }) }) }), _jsx("div", { className: "min-h-0 flex-1 overflow-y-auto px-6 py-5", children: _jsx(Section, { sessions: sessions, onDeleteSession: onDeleteSession, onRequestClose: close, open: settingsModalOpen }) })] })) }));
}
