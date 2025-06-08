import Button from "@mui/material/Button";
import type { ButtonProps } from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import { useCallback } from "react";

type LoadableButtonProps = Omit<ButtonProps, "loading"> & {
    isLoading: boolean;
}

const containedSpinnerStyle = { color: "white" } as const;

/**
 * Button with a loading spinner that replaces the start icon when loading
 */
export function LoadableButton({
    children,
    disabled = false,
    isLoading = false,
    startIcon,
    variant,
    ...props
}: LoadableButtonProps) {
    const getStartIcon = useCallback(function getStartIconCallback() {
        if (isLoading) {
            return <CircularProgress size={24} sx={variant === "contained" ? containedSpinnerStyle : undefined} />;
        }
        if (startIcon) {
            return startIcon;
        }
        return null;
    }, [isLoading, startIcon, variant]);

    return (
        <Button
            disabled={disabled || isLoading}
            fullWidth
            startIcon={getStartIcon()}
            variant={variant}
            {...props}
        >{children}</Button>
    );
}
