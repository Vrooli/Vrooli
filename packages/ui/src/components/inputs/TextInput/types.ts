import { InputHTMLAttributes, TextareaHTMLAttributes, RefObject } from "react";

export type TextInputVariant = "outline" | "filled" | "underline";
export type TextInputSize = "sm" | "md" | "lg";

type CommonInputProps = {
    className?: string;
    disabled?: boolean;
    placeholder?: string;
    value?: string;
    defaultValue?: string;
    name?: string;
    id?: string;
    autoFocus?: boolean;
    autoComplete?: string;
    onBlur?: React.FocusEventHandler<HTMLInputElement | HTMLTextAreaElement>;
    onChange?: React.ChangeEventHandler<HTMLInputElement | HTMLTextAreaElement>;
    onFocus?: React.FocusEventHandler<HTMLInputElement | HTMLTextAreaElement>;
    onKeyDown?: React.KeyboardEventHandler<HTMLInputElement | HTMLTextAreaElement>;
    rows?: number;
    cols?: number;
    maxLength?: number;
    minLength?: number;
    readOnly?: boolean;
    required?: boolean;
    tabIndex?: number;
    type?: string;
};

export interface TailwindTextInputBaseProps extends CommonInputProps {
    /** Visual style variant */
    variant?: TextInputVariant;
    /** Size of the input */
    size?: TextInputSize;
    /** Whether the input has an error state */
    error?: boolean;
    /** Whether the input should take full width */
    fullWidth?: boolean;
    /** Additional CSS classes */
    className?: string;
    /** Helper text to display below the input */
    helperText?: string | boolean | null | undefined;
    /** Label text for the input */
    label?: string;
    /** Whether the field is required */
    isRequired?: boolean;
    /** Whether Enter key will submit */
    enterWillSubmit?: boolean;
    /** Function to call on submit (when Enter is pressed) */
    onSubmit?: () => unknown;
    /** Whether to render as textarea */
    multiline?: boolean;
    /** Element to display at the start of the input */
    startAdornment?: React.ReactNode;
    /** Element to display at the end of the input */
    endAdornment?: React.ReactNode;
    /** Ref for the input element */
    ref?: RefObject<HTMLInputElement | HTMLTextAreaElement>;
}

export type TailwindTextInputProps = Omit<TailwindTextInputBaseProps, "ref" | "value" | "onChange" | "onBlur" | "error" | "helperText" | "name"> & {
    /** Field name for Formik */
    name: string;
    /** Validation function for Formik */
    validate?: (value: any) => string | undefined;
    /** Ref for the input element */
    ref?: RefObject<HTMLElement>;
}

export interface TranslatedTailwindTextInputProps {
    /** Visual style variant */
    variant?: TextInputVariant;
    /** Size of the input */
    size?: TextInputSize;
    /** Whether the input should take full width */
    fullWidth?: boolean;
    /** Whether the field is required */
    isRequired?: boolean;
    /** Label text for the input */
    label?: string;
    /** Current language for translation */
    language: string;
    /** Field name */
    name: string;
    /** Placeholder text */
    placeholder?: string;
    /** Whether to allow multiline input */
    multiline?: boolean;
    /** Maximum number of rows for multiline */
    maxRows?: number;
    /** Minimum number of rows for multiline */
    minRows?: number;
    /** Auto focus the input */
    autoFocus?: boolean;
}