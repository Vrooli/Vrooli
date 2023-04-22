import { IconButton } from "@mui/material";
import { forwardRef } from "react";
import { ColorIconButtonProps } from "../types";

const buttonSx = (background: string, disabled: boolean | undefined) => ({
    background,
    pointerEvents: disabled ? "none" : "auto",
    filter: disabled ? "grayscale(1) opacity(0.5)" : "none",
    transition: "filter 0.2s ease-in-out",
    "&:hover": {
        background,
        filter: disabled ? "grayscale(1) opacity(0.5)" : "brightness(1.2)",
    },
});

/**
 * IconButton with a custom color
 */
export const ColorIconButton = forwardRef<HTMLButtonElement | HTMLAnchorElement, ColorIconButtonProps>(({
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
                ...buttonSx(background, disabled),
                ...(sx ?? {}),
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
                ...buttonSx(background, disabled),
                ...(sx ?? {}),
            }}
        >
            {children}
        </IconButton>
    );
});
