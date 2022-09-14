/**
 * Text that shrinks to fit its container.
 */

import { Typography } from "@mui/material";
import { useMemo } from "react";
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
}

export const TextShrink = ({
    id,
    minFontSize = '0.5rem',
    ...props
}: TextShrinkProps) => {

    // If text is too long find the largest font size that fits
    const fontSize = useMemo(() => {
        const el = document.getElementById(id);
        if (!el) return;
        // Calculate font size and max font size in pixels
        const minFontSizePx = convertToPixels(minFontSize, id);
        const fontSizePx = convertToPixels(window.getComputedStyle(el).fontSize, id);
        console.log('finding font size', fontSizePx, minFontSizePx, el.scrollWidth, el.offsetWidth)
        // While text is too long, decrease font size
        if (el.scrollWidth > el.offsetWidth) {
            let newFontSize = fontSizePx;
            console.log('decreasing font size', newFontSize)
            while (el.scrollWidth > el.offsetWidth && newFontSize > minFontSizePx) {
                newFontSize -= 0.5;
                el.style.fontSize = `${newFontSize}px`;
            }
            console.log('final font size', newFontSize)
        }
        return el.style.fontSize;
    }, [id, minFontSize]);

    return (
        <Typography
            {...props}
            id={id}
            sx={{
                ...(props.sx ?? {}),
                overflow: "visible",
                textOverflow: "unset",
                whiteSpace: "nowrap",
                fontSize: fontSize,
            }}
        />
    )
}