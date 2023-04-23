import { jsx as _jsx } from "react/jsx-runtime";
export const SvgBase = ({ props, children }) => (_jsx("svg", { id: props.id, style: props.style, width: !props.width ? "24px" : props.width === "unset" ? "5000px" : props.width, height: !props.height ? "24px" : props.height === "unset" ? "5000px" : props.height, viewBox: "0 0 24 24", xmlns: "http://www.w3.org/2000/svg", children: children }));
export const SvgPath = ({ d, props }) => (_jsx("svg", { id: props.id, style: props.style, width: !props.width ? "24px" : props.width === "unset" ? "5000px" : props.width, height: !props.height ? "24px" : props.height === "unset" ? "5000px" : props.height, pointerEvents: "none", viewBox: "0 0 24 24", xmlns: "http://www.w3.org/2000/svg", children: _jsx("path", { pointerEvents: "none", style: {
            fill: props.fill ?? "white",
            fillOpacity: 1,
        }, d: d }) }));
//# sourceMappingURL=base.js.map