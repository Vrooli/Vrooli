import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
import { AlertTriangle, Check, Copy, Download, ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import { formatBytes } from "../format";
// PreviewActions renders the shared toolbar affordances (download, open in new
// tab, copy path) every renderer can offer. Download/open only show when the
// model exposes a blob href.
export function PreviewActions({ model, className }) {
    const { t } = useTranslation();
    const [copied, setCopied] = useState(false);
    const copyPath = () => {
        if (!model.resolvedPath)
            return;
        void navigator.clipboard.writeText(model.resolvedPath);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };
    return (_jsxs("div", { className: cn("flex flex-wrap items-center gap-2", className), children: [model.blobHref && model.canDownload && (_jsxs("a", { href: model.blobHref, download: model.basename, "data-testid": "file-preview-download", className: "inline-flex items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary", children: [_jsx(Download, { className: "h-3.5 w-3.5" }), t(strings.messagesFileViewer.download)] })), model.blobHref && (_jsxs("a", { href: model.blobHref, target: "_blank", rel: "noreferrer", "data-testid": "file-preview-open-new-tab", className: "inline-flex items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary", children: [_jsx(ExternalLink, { className: "h-3.5 w-3.5" }), t(strings.messagesFileViewer.openInNewTab)] })), _jsxs("button", { type: "button", onClick: copyPath, "data-testid": "file-preview-copy-path", className: "inline-flex items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary", children: [copied ? _jsx(Check, { className: "h-3.5 w-3.5 text-green-400" }) : _jsx(Copy, { className: "h-3.5 w-3.5" }), copied ? t(strings.messagesFileViewer.copied) : t(strings.messagesFileViewer.copyPath)] })] }));
}
// PreviewMetaLine renders the kind/mime/size summary shared across renderers.
export function PreviewMetaLine({ model }) {
    const size = formatBytes(model.sizeBytes);
    return (_jsxs("p", { className: "text-xs text-wc-text-muted", children: [_jsx("span", { className: "uppercase tracking-wide", children: model.kind }), model.mimeType && _jsxs("span", { children: [" \u00B7 ", model.mimeType] }), size && _jsxs("span", { children: [" \u00B7 ", size] })] }));
}
// CenteredPreview is the standard full-height centered shell for media/image
// renderers, with a checkerboard backdrop so transparency is visible.
export function CenteredPreview({ children, checkerboard = false, testId, }) {
    return (_jsx("div", { "data-testid": testId, className: cn("flex h-full w-full flex-col items-center justify-center gap-4 overflow-auto p-6", checkerboard &&
            "bg-[linear-gradient(45deg,var(--wc-surface-base)_25%,transparent_25%,transparent_75%,var(--wc-surface-base)_75%),linear-gradient(45deg,var(--wc-surface-base)_25%,transparent_25%,transparent_75%,var(--wc-surface-base)_75%)] bg-[length:24px_24px] bg-[position:0_0,12px_12px]"), children: children }));
}
// PreviewNotice renders an inline warning/hint block (e.g. media-load failures,
// truncation notices).
export function PreviewNotice({ message, tone = "warn" }) {
    return (_jsxs("div", { "data-testid": "file-preview-notice", className: cn("flex items-start gap-2 rounded-xl border px-3 py-2 text-sm", tone === "warn"
            ? "border-amber-500/30 bg-amber-500/10 text-amber-300"
            : "border-wc-default bg-wc-surface-input text-wc-text-muted"), children: [_jsx(AlertTriangle, { className: "mt-0.5 h-4 w-4 shrink-0" }), _jsx("span", { children: message })] }));
}
