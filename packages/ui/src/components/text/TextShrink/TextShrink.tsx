import { Typography } from "@mui/material";
import { TextShrinkProps } from "../types";

/**
 * Converts font sizes (rem, em, px) to pixels
 * @param size - font size as string or number
 * @param id - id of element, used to get em size
 * @returns font size number in pixels
 */
const convertToPixels = (size: string | number, id: string): number => {
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

/**
 * Calculates the font size that will fit the text in the container
 */
const calculateFontSize = (id: string, minFontSize: string | number) => {
    const el = document.getElementById(id);
    if (!el) return;
    // Calculate font size and max font size in pixels
    const minFontSizePx = convertToPixels(minFontSize, id);
    const fontSizePx = convertToPixels(window.getComputedStyle(el).fontSize, id);
    // While text is too long, decrease font size
    if (el.scrollWidth > el.offsetWidth) {
        let newFontSize = fontSizePx;
        while (el.scrollWidth > el.offsetWidth && newFontSize > minFontSizePx) {
            newFontSize -= 0.25;
            el.style.fontSize = `${newFontSize}px`;
        }
    }
    return el.style.fontSize;
};

/**
 * Shrinks text to fit its container. 
 * Supports rem, em, and px font sizes, and text can be single or multiline.
 */
export const TextShrink = ({
    id,
    minFontSize = "0.5rem",
    ...props
}: TextShrinkProps) => {
    const fontSize = calculateFontSize(id, minFontSize);

    return (
        <Typography
            {...props}
            id={id}
            sx={{
                ...(props.sx ?? {}),
                color: "inherit",
                overflow: "visible",
                textOverflow: "unset",
                whiteSpace: "nowrap",
                fontSize,
            }}
        />
    );
};
