import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { Button } from "./ui/button";
export default function ErrorBoundaryFallback({ region, message, onReset, }) {
    const { t } = useTranslation();
    return (_jsxs("div", { "data-testid": `error-boundary-${region}`, className: "flex flex-col items-center justify-center gap-3 rounded-md border border-wc-error bg-wc-error-surface p-6 text-sm text-wc-error-text", children: [_jsx(AlertTriangle, { className: "h-6 w-6 text-wc-error-detail" }), _jsx("p", { className: "font-medium", children: t(strings.errorBoundary.somethingWentWrong, { region }) }), _jsx("p", { className: "max-w-md text-center text-xs text-wc-error-detail/70", children: message }), _jsxs(Button, { variant: "outline", size: "sm", onClick: onReset, className: "mt-2", children: [_jsx(RefreshCw, { className: "me-1.5 h-3.5 w-3.5" }), t(strings.errorBoundary.tryAgain)] })] }));
}
