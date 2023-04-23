import { jsx as _jsx } from "react/jsx-runtime";
import { Typography } from "@mui/material";
const convertToPixels = (size, id) => {
    if (typeof size === "number") {
        return size;
    }
    if (size.includes("px")) {
        return parseFloat(size);
    }
    if (size.includes("rem")) {
        return parseFloat(size) * 16;
    }
    if (size.includes("em")) {
        const el = document.getElementById(id);
        if (el) {
            const fontSize = window.getComputedStyle(el).fontSize;
            return parseFloat(size) * parseFloat(fontSize);
        }
    }
    return 0;
};
const calculateFontSize = (id, minFontSize) => {
    const el = document.getElementById(id);
    if (!el)
        return;
    const minFontSizePx = convertToPixels(minFontSize, id);
    const fontSizePx = convertToPixels(window.getComputedStyle(el).fontSize, id);
    if (el.scrollWidth > el.offsetWidth) {
        let newFontSize = fontSizePx;
        while (el.scrollWidth > el.offsetWidth && newFontSize > minFontSizePx) {
            newFontSize -= 0.25;
            el.style.fontSize = `${newFontSize}px`;
        }
    }
    return el.style.fontSize;
};
export const TextShrink = ({ id, minFontSize = "0.5rem", ...props }) => {
    const fontSize = calculateFontSize(id, minFontSize);
    return (_jsx(Typography, { ...props, id: id, sx: {
            ...(props.sx ?? {}),
            overflow: "visible",
            textOverflow: "unset",
            whiteSpace: "nowrap",
            fontSize,
        } }));
};
//# sourceMappingURL=TextShrink.js.map