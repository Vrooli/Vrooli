import type { ReactNode } from "react";

export type InputVariant = "outline" | "filled" | "underline";
export type InputSize = "sm" | "md" | "lg";

export interface InputContainerProps extends Omit<React.HTMLAttributes<HTMLDivElement>, "onClick"> {
    /** Visual style variant */
    variant?: InputVariant;
    /** Size of the input */
    size?: InputSize;
    /** Whether the input has an error state */
    error?: boolean;
    /** Whether the input is disabled */
    disabled?: boolean;
    /** Whether the input should take full width */
    fullWidth?: boolean;
    /** Whether the input is currently focused (for controlled focus state) */
    focused?: boolean;
    /** Click handler for the container */
    onClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
    /** Focus handler */
    onFocus?: (event: React.FocusEvent<HTMLDivElement>) => void;
    /** Blur handler */
    onBlur?: (event: React.FocusEvent<HTMLDivElement>) => void;
    /** Additional CSS classes */
    className?: string;
    /** The input element(s) to render inside the container */
    children: ReactNode;
    /** Label text for the input */
    label?: string;
    /** Whether the field is required */
    isRequired?: boolean;
    /** Helper text to display below the input */
    helperText?: string | boolean | null | undefined;
    /** Element to display at the start of the input */
    startAdornment?: ReactNode;
    /** Element to display at the end of the input */
    endAdornment?: ReactNode;
    /** ID to associate the label with an input */
    htmlFor?: string;
}
