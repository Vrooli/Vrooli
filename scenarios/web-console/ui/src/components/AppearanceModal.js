import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useEffect, useState } from "react";
import { Check, CopyCheck, RotateCcw, Settings2, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useWorkspaceStore, } from "../stores/useWorkspaceStore";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { DEFAULT_THEME_ID } from "../consts/config";
import { cn } from "../lib/classnames";
import { strings } from "../consts/strings";
import { Button } from "./ui/button";
import { ConfirmDialog } from "./ConfirmDialog";
import { DrawerShell } from "./DrawerShell";
import { SettingsCard } from "./settings/primitives";
import AppearancePreview from "./appearance/AppearancePreview";
import HeaderColorPicker from "./appearance/HeaderColorPicker";
import ThemePicker from "./appearance/ThemePicker";
import FontSizeStepper from "./appearance/FontSizeStepper";
/** Transient success feedback lifetime. Errors stay until the next action. */
const FEEDBACK_TTL_MS = 4000;
export default function AppearanceModal() {
    const { t } = useTranslation();
    const appearanceModalPane = useWorkspaceStore((s) => s.appearanceModalPane);
    const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
    const panes = useWorkspaceStore((s) => s.panes);
    const setPaneColor = useWorkspaceStore((s) => s.setPaneColor);
    const setPaneTheme = useWorkspaceStore((s) => s.setPaneTheme);
    const setPaneFontSize = useWorkspaceStore((s) => s.setPaneFontSize);
    const setDeviceFontSize = useWorkspaceStore((s) => s.setDeviceFontSize);
    const applyAppearance = useWorkspaceStore((s) => s.applyAppearance);
    const defaultHeaderColor = useWorkspaceStore((s) => s.defaultHeaderColor);
    const defaultThemeId = useWorkspaceStore((s) => s.defaultThemeId);
    const defaultFontSize = useWorkspaceStore((s) => s.defaultFontSize);
    const setSettingsModalOpen = useWorkspaceStore((s) => s.setSettingsModalOpen);
    const setSettingsInitialTab = useWorkspaceStore((s) => s.setSettingsInitialTab);
    const { syncPaneUpdate, syncPaneUpdates } = useWorkspaceSync();
    const [selectedProps, setSelectedProps] = useState({
        headerColor: true,
        themeId: true,
        fontSize: true,
    });
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [feedback, setFeedback] = useState(null);
    useEffect(() => {
        if (!feedback || feedback.kind === "error")
            return;
        const timer = setTimeout(() => setFeedback(null), FEEDBACK_TTL_MS);
        return () => clearTimeout(timer);
    }, [feedback]);
    const pane = panes.find((p) => p.sessionId === appearanceModalPane);
    const close = useCallback(() => setAppearanceModalPane(null), [setAppearanceModalPane]);
    if (!appearanceModalPane || !pane)
        return null;
    const sessionId = pane.sessionId;
    const currentColor = pane.headerColor;
    const currentThemeId = pane.themeId ?? DEFAULT_THEME_ID;
    const currentFontSize = pane.fontSize ?? 14;
    const otherPaneIds = panes
        .filter((p) => p.sessionId !== sessionId)
        .map((p) => p.sessionId);
    const propertyList = Object.keys(selectedProps).filter((prop) => selectedProps[prop]);
    const noneSelected = propertyList.length === 0;
    const propertyLabels = {
        headerColor: t(strings.appearance.applySection.propHeaderColor),
        themeId: t(strings.appearance.applySection.propTheme),
        fontSize: t(strings.appearance.applySection.propFontSize),
    };
    /** Server payload carrying only the selected properties. */
    const buildPayload = () => ({
        ...(selectedProps.headerColor ? { header_color: currentColor } : {}),
        ...(selectedProps.themeId ? { theme_id: currentThemeId } : {}),
        ...(selectedProps.fontSize ? { font_size: currentFontSize } : {}),
    });
    const handleApplyToOpen = async () => {
        setConfirmOpen(false);
        applyAppearance(sessionId, {
            properties: propertyList,
            toExistingPanes: true,
            asNewPaneDefault: false,
        });
        const failed = await syncPaneUpdates(otherPaneIds, buildPayload());
        setFeedback(failed.length > 0
            ? { kind: "error" }
            : { kind: "applied", count: otherPaneIds.length });
    };
    const handleSetDefault = () => {
        applyAppearance(sessionId, {
            properties: propertyList,
            toExistingPanes: false,
            asNewPaneDefault: true,
        });
        setFeedback({ kind: "defaultSaved" });
    };
    const handleResetToDefaults = () => {
        setPaneColor(sessionId, defaultHeaderColor);
        setPaneTheme(sessionId, defaultThemeId);
        setPaneFontSize(sessionId, defaultFontSize);
        syncPaneUpdate(sessionId, {
            header_color: defaultHeaderColor,
            theme_id: defaultThemeId,
            font_size: defaultFontSize,
        });
    };
    const openDefaultsSettings = () => {
        close();
        setSettingsInitialTab("new-pane-defaults");
        setSettingsModalOpen(true);
    };
    const propChip = (prop) => {
        const on = selectedProps[prop];
        return (_jsxs("button", { type: "button", "data-testid": `appearance-prop-${prop}`, "aria-pressed": on, className: cn("flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium transition-colors", on
                ? "border-wc-accent bg-wc-accent/10 text-wc-text-primary"
                : "border-wc-default text-wc-text-muted hover:text-wc-text-primary"), onClick: () => setSelectedProps((s) => ({ ...s, [prop]: !s[prop] })), children: [_jsx(Check, { className: cn("h-3 w-3", on ? "text-wc-accent" : "opacity-30"), "aria-hidden": "true" }), propertyLabels[prop]] }, prop));
    };
    const feedbackText = feedback
        ? feedback.kind === "applied"
            ? t(strings.appearance.applySection.appliedFeedback, { count: feedback.count })
            : feedback.kind === "defaultSaved"
                ? t(strings.appearance.applySection.defaultSavedFeedback)
                : t(strings.appearance.applySection.applyError)
        : null;
    return (_jsxs(DrawerShell, { open: true, onClose: close, size: "compact", closeAriaLabel: t(strings.appearance.closeAriaLabel), title: t(strings.appearance.title), panelTestId: "appearance-modal", children: [_jsxs("div", { className: "h-full space-y-4 overflow-y-auto p-4", children: [_jsxs("section", { children: [_jsx("h3", { className: "text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2", children: t(strings.appearance.previewHeading) }), _jsx(AppearancePreview, { headerColor: currentColor, themeId: currentThemeId, fontSize: currentFontSize, sessionName: pane.name })] }), _jsxs(SettingsCard, { className: "space-y-5", children: [_jsx(HeaderColorPicker, { currentColor: currentColor, onSelectColor: (color) => {
                                    setPaneColor(sessionId, color);
                                    syncPaneUpdate(sessionId, { header_color: color });
                                }, testIdPrefix: "appearance" }), _jsx(ThemePicker, { currentThemeId: currentThemeId, onSelectTheme: (themeId) => {
                                    setPaneTheme(sessionId, themeId);
                                    syncPaneUpdate(sessionId, { theme_id: themeId });
                                }, testIdPrefix: "appearance" }), _jsx(FontSizeStepper, { currentSize: currentFontSize, onChangeSize: (size) => {
                                    setDeviceFontSize(sessionId, size);
                                    syncPaneUpdate(sessionId, { font_size: size });
                                }, testIdPrefix: "appearance" }), _jsx("div", { className: "border-t border-wc-default pt-3", children: _jsxs("button", { type: "button", "data-testid": "appearance-reset-defaults", className: "flex items-center gap-1.5 text-xs font-medium text-wc-text-muted hover:text-wc-text-primary", onClick: handleResetToDefaults, title: t(strings.appearance.resetToDefaultsHint), children: [_jsx(RotateCcw, { className: "h-3.5 w-3.5", "aria-hidden": "true" }), t(strings.appearance.resetToDefaults)] }) })] }), _jsxs(SettingsCard, { className: "space-y-3", children: [_jsxs("div", { children: [_jsx("h3", { className: "text-xs font-semibold uppercase tracking-wider text-wc-text-muted", children: t(strings.appearance.applySection.heading) }), _jsx("p", { className: "mt-1 text-[11px] text-wc-text-faint", children: t(strings.appearance.applySection.description) })] }), _jsx("div", { className: "flex flex-wrap gap-1.5", children: ["headerColor", "themeId", "fontSize"].map(propChip) }), noneSelected && (_jsx("p", { className: "text-[11px] text-wc-text-muted", children: t(strings.appearance.applySection.noPropsHint) })), _jsxs("div", { className: "space-y-2", children: [otherPaneIds.length > 0 && (_jsxs(Button, { "data-testid": "appearance-apply-open", variant: "outline", className: "w-full", disabled: noneSelected, onClick: () => setConfirmOpen(true), children: [_jsx(CopyCheck, { className: "h-4 w-4 me-2", "aria-hidden": "true" }), t(strings.appearance.applySection.applyToOpen, {
                                                count: otherPaneIds.length,
                                            })] })), _jsxs(Button, { "data-testid": "appearance-set-default", variant: "outline", className: "w-full", disabled: noneSelected, onClick: handleSetDefault, children: [_jsx(Sparkles, { className: "h-4 w-4 me-2", "aria-hidden": "true" }), t(strings.appearance.applySection.setDefault)] })] }), _jsx("div", { "aria-live": "polite", children: feedbackText && (_jsx("p", { "data-testid": "appearance-apply-feedback", className: cn("text-[11px]", feedback?.kind === "error" ? "text-wc-error-text" : "text-wc-text-muted"), children: feedbackText })) }), _jsxs("button", { type: "button", "data-testid": "appearance-manage-defaults", className: "flex items-center gap-1.5 text-[11px] font-medium text-wc-text-muted hover:text-wc-text-primary", onClick: openDefaultsSettings, children: [_jsx(Settings2, { className: "h-3.5 w-3.5", "aria-hidden": "true" }), t(strings.appearance.applySection.manageDefaults)] })] })] }), _jsx(ConfirmDialog, { open: confirmOpen, title: t(strings.appearance.applySection.confirmTitle), body: t(strings.appearance.applySection.confirmBody, {
                    count: otherPaneIds.length,
                    properties: propertyList.map((prop) => propertyLabels[prop]).join(", "),
                }), cancelLabel: t(strings.appearance.applySection.confirmCancel), confirmLabel: t(strings.appearance.applySection.confirmApply), onCancel: () => setConfirmOpen(false), onConfirm: () => {
                    void handleApplyToOpen();
                }, testIdPrefix: "appearance-apply" })] }));
}
