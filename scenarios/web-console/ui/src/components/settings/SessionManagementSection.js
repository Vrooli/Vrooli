import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useEffect, useRef, useState } from "react";
import { AlertCircle, ChevronDown, ChevronUp, Clock, Focus, RotateCcw, Timer, Trash2, } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import { HEADER_COLORS } from "../../consts/config";
import { BACKEND_OPTIONS } from "../../consts/backend-options";
import { POLICY_OPTIONS, parsePolicySelection, policyKey } from "../../consts/policy-options";
import { useCountdown } from "../../hooks/useCountdown";
import { useWorkspaceSync } from "../../hooks/useWorkspaceSync";
import { updateSessionPolicy } from "../../api/sessions";
import { toErrorInfo } from "../../lib/errors";
import { getSessionDefaults, updateSessionDefaults } from "../../api/settings";
import { fetchCapabilities } from "../../api/capabilities";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { Button } from "../ui/button";
import { SettingsCard, SettingsSectionIntro } from "./primitives";
function SessionPolicyControl({ session, onPolicyChange, }) {
    const currentKey = policyKey(session.policy.mode, session.policy.duration);
    const countdown = useCountdown(session.created_at, session.policy.mode, session.policy.duration);
    return (_jsxs("div", { className: "flex items-center gap-1.5", children: [_jsx(Timer, { className: "h-3 w-3 shrink-0 text-wc-text-faint" }), _jsx("select", { "data-testid": `sessions-policy-select-${session.id}`, className: "h-6 rounded-lg border border-wc-default bg-wc-surface px-2 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none", value: currentKey, onChange: (event) => {
                    const parsed = parsePolicySelection(event.target.value);
                    if (!parsed)
                        return;
                    onPolicyChange(session.id, parsed.mode, parsed.duration);
                }, children: POLICY_OPTIONS.map((option) => (_jsx("option", { value: policyKey(option.mode, option.duration), children: option.label }, policyKey(option.mode, option.duration)))) }), countdown && (_jsxs("span", { className: "flex items-center gap-1 text-xs text-wc-text-faint", children: [_jsx(Clock, { className: "h-2.5 w-2.5" }), countdown] }))] }));
}
function SessionDefaultsControl() {
    const { t } = useTranslation();
    const [defaultBackend, setDefaultBackend] = useState("standard");
    const [defaultPolicyKey, setDefaultPolicyKey] = useState("never");
    const [availableBackends, setAvailableBackends] = useState(["standard"]);
    const [saving, setSaving] = useState(false);
    useEffect(() => {
        let cancelled = false;
        // Load current defaults
        getSessionDefaults().then((d) => {
            if (cancelled)
                return;
            setDefaultBackend(d.default_backend);
            if (d.default_policy) {
                setDefaultPolicyKey(policyKey(d.default_policy.mode, d.default_policy.duration));
            }
        }).catch(() => { });
        // Load available backends
        fetchCapabilities().then((caps) => {
            if (cancelled)
                return;
            if (caps.session_backends) {
                setAvailableBackends(caps.session_backends.filter((b) => b.available).map((b) => b.id));
            }
        }).catch(() => { });
        return () => { cancelled = true; };
    }, []);
    const handleBackendChange = useCallback(async (backend) => {
        setDefaultBackend(backend);
        setSaving(true);
        try {
            await updateSessionDefaults({ default_backend: backend });
        }
        catch {
            // Revert on failure would need previous value — best-effort for now
        }
        finally {
            setSaving(false);
        }
    }, []);
    const handlePolicyChange = useCallback(async (value) => {
        setDefaultPolicyKey(value);
        const parsed = parsePolicySelection(value);
        if (!parsed)
            return;
        setSaving(true);
        try {
            await updateSessionDefaults({
                default_policy: { mode: parsed.mode, duration: parsed.duration },
            });
        }
        catch {
            // Best-effort
        }
        finally {
            setSaving(false);
        }
    }, []);
    const showBackendSelector = availableBackends.length > 1;
    const backendOptions = BACKEND_OPTIONS.filter((b) => availableBackends.includes(b.id));
    return (_jsxs(SettingsCard, { className: "space-y-3", children: [_jsxs("div", { children: [_jsx("div", { className: "text-sm font-medium text-wc-text-secondary", children: t(strings.settings.sessionsSection.defaultsTitle) }), _jsx("div", { className: "text-[11px] text-wc-text-muted", children: t(strings.settings.sessionsSection.defaultsHint) })] }), _jsxs("div", { className: "flex flex-wrap items-center gap-4", children: [showBackendSelector && (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("label", { className: "text-xs text-wc-text-secondary", children: t(strings.settings.sessionsSection.defaultBackendLabel) }), _jsx("select", { "data-testid": "session-defaults-backend", className: "h-7 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none", value: defaultBackend, disabled: saving, onChange: (e) => handleBackendChange(e.target.value), children: backendOptions.map((b) => (_jsx("option", { value: b.id, children: b.label }, b.id))) })] })), _jsxs("div", { className: "flex items-center gap-2", children: [_jsx("label", { className: "text-xs text-wc-text-secondary", children: t(strings.settings.sessionsSection.defaultTimeoutLabel) }), _jsx("select", { "data-testid": "session-defaults-policy", className: "h-7 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none", value: defaultPolicyKey, disabled: saving, onChange: (e) => handlePolicyChange(e.target.value), children: POLICY_OPTIONS.map((opt) => (_jsx("option", { value: policyKey(opt.mode, opt.duration), children: opt.label }, policyKey(opt.mode, opt.duration)))) })] })] })] }));
}
export default function SessionManagementSection({ sessions, onDeleteSession, onRequestClose, }) {
    const { t } = useTranslation();
    const panes = useWorkspaceStore((state) => state.panes);
    const activePane = useWorkspaceStore((state) => state.activePane);
    const movePaneToIndex = useWorkspaceStore((state) => state.movePaneToIndex);
    const setActivePane = useWorkspaceStore((state) => state.setActivePane);
    const setPaneColor = useWorkspaceStore((state) => state.setPaneColor);
    const renamePaneById = useWorkspaceStore((state) => state.renamePaneById);
    const resetLayout = useWorkspaceStore((state) => state.resetLayout);
    const { syncActivePane, syncPaneOrder, syncPaneUpdate } = useWorkspaceSync();
    const [editingName, setEditingName] = useState(null);
    const [editValue, setEditValue] = useState("");
    const [policyError, setPolicyError] = useState(null);
    const policyErrorTimer = useRef(null);
    useEffect(() => {
        return () => {
            if (policyErrorTimer.current) {
                clearTimeout(policyErrorTimer.current);
            }
        };
    }, []);
    const handlePolicyChange = useCallback(async (sessionId, mode, duration) => {
        try {
            await updateSessionPolicy(sessionId, { mode, duration });
            setPolicyError(null);
        }
        catch (error) {
            const info = toErrorInfo(error);
            setPolicyError(info.recovery || info.message);
            if (policyErrorTimer.current) {
                clearTimeout(policyErrorTimer.current);
            }
            policyErrorTimer.current = setTimeout(() => setPolicyError(null), 5000);
        }
    }, []);
    const sessionMap = new Map(sessions.map((item) => [item.session.id, item.session]));
    return (_jsxs("div", { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.sessionsSection.eyebrow), title: t(strings.settings.sessionsSection.title), description: t(strings.settings.sessionsSection.description) }), _jsx(SessionDefaultsControl, {}), _jsxs(SettingsCard, { className: "space-y-4", children: [_jsxs("div", { className: "flex items-center justify-between gap-3", children: [_jsxs("div", { children: [_jsx("div", { className: "text-sm font-medium text-wc-text-secondary", children: t(strings.settings.sessionsSection.openTerminals) }), _jsx("div", { className: "text-[11px] text-wc-text-muted", children: t(strings.settings.sessionsSection.openTerminalsHint) })] }), _jsxs(Button, { "data-testid": "sessions-reset-layout", variant: "outline", size: "sm", className: "h-8 px-3 text-xs", onClick: resetLayout, children: [_jsx(RotateCcw, { className: "me-1 h-3 w-3" }), t(strings.settings.sessionsSection.resetLayout)] })] }), policyError && (_jsxs("div", { "data-testid": "sessions-policy-error", className: "flex items-start gap-2 rounded-xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300", children: [_jsx(AlertCircle, { className: "mt-0.5 h-3.5 w-3.5 shrink-0" }), _jsx("span", { children: policyError })] })), panes.length === 0 ? (_jsx("div", { className: "py-6 text-center text-xs text-wc-text-faint", children: t(strings.settings.sessionsSection.noTerminalsOpen) })) : (_jsx("div", { className: "space-y-3", children: panes.map((pane, index) => {
                            const session = sessionMap.get(pane.sessionId);
                            return (_jsx("div", { "data-testid": `sessions-pane-${pane.sessionId}`, className: "rounded-xl border border-wc-default bg-wc-surface-base/70 p-3", children: _jsxs("div", { className: "flex items-start justify-between gap-3", children: [_jsxs("div", { className: "min-w-0 flex-1 space-y-2", children: [_jsxs("div", { className: "flex items-center gap-2", children: [_jsxs("div", { className: "relative group", children: [_jsx("div", { className: "h-4 w-4 rounded-full border border-wc-default", style: {
                                                                        backgroundColor: pane.headerColor !== "transparent"
                                                                            ? pane.headerColor
                                                                            : "rgb(var(--wc-surface-input))",
                                                                    } }), _jsxs("div", { className: "absolute left-0 top-full z-wc-chrome mt-1 hidden gap-1 rounded-xl border border-wc-default bg-wc-surface-raised p-2 shadow-xl group-hover:flex", children: [_jsx("button", { className: "h-4 w-4 rounded-full border border-wc-default", style: { backgroundColor: "rgb(var(--wc-surface-input))" }, onClick: () => {
                                                                                setPaneColor(pane.sessionId, "transparent");
                                                                                syncPaneUpdate(pane.sessionId, { header_color: "transparent" });
                                                                            }, title: t(strings.settings.sessionsSection.noColor) }), HEADER_COLORS.map((color) => (_jsx("button", { className: "h-4 w-4 rounded-full border border-wc-default", style: { backgroundColor: color }, onClick: () => {
                                                                                setPaneColor(pane.sessionId, color);
                                                                                syncPaneUpdate(pane.sessionId, { header_color: color });
                                                                            }, title: color }, color)))] })] }), editingName === pane.sessionId ? (_jsx("input", { className: "min-w-0 flex-1 rounded-lg border border-wc-accent bg-wc-surface-input px-2 py-1 text-sm text-wc-text-primary outline-none", value: editValue, autoFocus: true, onChange: (event) => setEditValue(event.target.value), onBlur: () => {
                                                                if (editValue.trim()) {
                                                                    const trimmed = editValue.trim();
                                                                    renamePaneById(pane.sessionId, trimmed);
                                                                    syncPaneUpdate(pane.sessionId, { name: trimmed });
                                                                }
                                                                setEditingName(null);
                                                            }, onKeyDown: (event) => {
                                                                if (event.key === "Enter") {
                                                                    if (editValue.trim()) {
                                                                        const trimmed = editValue.trim();
                                                                        renamePaneById(pane.sessionId, trimmed);
                                                                        syncPaneUpdate(pane.sessionId, { name: trimmed });
                                                                    }
                                                                    setEditingName(null);
                                                                }
                                                                else if (event.key === "Escape") {
                                                                    setEditingName(null);
                                                                }
                                                            } })) : (_jsx("button", { className: "truncate text-start text-sm font-medium text-wc-text-secondary", onClick: () => {
                                                                setEditingName(pane.sessionId);
                                                                setEditValue(pane.name);
                                                            }, title: t(strings.settings.sessionsSection.renamePane), children: pane.name }))] }), session && (_jsxs("div", { className: "flex items-center gap-3", children: [session.backend === "persistent" && (_jsx("span", { className: "inline-flex items-center rounded-full bg-blue-500/10 px-2 py-0.5 text-[10px] font-medium text-blue-300", children: t(strings.settings.sessionsSection.persistent) })), _jsx(SessionPolicyControl, { session: session, onPolicyChange: handlePolicyChange })] }))] }), _jsxs("div", { className: "flex items-center gap-1", children: [_jsx(Button, { "data-testid": `sessions-pane-up-${pane.sessionId}`, variant: "ghost", size: "icon", className: "h-8 w-8", disabled: index === 0, onClick: () => {
                                                        movePaneToIndex(pane.sessionId, index - 1);
                                                        const reordered = [...panes];
                                                        const removed = reordered.splice(index, 1);
                                                        const moved = removed[0];
                                                        if (moved)
                                                            reordered.splice(index - 1, 0, moved);
                                                        syncPaneOrder(reordered.map((entry) => entry.sessionId), activePane);
                                                    }, title: t(strings.settings.sessionsSection.moveUp), children: _jsx(ChevronUp, { className: "h-3.5 w-3.5" }) }), _jsx(Button, { "data-testid": `sessions-pane-down-${pane.sessionId}`, variant: "ghost", size: "icon", className: "h-8 w-8", disabled: index === panes.length - 1, onClick: () => {
                                                        movePaneToIndex(pane.sessionId, index + 1);
                                                        const reordered = [...panes];
                                                        const removed = reordered.splice(index, 1);
                                                        const moved = removed[0];
                                                        if (moved)
                                                            reordered.splice(index + 1, 0, moved);
                                                        syncPaneOrder(reordered.map((entry) => entry.sessionId), activePane);
                                                    }, title: t(strings.settings.sessionsSection.moveDown), children: _jsx(ChevronDown, { className: "h-3.5 w-3.5" }) }), _jsx(Button, { "data-testid": `sessions-pane-focus-${pane.sessionId}`, variant: "ghost", size: "icon", className: "h-8 w-8", onClick: () => {
                                                        setActivePane(pane.sessionId);
                                                        syncActivePane(panes.map((entry) => entry.sessionId), pane.sessionId);
                                                        onRequestClose();
                                                    }, title: t(strings.settings.sessionsSection.focusPane), children: _jsx(Focus, { className: "h-3.5 w-3.5" }) }), _jsx(Button, { "data-testid": `sessions-pane-remove-${pane.sessionId}`, variant: "ghost", size: "icon", className: "h-8 w-8", onClick: () => onDeleteSession(pane.sessionId), title: t(strings.settings.sessionsSection.terminateSession), children: _jsx(Trash2, { className: "h-3.5 w-3.5 text-wc-error-detail" }) })] })] }) }, pane.sessionId));
                        }) }))] })] }));
}
