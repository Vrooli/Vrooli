import Button from "@mui/material/Button";
import type { ButtonProps } from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import { useCallback } from "react";
import { TailwindButton } from "./TailwindButton.js";
import type { ButtonVariant } from "./TailwindButton.js";

type HybridLoadableButtonProps = Omit<ButtonProps, "loading"> & {
    isLoading: boolean;
    useTailwind?: boolean;
}

const containedSpinnerStyle = { color: "white" } as const;

/**
 * Hybrid button component that can use either MUI or Tailwind CSS.
 * This demonstrates the migration approach during the transition period.
 */
export function HybridLoadableButton({
    children,
    disabled = false,
    isLoading = false,
    startIcon,
    variant = "contained",
    useTailwind = false,
    className,
    onClick,
    ...props
}: HybridLoadableButtonProps) {
    const getStartIcon = useCallback(function getStartIconCallback() {
        if (isLoading) {
            return <CircularProgress size={24} sx={variant === "contained" ? containedSpinnerStyle : undefined} />;
        }
        if (startIcon) {
            return startIcon;
        }
        return null;
    }, [isLoading, startIcon, variant]);

    // If using Tailwind, use our professional TailwindButton component
    if (useTailwind) {
        // Map MUI variants to TailwindButton variants
        const tailwindVariant: ButtonVariant = 
            variant === "contained" ? "primary" :
            variant === "outlined" ? "outline" :
            variant === "text" ? "ghost" : "primary";

        return (
            <TailwindButton
                variant={tailwindVariant}
                isLoading={isLoading}
                disabled={disabled}
                startIcon={!isLoading ? startIcon : undefined}
                fullWidth
                className={className}
                onClick={onClick}
                {...props}
            >
                {children}
            </TailwindButton>
        );
    }

    // Default: Use MUI Button
    return (
        <Button
            disabled={disabled || isLoading}
            fullWidth
            startIcon={getStartIcon()}
            variant={variant}
            className={className}
            onClick={onClick}
            {...props}
        >{children}</Button>
    );
}