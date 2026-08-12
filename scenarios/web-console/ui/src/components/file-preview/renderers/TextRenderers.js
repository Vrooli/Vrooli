import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useMemo, useRef, useState } from "react";
import { Minus, Plus, WrapText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { MarkdownRenderer } from "../../markdown";
import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import { getLanguageFromPath, highlightCode, } from "../../../lib/codeHighlighter";
import { PreviewNotice } from "./shared";
// MarkdownPreview renders text-kind markdown via the shared MarkdownRenderer.
export function MarkdownPreview({ text }) {
    const { t } = useTranslation();
    const content = text?.content ?? "";
    if (content.trim() === "") {
        return _jsx(EmptyText, {});
    }
    return (_jsxs("div", { className: "h-full overflow-auto px-4 py-4", "data-testid": "file-preview-markdown", children: [text?.truncated && (_jsx("div", { className: "mb-3", children: _jsx(PreviewNotice, { message: t(strings.messagesFileViewer.truncatedNotice), tone: "info" }) })), _jsx(MarkdownRenderer, { content: content, className: "text-sm" })] }));
}
// CodePreview renders code/text/diff fallback content with line numbers,
// syntax highlighting, font sizing, wrap toggle, and target-line scroll.
export function CodePreview({ model, text }) {
    const content = text?.content ?? "";
    if (content === "") {
        return _jsx(EmptyText, {});
    }
    return (_jsx(CodeLinePreview, { content: content, path: model.resolvedPath, highlightLine: model.line ?? null, truncated: !!text?.truncated }));
}
function EmptyText() {
    const { t } = useTranslation();
    return (_jsx("div", { className: "flex h-full items-center justify-center text-sm text-wc-text-muted", "data-testid": "file-preview-empty", children: t(strings.messagesFileViewer.emptyFile) }));
}
const FONT_SIZES = [11, 12, 13, 14, 15, 16, 18];
const MIN_FONT_SIZE = 11;
const MAX_FONT_SIZE = 18;
const DEFAULT_FONT_SIZE = 13;
export function CodeLinePreview({ content, path, highlightLine, truncated = false, }) {
    const { t } = useTranslation();
    const scrollerRef = useRef(null);
    const lineRefs = useRef({});
    const plainLines = useMemo(() => content.split("\n"), [content]);
    const [highlighted, setHighlighted] = useState(null);
    const [wrap, setWrap] = useState(false);
    const [fontSize, setFontSize] = useState(DEFAULT_FONT_SIZE);
    const language = useMemo(() => getLanguageFromPath(path), [path]);
    useEffect(() => {
        let cancelled = false;
        setHighlighted(null);
        highlightCode(content, language)
            .then((lines) => {
            if (!cancelled)
                setHighlighted(lines);
        })
            .catch(() => {
            if (!cancelled)
                setHighlighted(null);
        });
        return () => {
            cancelled = true;
        };
    }, [content, language]);
    useEffect(() => {
        if (!highlightLine)
            return;
        const node = lineRefs.current[highlightLine];
        if (!node)
            return;
        const raf = requestAnimationFrame(() => {
            node.scrollIntoView({ block: "center", behavior: "smooth" });
        });
        return () => cancelAnimationFrame(raf);
    }, [highlightLine, highlighted, plainLines.length, wrap, fontSize]);
    const lineCount = highlighted?.length ?? plainLines.length;
    const gutterWidth = `${String(lineCount).length}ch`;
    const lines = highlighted ??
        plainLines.map((line, i) => ({
            lineNumber: i + 1,
            tokens: [{ content: line }],
        }));
    const adjustFont = (direction) => {
        setFontSize((prev) => {
            const idx = FONT_SIZES.indexOf(prev);
            const fallback = FONT_SIZES.indexOf(DEFAULT_FONT_SIZE);
            const current = idx === -1 ? fallback : idx;
            const next = Math.max(0, Math.min(FONT_SIZES.length - 1, current + direction));
            return FONT_SIZES[next] ?? prev;
        });
    };
    return (_jsxs("div", { className: "flex h-full flex-col bg-[#0d1117]", "data-testid": "file-preview-code", children: [_jsxs("div", { className: "flex shrink-0 items-center justify-between gap-2 border-b border-wc-default/60 bg-wc-surface-base px-3 py-1.5 text-xs text-wc-text-muted", children: [_jsxs("span", { className: "truncate font-mono text-[11px] uppercase tracking-wide", children: [language ?? t(strings.messagesFileViewer.plaintext), " \u00B7 ", t(strings.messagesFileViewer.linesSuffix, { count: lineCount }), truncated ? ` · ${t(strings.messagesFileViewer.truncatedNotice)}` : ""] }), _jsxs("div", { className: "flex shrink-0 items-center gap-1", children: [_jsx("button", { type: "button", onClick: () => adjustFont(-1), className: "rounded p-1 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-40", "aria-label": t(strings.messagesFileViewer.decreaseFontSize), title: t(strings.messagesFileViewer.decreaseFontSize), disabled: fontSize <= MIN_FONT_SIZE, children: _jsx(Minus, { className: "h-3.5 w-3.5" }) }), _jsxs("span", { className: "tabular-nums text-[11px]", children: [fontSize, "px"] }), _jsx("button", { type: "button", onClick: () => adjustFont(1), className: "rounded p-1 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-40", "aria-label": t(strings.messagesFileViewer.increaseFontSize), title: t(strings.messagesFileViewer.increaseFontSize), disabled: fontSize >= MAX_FONT_SIZE, children: _jsx(Plus, { className: "h-3.5 w-3.5" }) }), _jsxs("button", { type: "button", onClick: () => setWrap((prev) => !prev), className: cn("ms-1 flex items-center gap-1 rounded px-1.5 py-1 text-[11px] transition hover:bg-wc-surface-input hover:text-wc-text-primary", wrap && "bg-wc-accent/15 text-wc-accent hover:bg-wc-accent/20 hover:text-wc-accent"), "aria-label": wrap ? t(strings.messagesFileViewer.disableWordWrap) : t(strings.messagesFileViewer.enableWordWrap), "aria-pressed": wrap, title: wrap ? t(strings.messagesFileViewer.disableWordWrap) : t(strings.messagesFileViewer.enableWordWrap), children: [_jsx(WrapText, { className: "h-3.5 w-3.5" }), _jsx("span", { children: t(strings.messagesFileViewer.wrap) })] })] })] }), _jsx("div", { ref: scrollerRef, className: "min-h-0 flex-1 overflow-auto font-mono leading-[1.55]", style: { fontSize: `${fontSize}px` }, children: lines.map((line) => {
                    const lineNumber = line.lineNumber;
                    const isHighlighted = lineNumber === highlightLine;
                    return (_jsxs("div", { ref: (node) => {
                            lineRefs.current[lineNumber] = node;
                        }, className: cn("flex items-start gap-3 px-3 py-px", isHighlighted && "bg-wc-accent/15"), children: [_jsx("span", { className: cn("shrink-0 select-none text-end text-wc-text-faint/70 tabular-nums", isHighlighted && "text-wc-accent"), style: { minWidth: gutterWidth, fontSize: `${Math.max(10, fontSize - 1)}px` }, children: lineNumber }), _jsx("pre", { className: cn("m-0 flex-1 bg-transparent p-0 text-[#c9d1d9]", wrap ? "whitespace-pre-wrap break-words" : "whitespace-pre"), children: line.tokens.length === 0 || (line.tokens.length === 1 && line.tokens[0]?.content === "")
                                    ? " "
                                    : line.tokens.map((token, i) => (_jsx("span", { style: { color: token.color }, className: token.fontStyle === "italic" ? "italic" : token.fontStyle === "bold" ? "font-bold" : "", children: token.content }, i))) })] }, lineNumber));
                }) })] }));
}
