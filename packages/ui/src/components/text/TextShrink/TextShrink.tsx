import { Typography } from "@mui/material";
import { fontSizeToPixels } from "utils/display/stringTools";
import { TextShrinkProps } from "../types";

/**
 * Calculates the font size that will fit the text in the container
 */
const calculateFontSize = (id: string, minFontSize: string | number) => {
    const el = document.getElementById(id);
    if (!el) return;
    // Calculate font size and max font size in pixels
    const minFontSizePx = fontSizeToPixels(minFontSize, id);
    const fontSizePx = fontSizeToPixels(window.getComputedStyle(el).fontSize, id);
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
