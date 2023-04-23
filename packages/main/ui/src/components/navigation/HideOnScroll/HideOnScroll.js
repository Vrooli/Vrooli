import { jsx as _jsx } from "react/jsx-runtime";
import { Slide, useScrollTrigger } from "@mui/material";
export const HideOnScroll = ({ children, }) => {
    const trigger = useScrollTrigger();
    return (_jsx(Slide, { appear: false, direction: "down", in: !trigger, children: children }));
};
//# sourceMappingURL=HideOnScroll.js.map