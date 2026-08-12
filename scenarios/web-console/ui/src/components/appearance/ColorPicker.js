import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
/**
 * @vrooliComponentSource react-component-library:ColorPicker
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 623f00f8-2a74-40ec-83bc-67c4575b6cb6
 * @vrooliComponentAppliedAt 2026-07-22T16:50:28Z
 * @vrooliComponentSourceSha256 11b5610152b0268f567e6237e57cde9af6e82757747cc5205b3ee5a2390e7d5a
 * @vrooliComponentDriftHash 11b5610152b0268f567e6237e57cde9af6e82757747cc5205b3ee5a2390e7d5a
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { Check, Pipette, Plus, X } from "lucide-react";
import { useState } from "react";
import { isHexColor, isLightColor, parseColorValue, serializeColorValue } from "./colorUtils";
import { useDeferredColorCommit } from "./useDeferredColorCommit";
const fallbackColor = "#4dabf7";
export default function ColorPicker({ palette, value, onChange, recentColors = [], onRecordRecent, labels = {}, icons = {}, allowGradient = false, testIdPrefix = "color-picker" }) {
    const { colors, transparent } = parseColorValue(value);
    const [activeSlot, setActiveSlot] = useState(0);
    const { park, flush } = useDeferredColorCommit(onRecordRecent);
    const CheckIcon = icons.check ?? Check;
    const CustomIcon = icons.custom ?? Pipette;
    const AddIcon = icons.add ?? Plus;
    const RemoveIcon = icons.remove ?? X;
    const secondaryOpen = colors.length > 1 || activeSlot === 1;
    const activeColor = colors[activeSlot];
    const customColor = isHexColor(activeColor) ? activeColor : fallbackColor;
    const selected = (color) => !transparent && colors[activeSlot] === color;
    const choose = (color, record = true) => {
        const next = activeSlot === 0 ? [color, colors[1]] : [colors[0] ?? color, color];
        onChange(serializeColorValue(next.filter(isHexColor)));
        if (record)
            onRecordRecent?.(color);
    };
    const mark = (color) => _jsx(CheckIcon, { "aria-hidden": true, className: `h-3.5 w-3.5 ${isLightColor(color) ? "text-app-foreground" : "text-app-background"}` });
    const swatch = (color, suffix) => _jsx("button", { type: "button", "data-testid": `${testIdPrefix}-${suffix}`, className: `flex h-8 w-8 items-center justify-center rounded-full border transition-shadow focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-app-primary ${selected(color) ? "border-app-primary ring-2 ring-app-primary/50" : "border-app-border hover:ring-1 hover:ring-app-muted-foreground"}`, style: { backgroundColor: color }, onClick: () => choose(color), title: color, "aria-label": color, "aria-pressed": selected(color), children: selected(color) ? mark(color) : null }, suffix);
    return _jsxs("section", { className: "space-y-2", "aria-label": labels.heading ?? "Color picker", children: [labels.heading ? _jsx("h3", { className: "text-xs font-semibold uppercase tracking-wider text-app-muted-foreground", children: labels.heading }) : null, allowGradient ? _jsxs("div", { className: "flex items-center gap-1.5", children: [_jsx("button", { type: "button", "data-testid": `${testIdPrefix}-slot-0`, className: "h-8 w-8 rounded-md border border-app-border", style: colors[0] ? { backgroundColor: colors[0] } : undefined, onClick: () => setActiveSlot(0), "aria-label": labels.primary ?? labels.heading ?? "Color picker", "aria-pressed": activeSlot === 0 }), secondaryOpen ? _jsxs(_Fragment, { children: [_jsx("button", { type: "button", "data-testid": `${testIdPrefix}-slot-1`, className: "h-8 w-8 rounded-md border border-app-border", style: colors[1] ? { backgroundColor: colors[1] } : undefined, onClick: () => setActiveSlot(1), "aria-label": labels.secondary ?? labels.heading ?? "Color picker", "aria-pressed": activeSlot === 1 }), _jsx("button", { type: "button", "data-testid": `${testIdPrefix}-remove-gradient`, className: "flex h-8 w-8 items-center justify-center rounded border border-app-border", onClick: () => { setActiveSlot(0); onChange(serializeColorValue(colors.slice(0, 1))); }, "aria-label": labels.removeGradient ?? "Remove gradient", children: _jsx(RemoveIcon, { "aria-hidden": true, className: "h-4 w-4" }) })] }) : _jsxs("button", { type: "button", "data-testid": `${testIdPrefix}-add-gradient`, className: "flex h-8 items-center gap-1 rounded border border-dashed border-app-border px-2 text-xs", onClick: () => setActiveSlot(1), children: [_jsx(AddIcon, { "aria-hidden": true, className: "h-3.5 w-3.5" }), labels.addGradient ?? "Add gradient"] })] }) : null, _jsxs("div", { className: "flex flex-wrap gap-2", children: [_jsx("button", { type: "button", "data-testid": `${testIdPrefix}-transparent`, className: `flex h-8 w-8 items-center justify-center rounded-full border ${transparent ? "border-app-primary ring-2 ring-app-primary/50" : "border-app-border"}`, onClick: () => { setActiveSlot(0); onChange("transparent"); }, "aria-label": labels.transparent ?? "Transparent", "aria-pressed": transparent, children: transparent ? mark() : null }), palette.map((color) => swatch(color, `palette-${color}`)), _jsxs("label", { "data-testid": `${testIdPrefix}-custom`, className: `relative flex h-8 w-8 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-app-border ${isHexColor(activeColor) && !palette.includes(activeColor) ? "ring-2 ring-app-primary/50" : ""}`, style: isHexColor(activeColor) && !palette.includes(activeColor) ? { backgroundColor: activeColor } : undefined, title: labels.custom ?? "Custom color", children: [isHexColor(activeColor) && !palette.includes(activeColor) ? mark(activeColor) : _jsx(CustomIcon, { "aria-hidden": true, className: "h-4 w-4 text-app-muted-foreground" }), _jsx("input", { type: "color", "data-testid": `${testIdPrefix}-custom-input`, className: "absolute inset-0 cursor-pointer opacity-0", value: customColor, onChange: (event) => { choose(event.target.value, false); park(event.target.value); }, onBlur: flush, "aria-label": labels.custom ?? "Custom color" })] })] }), recentColors.length ? _jsxs("div", { "data-testid": `${testIdPrefix}-recents`, children: [_jsx("p", { className: "mb-1 text-xs font-medium text-app-muted-foreground", children: labels.recents ?? "Recent colors" }), _jsx("div", { className: "flex flex-wrap gap-2", children: recentColors.map((color) => swatch(color, `recent-${color}`)) })] }) : null] });
}
