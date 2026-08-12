import { jsx as _jsx, Fragment as _Fragment } from "react/jsx-runtime";
import { Component, memo, useMemo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { remarkProsePaths } from "./utils/remarkProsePaths";
import { CodeBlock } from "./components/CodeBlock";
import { InlineCode } from "./components/InlineCode";
import { MermaidDiagram } from "./components/MermaidDiagram";
import { isExternalHref } from "../../lib/fileReferences";
class MarkdownErrorBoundary extends Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false };
    }
    static getDerivedStateFromError() {
        return { hasError: true };
    }
    componentDidCatch(error, errorInfo) {
        console.error("[MarkdownRenderer] Failed to render content:", error, errorInfo);
    }
    render() {
        if (this.state.hasError) {
            return (_jsx("pre", { className: "whitespace-pre-wrap text-sm text-wc-text-primary font-mono", children: this.props.content }));
        }
        return this.props.children;
    }
}
/** Renders markdown content with syntax highlighting, mermaid diagrams, and GFM support. */
export const MarkdownRenderer = memo(function MarkdownRenderer({ content, className, onLinkClick, onFileReferenceClick, onMermaidOpen, }) {
    const components = useMemo(() => ({
        code: ({ inline, className: codeClassName, children, ...props }) => {
            const codeContent = extractTextContent(children);
            const isInline = inline ?? (!codeClassName && !codeContent.includes("\n"));
            if (isInline)
                return _jsx(InlineCode, { onFileReferenceClick: onFileReferenceClick, children: children });
            if (codeClassName === "language-mermaid")
                return _jsx(MermaidDiagram, { code: codeContent, onOpenFullscreen: onMermaidOpen });
            return _jsx(CodeBlock, { code: codeContent, className: codeClassName, ...props });
        },
        pre: ({ children }) => _jsx(_Fragment, { children: children }),
        a: ({ href, children, ...props }) => {
            const safeHref = href ?? "";
            const external = isExternalHref(safeHref);
            // Auto-detected bare paths get a quieter treatment than authored
            // links: dotted underline, solid on hover.
            const isProsePath = props["data-prose-path"] === "true";
            return (_jsx("a", { href: safeHref, target: external ? "_blank" : undefined, rel: external ? "noopener noreferrer" : undefined, className: isProsePath
                    ? "text-wc-accent/90 underline decoration-dotted underline-offset-2 hover:decoration-solid hover:text-wc-accent"
                    : "text-wc-accent underline underline-offset-2 hover:text-wc-accent/80", title: isProsePath ? `Open ${safeHref}` : undefined, onClick: (event) => onLinkClick?.(safeHref, event), ...props, children: children }));
        },
        h1: ({ children }) => (_jsx("h1", { className: "text-2xl font-bold mt-6 mb-4 text-wc-text-primary", children: children })),
        h2: ({ children }) => (_jsx("h2", { className: "text-xl font-bold mt-5 mb-3 text-wc-text-primary", children: children })),
        h3: ({ children }) => (_jsx("h3", { className: "text-lg font-semibold mt-4 mb-2 text-wc-text-primary", children: children })),
        h4: ({ children }) => (_jsx("h4", { className: "text-base font-semibold mt-3 mb-2 text-wc-text-secondary", children: children })),
        p: ({ children }) => (_jsx("p", { className: "my-2 leading-relaxed break-words [overflow-wrap:anywhere]", children: children })),
        ul: ({ children }) => (_jsx("ul", { className: "list-disc list-inside my-2 space-y-1", children: children })),
        ol: ({ children }) => (_jsx("ol", { className: "list-decimal list-inside my-2 space-y-1", children: children })),
        li: ({ children }) => (_jsx("li", { className: "leading-relaxed break-words [overflow-wrap:anywhere]", children: children })),
        blockquote: ({ children }) => (_jsx("blockquote", { className: "border-l-4 border-wc-accent ps-4 my-3 italic text-wc-text-secondary", children: children })),
        hr: () => _jsx("hr", { className: "my-6 border-wc-default" }),
        table: ({ children }) => (_jsx("div", { className: "overflow-x-auto my-4", children: _jsx("table", { className: "w-auto border-collapse border border-wc-default", children: children }) })),
        thead: ({ children }) => (_jsx("thead", { className: "bg-wc-surface-raised", children: children })),
        th: ({ children }) => (_jsx("th", { className: "border border-wc-default px-4 py-2 text-start font-semibold align-top min-w-[8rem]", children: children })),
        td: ({ children }) => (_jsx("td", { className: "border border-wc-default px-4 py-2 align-top min-w-[8rem] [overflow-wrap:anywhere]", children: children })),
        strong: ({ children }) => (_jsx("strong", { className: "font-semibold text-wc-text-primary", children: children })),
        em: ({ children }) => _jsx("em", { className: "italic", children: children }),
        del: ({ children }) => (_jsx("del", { className: "line-through text-wc-text-faint", children: children })),
    }), [onLinkClick, onFileReferenceClick, onMermaidOpen]);
    if (!content)
        return null;
    const safeContent = typeof content === "string" ? content : String(content);
    // If search is active, wrap rendered output with highlighting
    const rendered = (_jsx(ReactMarkdown, { remarkPlugins: [remarkGfm, remarkProsePaths], components: components, children: safeContent }));
    return (_jsx(MarkdownErrorBoundary, { content: safeContent, children: _jsx("div", { className: `markdown-content min-w-0 max-w-full break-words [overflow-wrap:anywhere] ${className || ""}`, children: rendered }) }));
}, (prevProps, nextProps) => {
    return (prevProps.content === nextProps.content &&
        prevProps.className === nextProps.className &&
        prevProps.onLinkClick === nextProps.onLinkClick &&
        prevProps.onFileReferenceClick === nextProps.onFileReferenceClick &&
        prevProps.onMermaidOpen === nextProps.onMermaidOpen);
});
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
