import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useMemo } from "react";
import { Check, Copy, FileText } from "lucide-react";
import { useCodeCopy } from "../hooks/useCodeCopy";
import { looksLikeInlineFileReference } from "../../../lib/fileReferences";
/** Styled inline code with hover-reveal copy button. */
export function InlineCode({ children, onFileReferenceClick }) {
    const textContent = useMemo(() => extractTextContent(children), [children]);
    const { copied, copyCode } = useCodeCopy(textContent);
    const isFileRef = onFileReferenceClick !== undefined &&
        textContent.length > 0 &&
        looksLikeInlineFileReference(textContent);
    if (isFileRef) {
        return (_jsxs("span", { className: "group inline-flex max-w-full items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/80 px-2 py-0.5 text-xs font-mono align-middle", children: [_jsxs("button", { type: "button", onClick: () => onFileReferenceClick?.(textContent), className: "inline-flex min-w-0 items-center gap-1 text-wc-accent hover:text-wc-accent/80 underline underline-offset-2", title: `Open ${textContent}`, children: [_jsx(FileText, { className: "h-3 w-3 shrink-0", "aria-hidden": "true" }), _jsx("code", { className: "min-w-0 break-all [overflow-wrap:anywhere] leading-relaxed", children: children })] }), _jsx("button", { type: "button", onClick: copyCode, className: "opacity-0 group-hover:opacity-100 transition-opacity text-wc-text-muted hover:text-wc-text-primary", "aria-label": copied ? "Copied" : "Copy path", children: copied ? (_jsx(Check, { className: "h-3.5 w-3.5 text-green-400" })) : (_jsx(Copy, { className: "h-3.5 w-3.5" })) })] }));
    }
    return (_jsxs("span", { className: "group inline-flex max-w-full items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/80 px-2 py-0.5 text-xs font-mono text-wc-text-primary align-middle", children: [_jsx("code", { className: "leading-relaxed min-w-0 break-all [overflow-wrap:anywhere]", children: children }), textContent ? (_jsx("button", { type: "button", onClick: copyCode, className: "opacity-0 group-hover:opacity-100 transition-opacity text-wc-text-muted hover:text-wc-text-primary", "aria-label": copied ? "Copied" : "Copy inline code", children: copied ? (_jsx(Check, { className: "h-3.5 w-3.5 text-green-400" })) : (_jsx(Copy, { className: "h-3.5 w-3.5" })) })) : null] }));
}
function extractTextContent(children) {
    if (typeof children === "string")
        return children;
    if (Array.isArray(children))
        return children.map(extractTextContent).join("");
    if (children && typeof children === "object" && "props" in children) {
        return extractTextContent(children.props.children);
    }
    return "";
}
