import { IconButton } from "@mui/material";
import { forwardRef } from "react";
import { highlightStyle } from "styles";

/**
 * IconButton with a custom color
 */
export const ColorIconButton = forwardRef<HTMLButtonElement | HTMLAnchorElement, any>(({
    background,
    children,
    disabled,
    href,
    onClick,
    sx,
    ...props
}, ref) => {
    // If href is provided, use an anchor tag
    if (href) return (
        <IconButton
            component="a"
            href={href}
            onClick={onClick}
            sx={{
                ...highlightStyle(background, disabled),
                ...sx,
            }}
        >
            {children}
        </IconButton>
    );
    // Otherwise, treat as a normal button
    return (
        <IconButton
            {...props}
            onClick={onClick}
            sx={{
                ...highlightStyle(background, disabled),
                ...sx,
            }}
        >
            {children}
        </IconButton>
    );
});
