import { cn } from "../../utils/tailwind-theme.js";
import { FormControlLabelProps } from "./types.js";

interface FormControlLabelStyleOptions {
    labelPlacement?: FormControlLabelProps["labelPlacement"];
    disabled?: boolean;
    required?: boolean;
    sx?: FormControlLabelProps["sx"];
}

const placementClasses = {
    end: {
        root: "tw-flex-row",
        controlWrapper: "tw-mr-2",
    },
    start: {
        root: "tw-flex-row-reverse",
        controlWrapper: "tw-ml-2",
    },
    top: {
        root: "tw-flex-col-reverse",
        controlWrapper: "tw-mt-1",
    },
    bottom: {
        root: "tw-flex-col",
        controlWrapper: "tw-mb-1",
    },
};

export function getFormControlLabelStyles({
    labelPlacement = "end",
    disabled = false,
    required = false,
    sx,
}: FormControlLabelStyleOptions) {
    const placementStyles = placementClasses[labelPlacement];

    return {
        root: cn(
            // Base styles
            "tw-inline-flex tw-items-center tw-cursor-pointer",
            "tw-select-none tw-relative",
            
            // Placement-specific styles
            placementStyles.root,
            
            // State styles
            disabled && "tw-cursor-not-allowed tw-opacity-60",
            
            // Transitions
            "tw-transition-opacity tw-duration-200",
        ),
        
        controlWrapper: cn(
            "tw-inline-flex tw-items-center tw-justify-center",
            placementStyles.controlWrapper,
        ),
        
        labelText: cn(
            // Typography
            "tw-text-base tw-leading-normal",
            "tw-text-text-primary",
            
            // State styles
            disabled && "tw-text-text-secondary",
            
            // User select
            "tw-select-none",
        ),
        
        asterisk: cn(
            "tw-text-danger-main",
            "tw-ml-0.5",
            disabled && "tw-opacity-60",
        ),
    };
}