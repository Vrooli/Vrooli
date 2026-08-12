import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LayoutGrid, LayoutList, PanelLeft } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useWakeLockStatus } from "../../stores/useWakeLockStatus";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { strings } from "../../consts/strings";
import { Button } from "../ui/button";
import { SettingsCard, SettingsRow, SettingsSectionIntro, SettingsToggle } from "./primitives";
import LocaleSwitcher from "../LocaleSwitcher";
import { deviceIdentity, setDeviceLabel } from "../../lib/deviceIdentity";
const STATUS_HINT_KEYS = {
    active: strings.settings.workspaceSection.wakeLockActive,
    off: strings.settings.workspaceSection.wakeLockDefault,
    unsupported: strings.settings.workspaceSection.wakeLockUnsupported,
    denied: strings.settings.workspaceSection.wakeLockDenied,
    released: strings.settings.workspaceSection.wakeLockReleased,
};
const STATUS_HINT_CLASSES = {
    active: "text-wc-accent",
    denied: "text-wc-error",
    released: "text-yellow-500",
};
export default function WorkspaceSection() {
    const { t } = useTranslation();
    const isMinimapVisible = useWorkspaceStore((state) => state.isMinimapVisible);
    const setMinimapVisible = useWorkspaceStore((state) => state.setMinimapVisible);
    const displayMode = useWorkspaceStore((state) => state.displayMode);
    const setDisplayMode = useWorkspaceStore((state) => state.setDisplayMode);
    const toolbarLayout = useWorkspaceStore((state) => state.toolbarLayout);
    const setToolbarLayout = useWorkspaceStore((state) => state.setToolbarLayout);
    const keepScreenAwake = useWorkspaceStore((state) => state.keepScreenAwake);
    const setKeepScreenAwake = useWorkspaceStore((state) => state.setKeepScreenAwake);
    const adaptiveChrome = useWorkspaceStore((state) => state.adaptiveChrome);
    const setAdaptiveChrome = useWorkspaceStore((state) => state.setAdaptiveChrome);
    const wakeLockStatus = useWakeLockStatus((s) => s.status);
    const [deviceLabel, setLocalDeviceLabel] = useState(() => deviceIdentity().label);
    const defaultHintKey = strings.settings.workspaceSection.wakeLockDefault;
    const wakeLockHint = keepScreenAwake
        ? t(STATUS_HINT_KEYS[wakeLockStatus] ?? defaultHintKey)
        : t(defaultHintKey);
    const wakeLockHintClass = keepScreenAwake
        ? STATUS_HINT_CLASSES[wakeLockStatus]
        : undefined;
    return (_jsxs("div", { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.workspaceSection.eyebrow), title: t(strings.settings.workspaceSection.title), description: t(strings.settings.workspaceSection.description) }), _jsxs(SettingsCard, { className: "space-y-4", children: [_jsx(SettingsRow, { label: t(strings.settings.workspaceSection.paneLayoutLabel), hint: t(strings.settings.workspaceSection.paneLayoutHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsxs(Button, { "data-testid": "display-mode-grid", variant: displayMode === "grid" ? "default" : "outline", size: "sm", className: "h-8 px-3", onClick: () => setDisplayMode("grid"), children: [_jsx(LayoutGrid, { className: "me-1 h-3.5 w-3.5" }), t(strings.settings.workspaceSection.grid)] }), _jsxs(Button, { "data-testid": "display-mode-tabs", variant: displayMode === "tabs" ? "default" : "outline", size: "sm", className: "h-8 px-3", onClick: () => setDisplayMode("tabs"), children: [_jsx(LayoutList, { className: "me-1 h-3.5 w-3.5" }), t(strings.settings.workspaceSection.tabs)] }), _jsxs(Button, { "data-testid": "display-mode-sidebar", variant: displayMode === "sidebar" ? "default" : "outline", size: "sm", className: "h-8 px-3", onClick: () => setDisplayMode("sidebar"), children: [_jsx(PanelLeft, { className: "me-1 h-3.5 w-3.5" }), t(strings.settings.workspaceSection.sidebar)] })] })) }), _jsx(SettingsRow, { label: t(strings.settings.workspaceSection.mobileToolbarLabel), hint: t(strings.settings.workspaceSection.mobileToolbarHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx(Button, { "data-testid": "toolbar-layout-compact", variant: toolbarLayout === "compact" ? "default" : "outline", size: "sm", className: "h-8 px-3", onClick: () => setToolbarLayout("compact"), children: t(strings.settings.workspaceSection.compact) }), _jsx(Button, { "data-testid": "toolbar-layout-expanded", variant: toolbarLayout === "expanded" ? "default" : "outline", size: "sm", className: "h-8 px-3", onClick: () => setToolbarLayout("expanded"), children: t(strings.settings.workspaceSection.expanded) })] })) }), displayMode === "grid" && (_jsx(SettingsRow, { label: t(strings.settings.workspaceSection.minimapLabel), hint: t(strings.settings.workspaceSection.minimapHint), control: (_jsx(SettingsToggle, { testId: "minimap-toggle", checked: isMinimapVisible, onClick: () => setMinimapVisible(!isMinimapVisible) })) })), _jsx(SettingsRow, { label: t(strings.settings.workspaceSection.adaptiveChromeLabel), hint: t(strings.settings.workspaceSection.adaptiveChromeHint), control: (_jsx(SettingsToggle, { testId: "adaptive-chrome-toggle", checked: adaptiveChrome, onClick: () => setAdaptiveChrome(!adaptiveChrome) })) }), _jsx(SettingsRow, { label: t(strings.deviceIdentity.label), hint: t(strings.deviceIdentity.hint), control: _jsx("input", { className: "h-8 rounded border border-wc-default bg-wc-surface-input px-2 text-sm", value: deviceLabel, onChange: (event) => { setLocalDeviceLabel(event.target.value); setDeviceLabel(event.target.value); } }) }), _jsx(SettingsRow, { label: t(strings.settings.workspaceSection.localeLabel), hint: t(strings.settings.workspaceSection.localeHint), control: _jsx(LocaleSwitcher, {}) }), _jsx(SettingsRow, { label: t(strings.settings.workspaceSection.keepAwakeLabel), hint: wakeLockHint, hintClassName: wakeLockHintClass, control: (_jsx(SettingsToggle, { testId: "keep-screen-awake-toggle", checked: keepScreenAwake, onClick: () => setKeepScreenAwake(!keepScreenAwake) })) })] })] }));
}
