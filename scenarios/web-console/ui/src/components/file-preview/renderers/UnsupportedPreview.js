import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { FileQuestion } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { PreviewActions, PreviewMetaLine } from "./shared";
// UnsupportedPreview is the renderer for files with no dedicated viewer. It
// stays useful: it shows metadata plus download/open/copy-path affordances.
export function UnsupportedPreview({ model }) {
    const { t } = useTranslation();
    return (_jsx("div", { className: "flex h-full items-center justify-center p-6", "data-testid": "file-preview-unsupported", children: _jsxs("div", { className: "flex max-w-md flex-col items-center gap-3 text-center", children: [_jsx("div", { className: "rounded-full border border-wc-default bg-wc-surface-input p-3 text-wc-text-muted", children: _jsx(FileQuestion, { className: "h-6 w-6" }) }), _jsx("h3", { className: "text-sm font-semibold text-wc-text-primary", children: t(strings.messagesFileViewer.unsupportedTitle) }), _jsx("p", { className: "text-sm text-wc-text-muted", children: t(strings.messagesFileViewer.unsupportedBody) }), _jsx(PreviewMetaLine, { model: model }), _jsx(PreviewActions, { model: model, className: "justify-center" })] }) }));
}
