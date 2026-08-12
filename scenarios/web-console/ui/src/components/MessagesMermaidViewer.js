import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useRef, useState } from "react";
import { AlertTriangle, Check, Code, Copy, Eye, Loader2, Maximize, Minus, Plus, RotateCcw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { DrawerShell } from "./DrawerShell";
import { useCodeCopy } from "./markdown/hooks/useCodeCopy";
import { useMermaidSvg } from "./markdown/hooks/useMermaidSvg";
import { MermaidZoomSurface } from "./mermaid-viewer/MermaidZoomSurface";
import { formatScalePercent } from "./mermaid-viewer/zoomTransform";
const toolbarButton = "inline-flex h-8 w-8 items-center justify-center rounded-lg border border-wc-default bg-wc-surface-input text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary";
/**
 * MessagesMermaidViewer is the full-screen, zoomable Mermaid diagram drawer. It
 * reuses the shared DrawerShell and the Mermaid render hook (so initialization
 * stays a singleton), then layers zoom/pan, source toggle, and copy on top.
 * Diagrams are message-local UI content, so this never touches the file-preview
 * controller or its model.
 */
export default function MessagesMermaidViewer({ open, code, onClose }) {
    const { t } = useTranslation();
    const { svgHtml, error, loading } = useMermaidSvg(open ? code : "");
    const [showSource, setShowSource] = useState(false);
    const [scale, setScale] = useState(1);
    const { copied, copyCode } = useCodeCopy(code);
    const surfaceRef = useRef(null);
    // Each newly opened diagram starts on the diagram view.
    useEffect(() => {
        if (open)
            setShowSource(false);
    }, [open, code]);
    const showDiagram = !showSource && !error;
    const headerActions = (_jsxs("div", { className: "flex items-center gap-1.5", children: [showDiagram && (_jsxs(_Fragment, { children: [_jsx("button", { type: "button", onClick: () => surfaceRef.current?.zoomOut(), className: toolbarButton, "aria-label": t(strings.mermaid.zoomOut), title: t(strings.mermaid.zoomOut), children: _jsx(Minus, { className: "h-4 w-4" }) }), _jsx("span", { className: "w-12 text-center font-mono text-xs text-wc-text-muted", "data-testid": "mermaid-zoom-level", children: formatScalePercent(scale) }), _jsx("button", { type: "button", onClick: () => surfaceRef.current?.zoomIn(), className: toolbarButton, "aria-label": t(strings.mermaid.zoomIn), title: t(strings.mermaid.zoomIn), children: _jsx(Plus, { className: "h-4 w-4" }) }), _jsx("button", { type: "button", onClick: () => surfaceRef.current?.fit(), className: toolbarButton, "aria-label": t(strings.mermaid.fitToScreen), title: t(strings.mermaid.fitToScreen), children: _jsx(Maximize, { className: "h-4 w-4" }) }), _jsx("button", { type: "button", onClick: () => surfaceRef.current?.reset(), className: toolbarButton, "aria-label": t(strings.mermaid.resetZoom), title: t(strings.mermaid.resetZoom), children: _jsx(RotateCcw, { className: "h-4 w-4" }) }), _jsx("span", { className: "mx-1 h-5 w-px bg-wc-default", "aria-hidden": "true" })] })), !error && (_jsx("button", { type: "button", onClick: () => setShowSource((prev) => !prev), className: toolbarButton, "aria-label": showSource ? t(strings.mermaid.showDiagram) : t(strings.mermaid.showSource), title: showSource ? t(strings.mermaid.showDiagram) : t(strings.mermaid.showSource), children: showSource ? _jsx(Eye, { className: "h-4 w-4" }) : _jsx(Code, { className: "h-4 w-4" }) })), _jsx("button", { type: "button", onClick: copyCode, className: toolbarButton, "aria-label": copied ? t(strings.mermaid.copied) : t(strings.mermaid.copySource), title: copied ? t(strings.mermaid.copied) : t(strings.mermaid.copySource), children: copied ? _jsx(Check, { className: "h-4 w-4 text-green-400" }) : _jsx(Copy, { className: "h-4 w-4" }) })] }));
    const badges = (_jsxs("div", { className: "mt-2 flex flex-wrap gap-2 text-[11px] text-wc-text-faint", children: [_jsx("span", { className: "rounded-full border border-wc-default px-2 py-0.5", children: t(strings.mermaid.badgeMessageDiagram) }), _jsx("span", { className: "rounded-full border border-wc-default px-2 py-0.5", children: "mermaid" })] }));
    return (_jsx(DrawerShell, { open: open, onClose: onClose, closeAriaLabel: t(strings.mermaid.closeViewer), title: t(strings.mermaid.viewerTitle), headerActions: headerActions, headerExtra: badges, panelTestId: "messages-mermaid-viewer-panel", children: showSource ? (_jsx("pre", { "data-testid": "mermaid-viewer-source", className: "h-full overflow-auto whitespace-pre p-4 font-mono text-sm text-wc-text-primary", children: code })) : error ? (_jsx("div", { className: "h-full overflow-auto px-4 py-4", children: _jsxs("div", { className: "mx-auto max-w-2xl rounded-2xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-300", children: [_jsxs("div", { className: "mb-2 flex items-center gap-2 font-medium", children: [_jsx(AlertTriangle, { className: "h-4 w-4" }), _jsx("span", { children: t(strings.mermaid.renderError) })] }), _jsx("p", { className: "break-words", children: error }), _jsx("pre", { className: "mt-3 overflow-auto whitespace-pre rounded-lg bg-wc-surface-base p-3 font-mono text-xs text-wc-text-primary", children: code })] }) })) : svgHtml ? (_jsx(MermaidZoomSurface, { ref: surfaceRef, svgHtml: svgHtml, onScaleChange: setScale, ariaLabel: t(strings.mermaid.viewerTitle) })) : (_jsxs("div", { className: "flex h-full items-center justify-center gap-2 text-wc-text-muted", children: [_jsx(Loader2, { className: "h-5 w-5 animate-spin" }), loading && _jsx("span", { children: t(strings.mermaid.rendering) })] })) }));
}
