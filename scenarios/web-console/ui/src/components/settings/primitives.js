import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { cn } from "../../lib/classnames";
export function SettingsSectionIntro({ eyebrow, title, description, }) {
    return (_jsxs("div", { className: "space-y-1", children: [_jsx("p", { className: "text-[11px] font-semibold uppercase tracking-[0.22em] text-wc-text-muted", children: eyebrow }), _jsxs("div", { children: [_jsx("h3", { className: "text-lg font-semibold text-wc-text-primary", children: title }), _jsx("p", { className: "text-sm text-wc-text-faint", children: description })] })] }));
}
export function SettingsCard({ className, children, }) {
    return (_jsx("div", { className: cn("rounded-2xl border border-wc-default bg-wc-surface-input/80 p-4", className), children: children }));
}
export function SettingsRow({ label, hint, hintClassName, control, className, }) {
    return (_jsxs("div", { className: cn("flex items-center justify-between gap-4", className), children: [_jsxs("div", { className: "min-w-0", children: [_jsx("div", { className: "text-sm font-medium text-wc-text-secondary", children: label }), hint && _jsx("div", { className: cn("text-[11px] text-wc-text-muted", hintClassName), children: hint })] }), _jsx("div", { className: "shrink-0", children: control })] }));
}
export function SettingsToggle({ checked, onClick, testId, }) {
    return (_jsx("button", { "data-testid": testId, role: "switch", "aria-checked": checked, className: cn("relative inline-flex h-6 w-11 items-center rounded-full transition-colors", checked ? "bg-wc-accent" : "bg-wc-surface-base"), onClick: onClick, children: _jsx("span", { className: cn("inline-block h-4.5 w-4.5 rounded-full bg-white transition-transform", checked ? "translate-x-[22px]" : "translate-x-[3px]") }) }));
}
