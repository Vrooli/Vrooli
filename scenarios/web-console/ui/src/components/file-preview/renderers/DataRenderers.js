import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import { parseDelimited } from "../format";
import { CodeLinePreview } from "./TextRenderers";
import { PreviewNotice } from "./shared";
const MAX_CSV_ROWS = 1000;
// CsvPreview renders CSV/TSV as a scrollable table with a sticky header,
// falling back to raw text when parsing yields nothing useful.
export function CsvPreview({ model, text }) {
    const { t } = useTranslation();
    const content = text?.content ?? "";
    const delimiter = model.resolvedPath.toLowerCase().endsWith(".tsv") ? "\t" : ",";
    const parsed = useMemo(() => parseDelimited(content, delimiter), [content, delimiter]);
    if (content.trim() === "") {
        return (_jsx("div", { className: "flex h-full items-center justify-center text-sm text-wc-text-muted", "data-testid": "file-preview-empty", children: t(strings.messagesFileViewer.emptyFile) }));
    }
    if (parsed.length === 0 || parsed.every((r) => r.length <= 1 && (r[0] ?? "") === "")) {
        return (_jsxs("div", { className: "flex h-full flex-col", "data-testid": "file-preview-csv-fallback", children: [_jsx("div", { className: "px-4 pt-3", children: _jsx(PreviewNotice, { message: t(strings.messagesFileViewer.csvParseFallback), tone: "info" }) }), _jsx("div", { className: "min-h-0 flex-1", children: _jsx(CodeLinePreview, { content: content, path: model.resolvedPath, highlightLine: null }) })] }));
    }
    const header = parsed[0] ?? [];
    const bodyRows = parsed.slice(1, 1 + MAX_CSV_ROWS);
    const truncated = parsed.length - 1 > MAX_CSV_ROWS;
    return (_jsxs("div", { className: "flex h-full flex-col", "data-testid": "file-preview-csv", children: [(truncated || text?.truncated) && (_jsx("div", { className: "px-4 pt-3", children: _jsx(PreviewNotice, { message: t(strings.messagesFileViewer.truncatedNotice), tone: "info" }) })), _jsx("div", { className: "min-h-0 flex-1 overflow-auto p-4", children: _jsxs("table", { className: "w-full border-collapse text-left text-xs", children: [_jsx("thead", { className: "sticky top-0 z-wc-chrome", children: _jsx("tr", { children: header.map((cell, i) => (_jsx("th", { className: "border border-wc-default bg-wc-surface-raised px-2 py-1 font-semibold text-wc-text-primary", children: cell }, i))) }) }), _jsx("tbody", { children: bodyRows.map((r, ri) => (_jsx("tr", { className: cn(ri % 2 === 1 && "bg-wc-surface-input/40"), children: header.map((_, ci) => (_jsx("td", { className: "border border-wc-default px-2 py-1 align-top text-wc-text-secondary", children: r[ci] ?? "" }, ci))) }, ri))) })] }) })] }));
}
function classifyDiffLine(line) {
    if (line.startsWith("@@"))
        return "hunk";
    if (line.startsWith("+++") || line.startsWith("---") || line.startsWith("diff ") || line.startsWith("index ")) {
        return "meta";
    }
    if (line.startsWith("+"))
        return "add";
    if (line.startsWith("-"))
        return "remove";
    return "context";
}
const DIFF_LINE_CLASS = {
    add: "bg-emerald-500/10 text-emerald-300",
    remove: "bg-red-500/10 text-red-300",
    hunk: "bg-wc-accent/10 text-wc-accent",
    meta: "text-wc-text-muted",
    context: "text-[#c9d1d9]",
};
// DiffPreview renders unified diff/patch text with additions/removals/hunk
// highlighting and line numbers. Preserves content verbatim.
export function DiffPreview({ model, text }) {
    const { t } = useTranslation();
    const content = text?.content ?? "";
    const lines = useMemo(() => content.split("\n"), [content]);
    if (content.trim() === "") {
        return (_jsx("div", { className: "flex h-full items-center justify-center text-sm text-wc-text-muted", "data-testid": "file-preview-empty", children: t(strings.messagesFileViewer.emptyFile) }));
    }
    return (_jsxs("div", { className: "flex h-full flex-col bg-[#0d1117]", "data-testid": "file-preview-diff", children: [text?.truncated && (_jsx("div", { className: "px-4 pt-3", children: _jsx(PreviewNotice, { message: t(strings.messagesFileViewer.truncatedNotice), tone: "info" }) })), _jsx("div", { className: "min-h-0 flex-1 overflow-auto font-mono text-xs leading-[1.55]", children: lines.map((line, i) => {
                    const kind = classifyDiffLine(line);
                    return (_jsxs("div", { className: cn("flex items-start gap-3 px-3 py-px", DIFF_LINE_CLASS[kind]), children: [_jsx("span", { className: "w-10 shrink-0 select-none text-end text-wc-text-faint/60 tabular-nums", children: i + 1 }), _jsx("pre", { className: "m-0 flex-1 whitespace-pre-wrap break-words bg-transparent p-0", children: line === "" ? " " : line })] }, i));
                }) }), _jsx("div", { className: "shrink-0 border-t border-wc-default/60 bg-wc-surface-base px-4 py-2", children: _jsx("span", { className: "font-mono text-[11px] uppercase tracking-wide text-wc-text-muted", children: model.basename }) })] }));
}
