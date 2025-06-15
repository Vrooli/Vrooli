import { cn } from "../../utils/tailwind-theme.js";
import { FormGroupProps } from "./types.js";

interface FormGroupStyleOptions {
    row?: boolean;
    sx?: FormGroupProps["sx"];
}

export function getFormGroupStyles({
    row = false,
    sx,
}: FormGroupStyleOptions) {
    return {
        root: cn(
            // Base styles
            "tw-flex",
            
            // Direction
            row ? "tw-flex-row tw-flex-wrap" : "tw-flex-col",
            
            // Spacing between items
            row ? "tw-gap-x-6 tw-gap-y-2" : "tw-gap-y-3",
            
            // Alignment
            row ? "tw-items-center" : "tw-items-start",
        ),
    };
}