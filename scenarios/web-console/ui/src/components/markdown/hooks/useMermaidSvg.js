import { useEffect, useRef, useState } from "react";
// Lazy-loaded mermaid instance (singleton). Both the inline diagram and the
// full-screen viewer share this so api/internal/<domain>/theme is one source of truth.
let mermaidPromise = null;
function getMermaid() {
    if (!mermaidPromise) {
        mermaidPromise = import("mermaid")
            .then((mod) => {
            const mermaid = mod.default;
            mermaid.initialize({
                startOnLoad: false,
                theme: "dark",
                securityLevel: "strict",
                fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace",
                themeVariables: {
                    primaryColor: "#334155",
                    primaryTextColor: "#e2e8f0",
                    primaryBorderColor: "#475569",
                    lineColor: "#6366f1",
                    secondaryColor: "#1e293b",
                    tertiaryColor: "#0f172a",
                    noteBkgColor: "#1e293b",
                    noteTextColor: "#e2e8f0",
                    noteBorderColor: "#475569",
                },
            });
            return mermaid;
        })
            .catch((err) => {
            mermaidPromise = null;
            throw err;
        });
    }
    return mermaidPromise;
}
/**
 * useMermaidSvg renders Mermaid source to SVG markup using the shared singleton
 * instance. It debounces rapid `code` changes and cancels stale renders, so the
 * latest source always wins. `loading` is true while a non-empty diagram is
 * being rendered and no result/error is available yet.
 */
export function useMermaidSvg(code) {
    const [svgHtml, setSvgHtml] = useState(null);
    const [error, setError] = useState(null);
    const [loading, setLoading] = useState(() => code.trim().length > 0);
    const debounceRef = useRef(null);
    useEffect(() => {
        let cancelled = false;
        setError(null);
        if (!code.trim()) {
            setSvgHtml(null);
            setLoading(false);
            return;
        }
        setLoading(true);
        if (debounceRef.current)
            clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            async function render() {
                try {
                    const mermaid = await getMermaid();
                    if (cancelled)
                        return;
                    const id = `mermaid-${crypto.randomUUID()}`;
                    const { svg } = await mermaid.render(id, code);
                    if (cancelled)
                        return;
                    setSvgHtml(svg);
                    setError(null);
                }
                catch (err) {
                    if (cancelled)
                        return;
                    setError(err instanceof Error ? err.message : "Failed to render diagram");
                    setSvgHtml(null);
                }
                finally {
                    if (!cancelled)
                        setLoading(false);
                }
            }
            void render();
        }, 100);
        return () => {
            cancelled = true;
            if (debounceRef.current)
                clearTimeout(debounceRef.current);
        };
    }, [code]);
    return { svgHtml, error, loading };
}
