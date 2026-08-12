import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useState } from "react";
import { Plus, Minus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { FONT_SIZE_MIN, FONT_SIZE_MAX, clampFontSize } from "../../lib/fontSizeUtils";
import { strings } from "../../consts/strings";
import { Button } from "../ui/button";
export default function FontSizeStepper({ currentSize, onChangeSize, testIdPrefix = "appearance", }) {
    const { t } = useTranslation();
    // Draft of the direct-entry field; committed (clamped) on blur/Enter so
    // half-typed values like "" or "2" don't thrash the live preview.
    const [draft, setDraft] = useState(String(currentSize));
    useEffect(() => setDraft(String(currentSize)), [currentSize]);
    const commitDraft = () => {
        const parsed = Number.parseInt(draft, 10);
        if (Number.isNaN(parsed)) {
            setDraft(String(currentSize));
            return;
        }
        const next = clampFontSize(parsed);
        setDraft(String(next));
        if (next !== currentSize)
            onChangeSize(next);
    };
    return (_jsxs("section", { children: [_jsx("h3", { className: "text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2", children: t(strings.appearance.fontSizeHeading) }), _jsxs("div", { className: "flex items-center gap-2", children: [_jsx(Button, { "data-testid": `${testIdPrefix}-font-decrease`, variant: "outline", size: "icon", className: "h-8 w-8", disabled: currentSize <= FONT_SIZE_MIN, onClick: () => onChangeSize(currentSize - 1), children: _jsx(Minus, { className: "h-3 w-3" }) }), _jsxs("div", { className: "flex items-baseline gap-1", children: [_jsx("input", { "data-testid": `${testIdPrefix}-font-value`, type: "text", inputMode: "numeric", className: "h-8 w-12 rounded-md border border-wc-default bg-wc-surface-input text-center font-mono text-sm text-wc-text-primary focus:border-wc-accent focus:outline-none", value: draft, "aria-label": t(strings.appearance.fontSizeInputAriaLabel), onChange: (e) => setDraft(e.target.value), onBlur: commitDraft, onKeyDown: (e) => {
                                    if (e.key === "Enter")
                                        commitDraft();
                                } }), _jsx("span", { className: "text-xs text-wc-text-muted", children: t(strings.appearance.fontSizeUnit) })] }), _jsx(Button, { "data-testid": `${testIdPrefix}-font-increase`, variant: "outline", size: "icon", className: "h-8 w-8", disabled: currentSize >= FONT_SIZE_MAX, onClick: () => onChangeSize(currentSize + 1), children: _jsx(Plus, { className: "h-3 w-3" }) }), _jsx("span", { "data-testid": `${testIdPrefix}-font-sample`, className: "ms-auto font-mono text-wc-text-secondary", style: { fontSize: `${currentSize}px`, lineHeight: 1 }, "aria-hidden": "true", children: "Aa" })] })] }));
}
