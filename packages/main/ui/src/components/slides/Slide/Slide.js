import { jsx as _jsx } from "react/jsx-runtime";
import { SlideContainer } from "../SlideContainer/SlideContainer";
import { SlideContent } from "../SlideContent/SlideContent";
export const Slide = ({ id, children, sx, }) => {
    return (_jsx(SlideContainer, { id: id, sx: sx, children: _jsx(SlideContent, { children: children }) }));
};
//# sourceMappingURL=Slide.js.map