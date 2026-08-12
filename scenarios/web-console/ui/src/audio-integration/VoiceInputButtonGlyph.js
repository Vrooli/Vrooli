import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
const paths = {
    mic: (_jsxs(_Fragment, { children: [_jsx("rect", { x: "8", y: "3", width: "8", height: "12", rx: "4" }), _jsx("path", { d: "M5 11a7 7 0 0 0 14 0M12 21v-4M9 21h6" })] })),
    loader: (_jsxs(_Fragment, { children: [_jsx("path", { d: "M12 2a10 10 0 1 0 10 10" }), _jsx("path", { d: "M12 2v4" })] })),
    alert: (_jsxs(_Fragment, { children: [_jsx("path", { d: "M10.3 3.7 2.2 18a2 2 0 0 0 1.7 3h16.2a2 2 0 0 0 1.7-3L13.7 3.7a2 2 0 0 0-3.4 0Z" }), _jsx("path", { d: "M12 9v4M12 17h.01" })] })),
};
export function VoiceInputButtonGlyph({ kind, className = "", ...props }) {
    return (_jsx("svg", { ...props, "aria-hidden": "true", className: className, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: "2", strokeLinecap: "round", strokeLinejoin: "round", children: paths[kind] }));
}
