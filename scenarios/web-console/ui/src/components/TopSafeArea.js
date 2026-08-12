import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { cn } from "../lib/classnames";
/**
 * Owns the iOS/PWA top safe area without placing content inside it.
 *
 * The first child surface starts below the status-bar/notch region, while the
 * fill strip gives nav modes full control over the status area's color.
 */
export default function TopSafeArea({ children, className, fillClassName, testId, enabled = true, }) {
    return (_jsxs("div", { "data-testid": testId, className: cn("flex shrink-0 flex-col", className), children: [enabled && (_jsx("div", { "data-testid": testId ? `${testId}-fill` : undefined, "aria-hidden": "true", className: cn("h-[var(--wc-safe-top,0px)] shrink-0", fillClassName) })), children] }));
}
