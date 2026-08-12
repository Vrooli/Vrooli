import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Check } from "lucide-react";
import { useTranslation } from "react-i18next";
import { TERMINAL_THEMES } from "../../consts/config";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";
export default function ThemePicker({ currentThemeId, onSelectTheme, testIdPrefix = "appearance", }) {
    const { t } = useTranslation();
    return (_jsxs("section", { children: [_jsx("h3", { className: "text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2", children: t(strings.appearance.terminalThemeHeading) }), _jsx("div", { className: "grid grid-cols-2 gap-2", children: Object.values(TERMINAL_THEMES).map((theme) => {
                    const selected = currentThemeId === theme.id;
                    return (_jsxs("button", { type: "button", "data-testid": `${testIdPrefix}-theme-${theme.id}`, className: cn("rounded-lg border p-2 text-start transition-colors", selected
                            ? "border-wc-accent ring-1 ring-wc-accent"
                            : "border-wc-default hover:border-wc-text-faint"), onClick: () => onSelectTheme(theme.id), "aria-pressed": selected, children: [_jsxs("div", { className: "rounded px-2 py-1.5 mb-1.5 font-mono text-[10px] leading-tight", style: { backgroundColor: theme.colors.background, color: theme.colors.foreground }, children: [_jsx("span", { children: "$ hello world" }), _jsx("span", { className: "inline-block ms-0.5 h-2.5 w-1 align-middle rounded-sm", style: { backgroundColor: theme.colors.cursor } })] }), _jsxs("span", { className: "flex items-center gap-1 text-xs text-wc-text-secondary", children: [selected && _jsx(Check, { className: "h-3 w-3 text-wc-accent", "aria-hidden": "true" }), theme.label] })] }, theme.id));
                }) })] }));
}
