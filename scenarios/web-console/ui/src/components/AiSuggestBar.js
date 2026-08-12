import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState, useCallback, useRef, useEffect } from "react";
import { Sparkles, Loader2, AlertCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { generateAISuggestions } from "../api/ai";
import { strings } from "../consts/strings";
import { toErrorInfo } from "../lib/errors";
const DEBOUNCE_MS = 600;
/**
 * Inline AI suggestion bar for mobile. Sits above the MobileToolbar
 * (same visual pattern as AudioPlayerBar). Watches the textarea input
 * and shows 1–3 tappable command suggestions after a debounce.
 */
export default function AiSuggestBar({ inputText, onExecute, onClose: _onClose }) {
    const { t } = useTranslation();
    const [suggestions, setSuggestions] = useState([]);
    const [provider, setProvider] = useState(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState(null);
    const generationIdRef = useRef(0);
    const debounceRef = useRef(null);
    // Cleanup on unmount
    useEffect(() => {
        return () => {
            generationIdRef.current += 1;
            if (debounceRef.current !== null)
                clearTimeout(debounceRef.current);
        };
    }, []);
    // Debounced suggestion fetch when inputText changes
    useEffect(() => {
        if (debounceRef.current !== null)
            clearTimeout(debounceRef.current);
        const trimmed = inputText.trim();
        if (!trimmed) {
            setSuggestions([]);
            setProvider(null);
            setError(null);
            setIsLoading(false);
            return;
        }
        setIsLoading(true);
        debounceRef.current = setTimeout(() => {
            const thisGeneration = ++generationIdRef.current;
            generateAISuggestions(trimmed)
                .then((result) => {
                if (generationIdRef.current !== thisGeneration)
                    return;
                setSuggestions(result.commands);
                setProvider(result.provider);
                setError(null);
            })
                .catch((err) => {
                if (generationIdRef.current !== thisGeneration)
                    return;
                const info = toErrorInfo(err);
                setError(info.message);
                setSuggestions([]);
                setProvider(null);
            })
                .finally(() => {
                if (generationIdRef.current === thisGeneration) {
                    setIsLoading(false);
                }
            });
        }, DEBOUNCE_MS);
    }, [inputText]);
    const handleChipTap = useCallback((command) => {
        onExecute(command + "\n");
    }, [onExecute]);
    const isEmpty = !inputText.trim();
    return (_jsxs("div", { "data-testid": "ai-suggest-bar", className: "flex items-center gap-2 border-t border-wc-default bg-wc-surface-raised px-2 py-1.5 animate-in slide-in-from-bottom-2 duration-200 touch-manipulation select-none", onMouseDown: (e) => e.preventDefault(), children: [_jsx(Sparkles, { className: "h-3.5 w-3.5 shrink-0 text-wc-accent" }), isEmpty && (_jsx("span", { className: "flex-1 text-xs text-wc-text-muted truncate", children: t(strings.aiSuggestBar.empty) })), !isEmpty && isLoading && (_jsxs("div", { className: "flex flex-1 items-center gap-1.5", children: [_jsx(Loader2, { className: "h-3 w-3 animate-spin text-wc-text-muted" }), _jsx("span", { className: "text-xs text-wc-text-muted", children: t(strings.aiSuggestBar.generating) })] })), !isEmpty && !isLoading && error && (_jsxs("div", { className: "flex flex-1 items-center gap-1.5 min-w-0", children: [_jsx(AlertCircle, { className: "h-3 w-3 shrink-0 text-wc-error-detail" }), _jsx("span", { className: "text-xs text-wc-error-detail truncate", children: error })] })), !isEmpty && !isLoading && !error && suggestions.length > 0 && (_jsxs("div", { className: "flex flex-1 items-center gap-1.5 overflow-x-auto min-w-0", children: [suggestions.map((cmd, i) => (_jsx("button", { "data-testid": `ai-suggest-chip-${i}`, tabIndex: -1, onPointerDown: (e) => e.preventDefault(), onClick: () => handleChipTap(cmd), className: "shrink-0 rounded border border-wc-accent/40 bg-wc-accent/10 px-2 py-1 text-xs font-mono text-wc-text-primary transition active:bg-wc-accent/25 touch-manipulation max-w-[70vw] truncate", title: cmd, children: cmd }, i))), provider && (_jsx("span", { className: "shrink-0 text-[10px] text-wc-text-faint", children: t(strings.aiSuggestBar.viaProvider, { provider }) }))] })), !isEmpty && !isLoading && !error && suggestions.length === 0 && (_jsx("span", { className: "flex-1 text-xs text-wc-text-muted truncate", children: t(strings.aiSuggestBar.noSuggestions) }))] }));
}
