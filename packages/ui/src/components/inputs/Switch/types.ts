import type { InputHTMLAttributes, ReactNode } from "react";
import type { IconInfo } from "../../../icons/Icons.js";

// Export types for external use
export type SwitchVariant = "default" | "success" | "warning" | "danger" | "space" | "neon" | "theme" | "custom";
export type SwitchSize = "sm" | "md" | "lg";
export type LabelPosition = "left" | "right" | "none";

/** Base props for SwitchBase component - no Formik dependencies */
export interface SwitchBaseProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "size" | "type" | "onChange"> {
    /** Visual style variant of the switch */
    variant?: SwitchVariant;
    /** Size of the switch */
    size?: SwitchSize;
    /** Position of the label relative to the switch */
    labelPosition?: LabelPosition;
    /** Label text to display */
    label?: ReactNode;
    /** Custom color for the custom variant (hex, rgb, hsl, etc.) */
    color?: string;
    /** Whether the switch is checked */
    checked?: boolean;
    /** Whether the switch is disabled */
    disabled?: boolean;
    /** Additional CSS classes */
    className?: string;
    /** Change handler */
    onChange?: (checked: boolean, event: React.ChangeEvent<HTMLInputElement>) => void;
    /** Icon to display when switch is off */
    offIcon?: IconInfo | null | undefined;
    /** Icon to display when switch is on */
    onIcon?: IconInfo | null | undefined;
    /** Tooltip text to display on hover */
    tooltip?: string;
    /** Error state */
    error?: boolean;
    /** Helper text to display below switch */
    helperText?: ReactNode;
}

/** Props for Formik-integrated Switch component */
export interface SwitchFormikProps extends Omit<SwitchBaseProps, "checked" | "onChange" | "error" | "helperText"> {
    /** Formik field name */
    name: string;
    /** Optional validation function */
    validate?: (value: boolean) => string | void | Promise<string | void>;
}

/** Legacy SwitchProps for backward compatibility */
export interface SwitchProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "size" | "type"> {
    /** Visual style variant of the switch */
    variant?: SwitchVariant;
    /** Size of the switch */
    size?: SwitchSize;
    /** Position of the label relative to the switch */
    labelPosition?: LabelPosition;
    /** Label text to display */
    label?: ReactNode;
    /** Custom color for the custom variant (hex, rgb, hsl, etc.) */
    color?: string;
    /** Whether the switch is checked */
    checked?: boolean;
    /** Whether the switch is disabled */
    disabled?: boolean;
    /** Additional CSS classes */
    className?: string;
    /** Change handler */
    onChange?: (checked: boolean, event: React.ChangeEvent<HTMLInputElement>) => void;
    /** Icon to display when switch is off */
    offIcon?: IconInfo | null | undefined;
    /** Icon to display when switch is on */
    onIcon?: IconInfo | null | undefined;
    /** Tooltip text to display on hover */
    tooltip?: string;
}

// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed validate function parameter type from 'any' to 'boolean' for switch values
