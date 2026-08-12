import { jsx as _jsx } from "react/jsx-runtime";
import { Pressable } from "./Pressable";
export function IconButton({ "aria-label": ariaLabel, children, density = "comfortable", size = "icon", title, pending, disableTooltip = false, type = "button", variant = "ghost", ...props }) {
    return (_jsx(Pressable, { ...props, "aria-label": ariaLabel, type: type, title: disableTooltip ? undefined : title ?? ariaLabel, size: size, density: density, shape: "square", pending: pending, tone: variant, children: children }));
}
