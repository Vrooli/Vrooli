import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
/**
 * @vrooliComponentSource react-component-library:Pressable
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption fca0af9a-3a97-46e6-b43a-b8c6504d9361
 * @vrooliComponentAppliedAt 2026-08-09T14:56:08Z
 * @vrooliComponentSourceSha256 c602c0925568c371342c5099f747059a4e4804392e532f5c10e630d7ae3d7532
 * @vrooliComponentDriftHash 3c9f0b0a5da5221d46540c17755633a801b6b6f1ea5a8b08e736cf0a4d306ef5
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { forwardRef } from "react";
import { ControlBase, } from "./ControlBase";
const pressableStyles = `
[data-rcl-pressable-content] { display: inline-grid; place-items: center; min-inline-size: 0; }
[data-rcl-pressable-label], [data-rcl-pressable-pending] { grid-area: 1 / 1; display: inline-flex; align-items: center; gap: var(--space-2xs); }
[data-rcl-pressable-pending] { visibility: hidden; }
[data-rcl-pressable][data-rcl-pending="true"] [data-rcl-pressable-label] { visibility: hidden; }
[data-rcl-pressable][data-rcl-pending="true"] [data-rcl-pressable-pending] { visibility: visible; }
[data-rcl-pressable-spinner] { inline-size: var(--space-sm); block-size: var(--space-sm); flex: 0 0 auto; border: var(--border-strong) solid color-mix(in srgb, currentColor 28%, transparent); border-block-start-color: currentColor; border-radius: var(--radius-pill); animation: rcl-pressable-spin var(--dur-moderate) linear infinite; }
@keyframes rcl-pressable-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-pressable-spinner] { animation: none; } }
`;
export const Pressable = forwardRef(function Pressable({ pending = false, pendingLabel = "Working…", tone = "primary", children, disabled, "aria-busy": ariaBusy, "aria-disabled": ariaDisabled, size, density, shape, style, ...props }, ref) {
    const variant = tone;
    return (_jsxs(_Fragment, { children: [_jsx("style", { "data-rcl-pressable-styles": true, dangerouslySetInnerHTML: { __html: pressableStyles } }), _jsx(ControlBase, { ...props, disabled: disabled || pending, "aria-busy": ariaBusy ?? (pending || undefined), "aria-disabled": ariaDisabled ?? (disabled || pending || undefined), variant: variant, size: size, density: density, shape: shape, "data-rcl-pressable": "true", "data-rcl-pending": pending ? "true" : "false", ref: ref, style: style, children: _jsxs("span", { "data-rcl-pressable-content": true, children: [_jsx("span", { "data-rcl-pressable-label": true, "aria-hidden": pending, children: children }), _jsxs("span", { "data-rcl-pressable-pending": true, "aria-hidden": !pending, "aria-live": "polite", children: [_jsx("span", { "aria-hidden": "true", "data-rcl-pressable-spinner": true }), _jsx("span", { children: pendingLabel })] })] }) })] }));
});
