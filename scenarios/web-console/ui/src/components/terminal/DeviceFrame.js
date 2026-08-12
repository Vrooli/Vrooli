import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
// [REQ:P0-002e] The SVG is geometric chrome around the leader grid; it never
// attempts to identify the follower's physical hardware.
export function DeviceFrame({ archetype, chromeTier, rect, leaderDevice, gridCols, gridRows, onTakeOver }) {
    const { t } = useTranslation();
    const takeOver = t(strings.deviceFrame.takeOver);
    const label = `${leaderDevice || archetype} · ${gridCols}×${gridRows}`;
    const style = {
        left: rect.x,
        top: rect.y,
        width: rect.width,
        height: rect.height,
        transition: "left 240ms ease, top 240ms ease, width 240ms ease, height 240ms ease",
    };
    // The desktop silhouettes intentionally extend their stand below the
    // rectangular display bounds. Keep the controls clear of that geometry so
    // they remain a caption, rather than looking like part of the monitor.
    const hasDesktopStand = chromeTier === "hairline" && (archetype === "laptop" || archetype === "monitor" || archetype === "ultrawide");
    const controlsClass = chromeTier === "strip"
        ? "pointer-events-auto absolute left-0 top-0 flex h-7 w-full items-center justify-between gap-2 px-2 text-xs"
        : `pointer-events-auto absolute left-1/2 ${hasDesktopStand ? "top-[calc(100%+1.75rem)]" : "top-[calc(100%+0.5rem)]"} flex -translate-x-1/2 items-center gap-2 whitespace-nowrap rounded px-2 py-1 text-xs`;
    return _jsxs("div", { className: "pointer-events-none absolute z-wc-chrome-raised", style: style, "data-testid": `device-frame-${chromeTier}`, children: [_jsxs("svg", { "aria-hidden": "true", className: "absolute inset-0 h-full w-full overflow-visible", viewBox: "0 0 100 100", preserveAspectRatio: "none", children: [chromeTier === "full" && _jsxs(_Fragment, { children: [_jsx("rect", { x: "1", y: "1", width: "98", height: "98", rx: "5", fill: "var(--wc-device-frame-surface)", fillOpacity: "0.26", stroke: "var(--wc-device-frame-line)", vectorEffect: "non-scaling-stroke" }), _jsx("path", { d: "M43 4H57", stroke: "var(--wc-device-frame-line)", strokeWidth: "1.5", strokeLinecap: "round", vectorEffect: "non-scaling-stroke" })] }), chromeTier === "hairline" && _jsxs(_Fragment, { children: [_jsx("rect", { x: "0.75", y: "0.75", width: "98.5", height: "98.5", rx: "3", fill: "none", stroke: "var(--wc-device-frame-line)", vectorEffect: "non-scaling-stroke" }), (archetype === "laptop" || archetype === "monitor" || archetype === "ultrawide") && _jsx("path", { d: "M42 100V104H58V100M35 104H65", stroke: "var(--wc-device-frame-line)", strokeWidth: "1.2", strokeLinecap: "round", vectorEffect: "non-scaling-stroke" })] }), chromeTier === "strip" && _jsx("path", { d: "M0 1H100", stroke: "var(--wc-device-frame-line)", vectorEffect: "non-scaling-stroke" })] }), _jsxs("div", { className: controlsClass, style: { background: "var(--wc-device-frame-surface)", color: "var(--wc-device-frame-text)", border: chromeTier === "strip" ? undefined : "1px solid var(--wc-device-frame-line)" }, children: [_jsx("span", { children: label }), _jsx("button", { type: "button", onClick: onTakeOver, className: "rounded px-2 py-1", style: { border: "1px solid var(--wc-device-frame-line)" }, children: takeOver })] })] });
}
