import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
// DOC: docs/internal/SEAMS.md#capability-registry-seam
// DOC: docs/internal/SEAMS.md#connected-scenarios-registry-seam
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckCircle, AlertCircle, Circle, Boxes, Plug, Play, RotateCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useCapabilities } from "../hooks/useCapabilities";
import { strings } from "../consts/strings";
import { runCapabilityAction } from "../api/capabilities";
function statusIcon(status) {
    switch (status) {
        case "available":
            return _jsx(CheckCircle, { className: "h-4 w-4 text-emerald-500 shrink-0" });
        case "unavailable":
            return _jsx(AlertCircle, { className: "h-4 w-4 text-red-500 shrink-0" });
        default:
            return _jsx(Circle, { className: "h-4 w-4 text-wc-text-faint shrink-0" });
    }
}
function borderColor(status) {
    switch (status) {
        case "available":
            return "border-l-emerald-500";
        case "unavailable":
            return "border-l-red-500";
        default:
            return "border-l-wc-default";
    }
}
function actionIcon(kind) {
    if (kind === "scenario_restart") {
        return _jsx(RotateCw, { className: "h-3.5 w-3.5" });
    }
    return _jsx(Play, { className: "h-3.5 w-3.5" });
}
function supportsBackendAction(cap) {
    return cap.dependencyKind === "scenario" && (cap.actionKind === "scenario_start" || cap.actionKind === "scenario_restart");
}
function CapabilityCard({ cap, actionPending, actionResult, actionError, onRunAction }) {
    const { t } = useTranslation();
    const isUnavailable = cap.status === "unavailable";
    const isScenario = cap.dependencyKind === "scenario";
    const canRunAction = supportsBackendAction(cap);
    // For scenario integrations that are unavailable, the message is a
    // CLI-install hint rather than a diagnostic. Treat the badge accordingly.
    const showNotYetBadge = isScenario && cap.status !== "available";
    return (_jsxs("div", { "data-testid": `cap-card-${cap.id}`, className: `rounded-lg border border-wc-default ${borderColor(cap.status)} border-l-2 bg-wc-surface-input px-3 py-2 space-y-1.5`, children: [_jsxs("div", { className: "flex items-center gap-2", children: [statusIcon(cap.status), _jsx("span", { className: "text-xs font-medium text-wc-text-primary", children: cap.name }), _jsx("span", { className: "text-[11px] px-1.5 py-0.5 rounded bg-wc-surface text-wc-text-faint", children: cap.dependencyKind }), showNotYetBadge && (_jsx("span", { "data-testid": `cap-not-yet-${cap.id}`, className: "text-[11px] px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-300 border border-amber-500/30", children: t(strings.integrationsPanel.scenarioNotYetAvailable) }))] }), _jsx("p", { className: "text-[11px] text-wc-text-muted", children: cap.description }), cap.message && (_jsx("p", { "data-testid": `cap-message-${cap.id}`, className: "text-[11px] text-wc-text-faint font-mono", children: cap.message })), (cap.reasonCode || cap.actionLabel || cap.operatorCommand) && (_jsxs("div", { className: "rounded border border-wc-default bg-wc-surface px-2 py-1.5 space-y-1", children: [cap.reasonCode && (_jsxs("p", { "data-testid": `cap-reason-${cap.id}`, className: "text-[11px] text-wc-text-muted", children: [t(strings.integrationsPanel.reasonLabel), " ", _jsx("span", { className: "font-mono text-wc-text-primary", children: cap.reasonCode })] })), cap.actionLabel && (_jsxs("p", { "data-testid": `cap-action-${cap.id}`, className: "text-[11px] text-wc-text-muted", children: [t(strings.integrationsPanel.nextActionLabel), " ", _jsx("span", { className: "text-wc-text-primary", children: cap.actionLabel })] })), cap.operatorCommand && (_jsx("p", { "data-testid": `cap-command-${cap.id}`, className: "text-[11px] text-wc-text-faint font-mono break-all", children: cap.operatorCommand })), canRunAction && (_jsxs("button", { type: "button", "data-testid": `cap-run-action-${cap.id}`, disabled: actionPending, onClick: () => onRunAction(cap), className: "inline-flex h-7 items-center gap-1.5 rounded border border-wc-default bg-wc-surface-input px-2 text-[11px] text-wc-text-primary hover:bg-wc-surface disabled:cursor-wait disabled:opacity-60", children: [actionIcon(cap.actionKind), _jsx("span", { children: actionPending ? t(strings.integrationsPanel.actionRunning) : cap.actionLabel })] }))] })), actionResult && (_jsx("p", { "data-testid": `cap-action-result-${cap.id}`, className: `text-[11px] ${actionResult.success ? "text-emerald-400" : "text-red-400"}`, children: actionResult.message || actionResult.status })), actionError && (_jsx("p", { "data-testid": `cap-action-error-${cap.id}`, className: "text-[11px] text-red-400", children: t(strings.integrationsPanel.actionFailed, { message: actionError }) })), cap.features.length > 0 && (_jsx("div", { className: "flex flex-wrap gap-1 mt-1", children: cap.features.map((feature) => (_jsx("span", { className: `text-[11px] px-1.5 py-0.5 rounded-full border ${isUnavailable
                        ? "border-wc-default text-wc-text-faint line-through"
                        : "border-wc-default text-wc-text-muted"}`, children: feature }, feature))) }))] }));
}
function IntegrationsGroup({ testId, icon, heading, description, items, pendingCapabilityId, lastActionResult, lastActionError, onRunAction, }) {
    if (items.length === 0)
        return null;
    return (_jsxs("section", { "data-testid": testId, className: "space-y-2", children: [_jsxs("header", { className: "flex items-center gap-2 px-1", children: [_jsx("span", { className: "text-wc-text-muted", children: icon }), _jsx("h3", { className: "text-xs font-semibold uppercase tracking-wide text-wc-text-muted", children: heading })] }), _jsx("p", { className: "text-[11px] text-wc-text-faint px-1", children: description }), _jsx("div", { className: "flex flex-col gap-2", children: items.map((cap) => (_jsx(CapabilityCard, { cap: cap, actionPending: pendingCapabilityId === cap.id, actionResult: lastActionResult?.capabilityId === cap.id ? lastActionResult : undefined, actionError: lastActionError?.capabilityId === cap.id ? lastActionError.message : undefined, onRunAction: onRunAction }, cap.id))) })] }));
}
export default function IntegrationsPanel({ open }) {
    const { t } = useTranslation();
    const { data, isLoading, isError, error } = useCapabilities(open);
    const queryClient = useQueryClient();
    const [pendingCapabilityId, setPendingCapabilityId] = useState();
    const [lastActionResult, setLastActionResult] = useState();
    const [lastActionError, setLastActionError] = useState();
    const actionMutation = useMutation({
        mutationFn: (cap) => {
            if (!cap.actionKind)
                throw new Error("action kind missing");
            return runCapabilityAction(cap.id, cap.actionKind);
        },
        onMutate: (cap) => {
            setPendingCapabilityId(cap.id);
            setLastActionResult(undefined);
            setLastActionError(undefined);
        },
        onSuccess: (result) => {
            setLastActionResult(result);
            queryClient.setQueryData(["capabilities"], {
                capabilities: result.capabilities,
                timestamp: result.timestamp,
            });
        },
        onError: (err, cap) => {
            setLastActionError({ capabilityId: cap.id, message: err instanceof Error ? err.message : String(err) });
        },
        onSettled: () => {
            setPendingCapabilityId(undefined);
            queryClient.invalidateQueries({ queryKey: ["capabilities"] });
        },
    });
    const handleRunAction = (cap) => {
        if (supportsBackendAction(cap) && !actionMutation.isPending) {
            actionMutation.mutate(cap);
        }
    };
    if (isLoading) {
        return (_jsx("div", { className: "space-y-4", children: _jsx("p", { className: "text-[11px] text-wc-text-faint", children: t(strings.integrationsPanel.checking) }) }));
    }
    if (isError) {
        return (_jsx("div", { className: "space-y-4", children: _jsx("p", { className: "text-[11px] text-red-400", children: t(strings.integrationsPanel.loadFailed, { message: error.message }) }) }));
    }
    const capabilities = data?.capabilities ?? [];
    const activeCount = capabilities.filter((c) => c.status === "available").length;
    const scenarios = capabilities.filter((c) => c.dependencyKind === "scenario");
    const resources = capabilities.filter((c) => c.dependencyKind !== "scenario");
    if (capabilities.length === 0) {
        return (_jsx("div", { className: "space-y-3", children: _jsx("p", { className: "text-[11px] text-wc-text-faint", children: t(strings.integrationsPanel.noneConfigured) }) }));
    }
    return (_jsxs("div", { className: "space-y-4", children: [_jsx("div", { className: "flex items-center justify-between px-1", children: _jsx("span", { className: "text-[11px] text-wc-text-faint", children: t(strings.integrationsPanel.activeCount, { active: activeCount, total: capabilities.length }) }) }), _jsx(IntegrationsGroup, { testId: "integrations-group-scenarios", icon: _jsx(Boxes, { className: "h-4 w-4" }), heading: t(strings.integrationsPanel.connectedScenariosHeading), description: t(strings.integrationsPanel.connectedScenariosDescription), items: scenarios, pendingCapabilityId: pendingCapabilityId, lastActionResult: lastActionResult, lastActionError: lastActionError, onRunAction: handleRunAction }), _jsx(IntegrationsGroup, { testId: "integrations-group-resources", icon: _jsx(Plug, { className: "h-4 w-4" }), heading: t(strings.integrationsPanel.localResourcesHeading), description: t(strings.integrationsPanel.localResourcesDescription), items: resources, pendingCapabilityId: pendingCapabilityId, lastActionResult: lastActionResult, lastActionError: lastActionError, onRunAction: handleRunAction })] }));
}
