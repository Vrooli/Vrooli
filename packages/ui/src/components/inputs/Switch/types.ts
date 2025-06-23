import type { InputHTMLAttributes, ReactNode } from "react";

// Export types for external use
export type SwitchVariant = "default" | "success" | "warning" | "danger" | "space" | "neon" | "theme" | "custom";
export type SwitchSize = "sm" | "md" | "lg";
export type LabelPosition = "left" | "right" | "none";

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
}
