/**
 * Text that shrinks to fit its container.
 */

import { Typography } from "@mui/material";
import { useEffect } from "react";
import { TextShrinkProps } from "../types";

export const TextShrink = ({
    id,
    ...props
}: TextShrinkProps) => {

    // Change font size to fit container
    useEffect(() => {
        const el = document.getElementById(id);
        if (!el) return;
        const shrink = () => {
            const { clientWidth } = el;
            const { scrollWidth } = el;
            if (clientWidth < scrollWidth) {
                el.style.fontSize = `${(clientWidth / scrollWidth) * 100}%`;
            }
        }
        shrink();
    } , [id]);
    
    return (
        <Typography
            {...props}
            id={id}
            sx={{
                ...(props.sx ?? {}),
                overflow: "visible",
                textOverflow: "unset",
                whiteSpace: "nowrap",
            }}
        />
    )
}