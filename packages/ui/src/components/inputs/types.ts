/* c8 ignore start */
import type { BoxProps, CheckboxProps, TextFieldProps } from "@mui/material";
import { type CodeLanguage, type JSONVariable, type ResourceListFor, type ResourceVersion, type Tag, type TagShape, type TranslationFunc } from "@vrooli/shared";
import { type FieldProps } from "formik";
import { type CSSProperties, type RefObject } from "react";
import { type UseChatTaskReturn } from "../../hooks/tasks.js";
import { type IconInfo } from "../../icons/Icons.js";
import { type SxType } from "../../types.js";
import { type FindObjectType } from "../dialogs/types.js";
import { type ResourceListProps } from "../lists/types.js";
import { type InputVariant, type InputSize } from "./InputContainer/types.js";

export type CheckboxInputProps = Omit<(CheckboxProps & FieldProps), "form"> & {
    label: string;
};

/** Base props for CodeInputBase component - no Formik dependencies */
export interface CodeInputBaseProps {
    /**
     * The current language of the code input
     */
    codeLanguage: CodeLanguage;
    /**
     * The code itself
     */
    content: string;
    defaultValue?: string;
    disabled?: boolean;
    /**
     * NOTE: Only applicable for JSON standards.
     * 
     * JSON string representing the format that the 
     * input should follow. 
     * Any key/value that's a plain string (e.g. "name", "age") 
     * must appear in the input value exactly as shown. 
     * Any key/value that's wrapped in <> (e.g. <name>, <age>) 
     * refers to a variable.
     * Any key that starts with a ? is optional.
     * Any key that's wrapped in [] (e.g. [x]) 
     * means that additional keys can be added to the object. The value of these 
     * keys is also wrapped in [] (e.g. [string], [any], [number | string])
     * to specify the types of values allowed.
     * ex: {
     * "<721>": {
     *   "<policy_id>": {
     *     "<asset_name>": {
     *       "name": "<asset_name>",
     *       "image": "<ipfs_link>",
     *       "?mediaType": "<mime_type>",
     *       "?description": "<description>",
     *       "?files": [
     *         {
     *          "name": "<asset_name>",
     *           "mediaType": "<mime_type>",
     *           "src": "<ipfs_link>"
     *         }
     *       ],
     *       "[x]": "[any]"
     *     }
     *   },
     *   "version": "1.0"
     *  }
     * }
     */
    format?: string;
    handleCodeLanguageChange: (language: CodeLanguage) => unknown;
    handleContentChange: (content: string) => unknown;
    /**
     * Limit the languages that can be selected in the language dropdown.
     */
    limitTo?: readonly CodeLanguage[];
    name: string;
    /**
     * Dictionary which describes variables (e.g. <name>, <age>) in
     * the format JSON. 
     * Each variable can have a label, helper text, a type, and accepted values.
     * ex: {
     *  "policy_id": {
     *      "label": "Policy ID",
     *      "helperText": "Human-readable name of the policy.",
     *  },
     *  "721": {
     *      "label": "721",
     *      "helperText": "The transaction_metadatum_label that describes the type of data. 
     *      721 is the number for NFTs.",
     *   }
     * }
     */
    variables?: { [x: string]: JSONVariable };
}

/** Props for Formik-integrated CodeInput component */
export interface CodeInputFormikProps {
    /** Name of the content field */
    name?: string;
    /** Name of the code language field (optional) */
    codeLanguageField?: string;
    /** Name of the default value field (optional) */
    defaultValueField?: string;
    /** Name of the format field (optional) */
    formatField?: string;
    /** Name of the variables field (optional) */
    variablesField?: string;
    /** Whether the input is disabled */
    disabled?: boolean;
    /** Limit the languages that can be selected */
    limitTo?: readonly CodeLanguage[];
}

/** Legacy CodeInputProps for backward compatibility */
export type CodeInputProps = Omit<CodeInputBaseProps, "codeLanguage" | "content" | "defaultValue" | "format" | "handleCodeLanguageChange" | "handleContentChange" | "variables"> & {
    codeLanguageField?: string;
    disabled?: boolean;
    limitTo?: readonly CodeLanguage[];
}

/** Base props for DateInputBase component - no Formik dependencies */
export interface DateInputBaseProps {
    /** Whether the input is required */
    isRequired?: boolean;
    /** Label for the date input */
    label: string;
    /** Name attribute for the input */
    name: string;
    /** Type of date input */
    type?: "date" | "datetime-local";
    /** Current value */
    value: string;
    /** Change handler */
    onChange: (value: string) => void;
    /** Blur handler */
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Whether the input has an error */
    error?: boolean;
    /** Helper text to display below input */
    helperText?: string | boolean | null | undefined;
    /** Additional CSS classes */
    className?: string;
    /** MUI sx prop for custom styling */
    sx?: SxType;
    /** Disabled state */
    disabled?: boolean;
    /** Test ID for testing */
    "data-testid"?: string;
}

/** Props for Formik-integrated DateInput component */
export interface DateInputFormikProps extends Omit<DateInputBaseProps, "value" | "onChange" | "error" | "helperText" | "onBlur"> {
    /** Optional validation function */
    validate?: (value: unknown) => string | void | Promise<string | void>;
}

/** Legacy DateInputProps for backward compatibility */
export interface DateInputProps {
    isRequired?: boolean;
    label: string;
    name: string;
    type?: "date" | "datetime-local";
}

export interface DropzoneProps {
    acceptedFileTypes?: string[];
    cancelText?: string;
    disabled?: boolean;
    dropzoneText?: string;
    maxFiles?: number;
    onUpload: (files: File[]) => unknown;
    showThumbs?: boolean;
    uploadText?: string;
}

export type IntegerInputBaseProps = {
    allowDecimal?: boolean;
    autoFocus?: boolean;
    disabled?: boolean;
    error?: boolean;
    fullWidth?: boolean;
    helperText?: string | boolean | null | undefined;
    key?: string;
    initial?: number;
    label?: string;
    max?: number;
    min?: number;
    name: string;
    offset?: number;
    onBlur?: (event: React.FocusEvent<HTMLElement>) => unknown;
    onChange: (newValue: number) => unknown;
    size?: InputSize;
    step?: number;
    sx?: SxType;
    tooltip?: string;
    value: number;
    variant?: InputVariant;
    /** If provided, displays this text instead of 0 */
    zeroText?: string;
}

export type IntegerInputProps = Omit<IntegerInputBaseProps, "onChange" | "value">;

export type LanguageInputProps = {
    /**
     * Currently-selected language, if using this component to add/edit an object
     * with translations. Not needed if using this component to select languages 
     * for an advanced search, for example.
     */
    currentLanguage: string;
    disabled?: boolean;
    flexDirection?: "row" | "row-reverse";
    handleAdd: (language: string) => unknown;
    handleDelete: (language: string) => unknown;
    handleCurrent: (language: string) => unknown;
    /**
     * All languages that currently have translations for the object being edited.
     */
    languages: string[];
}

export interface LinkInputBaseProps {
    autoFocus?: boolean;
    disabled?: boolean;
    error?: boolean;
    fullWidth?: boolean;
    helperText?: string | boolean | null | undefined;
    key?: string;
    label?: string;
    limitTo?: FindObjectType[];
    name: string;
    onBlur?: (event: React.FocusEvent<HTMLElement>) => unknown;
    onChange: (newLink: string) => unknown;
    onObjectData?: ({ title, subtitle }: { title: string; subtitle: string }) => unknown;
    placeholder?: string;
    size?: InputSize;
    sxs?: { root?: SxType };
    tabIndex?: number;
    value: string;
    variant?: InputVariant;
}

export type LinkInputProps = Omit<LinkInputBaseProps, "onChange" | "value">;


/** Base props for PasswordTextInputBase component - no Formik dependencies */
export interface PasswordTextInputBaseProps {
    /** Auto-complete attribute for the input */
    autoComplete?: string;
    /** Whether to focus on mount */
    autoFocus?: boolean;
    /** Whether the input takes full width */
    fullWidth?: boolean;
    /** Label for the password input */
    label?: string;
    /** Name attribute for the input */
    name: string;
    /** Current value */
    value: string;
    /** Change handler */
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
    /** Blur handler */
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Whether the input has an error */
    error?: boolean;
    /** Helper text to display below input */
    helperText?: string | boolean | null | undefined;
    /** ID for the input element */
    id?: string;
    /** Additional props to pass to TextInput */
    [key: string]: unknown;
}

/** Props for Formik-integrated PasswordTextInput component */
export interface PasswordTextInputFormikProps extends Omit<PasswordTextInputBaseProps, "value" | "onChange" | "error" | "helperText" | "onBlur"> {
    /** Optional validation function */
    validate?: (value: unknown) => string | void | Promise<string | void>;
}

/** Legacy PasswordTextInputProps for backward compatibility */
export type PasswordTextInputProps = TextInputProps & {
    autoComplete?: string;
    autoFocus?: boolean;
    fullWidth?: boolean;
    label?: string;
    name: string;
}

export type PhoneNumberInputProps = TextInputProps & {
    autoComplete?: string;
    autoFocus?: boolean;
    fullWidth?: boolean;
    label?: string;
    name: string;
    onChange?: (value: string) => unknown;
}

export type PhoneNumberInputBaseProps = Omit<PhoneNumberInputProps, "onChange"> & {
    error?: boolean;
    helperText?: string | boolean | null | undefined;
    onBlur?: (event: React.FocusEvent<HTMLElement>) => unknown;
    onChange: (value: string) => unknown;
    setError: (error: string | undefined) => unknown;
    size?: InputSize;
    value: string;
    variant?: InputVariant;
}

export type PreviewSwitchProps = Omit<BoxProps, "onChange"> & {
    disabled?: boolean;
    isPreviewOn: boolean;
    onChange: (isPreviewOn: boolean) => unknown;
}

export interface ProfilePictureInputProps {
    name?: string;
    onBannerImageChange: (file: File | null) => unknown;
    onProfileImageChange: (file: File | null) => unknown;
    profile?: {
        __typename: "Team" | "User";
        isBot?: boolean;
        bannerImage?: string | File | null;
        profileImage?: string | File | null;
        /** Used for cache busting */
        updatedAt?: string;
    } | null | undefined;
}

export type ResourceListInputProps = Pick<ResourceListProps, "sxs"> & {
    disabled?: boolean;
    horizontal: boolean;
    isCreate: boolean;
    isLoading?: boolean;
    parent: { __typename: ResourceListFor | `${ResourceListFor}`, id: string };
}

export type SelectorVariant = "outline" | "filled" | "underline";
export type SelectorSize = "sm" | "md" | "lg";

export interface SelectorProps<T extends string | number | Record<string, unknown>> {
    addOption?: {
        label: string;
        onSelect: () => unknown;
    };
    autoFocus?: boolean;
    disabled?: boolean;
    fullWidth?: boolean;
    getDisplayIcon?: (option: T) => JSX.Element | undefined;
    getOptionDescription?: (option: T, t: TranslationFunc) => string | null | undefined;
    getOptionIcon?: (option: T) => JSX.Element | undefined;
    getOptionLabel: (option: T, t: TranslationFunc) => string | null | undefined;
    inputAriaLabel?: string;
    isRequired?: boolean,
    label?: string;
    multiple?: false;
    name: string;
    noneOption?: boolean;
    noneText?: string;
    onChange?: (value: T | null) => unknown;
    options: readonly T[];
    size?: SelectorSize;
    sx?: SxType;
    tabIndex?: number;
    variant?: SelectorVariant;
}

export interface SelectorBaseProps<T extends string | number | Record<string, unknown>> extends Omit<SelectorProps<T>, "onChange" | "sx"> {
    error?: boolean;
    helperText?: string | boolean | null | undefined;
    onBlur?: (event: React.FocusEvent<HTMLElement>) => unknown;
    onChange: (value: T) => unknown;
    value: T | null;
    sxs?: {
        fieldset?: SxType;
        root?: SxType;
    };
}

export type ObjectVersionSelectPayloads = {
    ResourceVersion: ResourceVersion,
}
export type ObjectVersionSelectSwitchProps<T extends keyof ObjectVersionSelectPayloads> = {
    canUpdate: boolean;
    disabled?: boolean;
    label: string;
    selected: {
        translations: ObjectVersionSelectPayloads[T]["translations"];
    } | null;
    objectType: T;
    onChange: (value: ObjectVersionSelectPayloads[T] | null) => unknown;
    tooltip: string;
}

export type TagSelectorBaseProps = {
    disabled?: boolean;
    handleTagsUpdate: (tags: (TagShape | Tag)[]) => unknown;
    isRequired?: boolean;
    placeholder?: string;
    tags: (TagShape | Tag)[];
    sx?: SxType;
}

export type TagSelectorProps = Pick<TagSelectorBaseProps, "disabled" | "placeholder" | "sx"> & {
    name: string;
}

export type TextInputProps = Omit<TextFieldProps, "ref"> & {
    enterWillSubmit?: boolean;
    onSubmit?: () => unknown;
    isRequired?: boolean;
    ref?: RefObject<HTMLElement>;
}

/** Base props for TimezoneSelectorBase component - no Formik dependencies */
export interface TimezoneSelectorBaseProps {
    /** Current selected timezone value */
    value: string;
    /** Change handler */
    onChange: (value: string) => void;
    /** Blur handler */
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Whether the input has an error */
    error?: boolean;
    /** Helper text to display below input */
    helperText?: string | boolean | null | undefined;
    /** Label for the timezone selector */
    label?: string;
    /** Whether the input is required */
    isRequired?: boolean;
    /** Whether the input is disabled */
    disabled?: boolean;
    /** Name attribute for the input */
    name?: string;
    /** Additional props to pass to TextInput */
    [key: string]: unknown;
}

/** Props for Formik-integrated TimezoneSelector component */
export interface TimezoneSelectorFormikProps extends Omit<TimezoneSelectorBaseProps, "value" | "onChange" | "error" | "helperText" | "onBlur"> {
    /** Formik field name */
    name: string;
    /** Optional validation function */
    validate?: (value: unknown) => string | void | Promise<string | void>;
}

/** Legacy TimezoneSelectorProps for backward compatibility */
export type TimezoneSelectorProps = Omit<SelectorProps<string>, "getOptionLabel" | "options">


export interface TranslatedTextInputProps {
    autoFocus?: boolean;
    fullWidth?: boolean;
    isRequired?: boolean;
    label?: string;
    language: string;
    maxRows?: number;
    minRows?: number;
    multiline?: boolean;
    name: string;
    placeholder?: string;
    InputProps?: TextInputProps["InputProps"];
}

/** Base props for VersionInputBase component - no Formik dependencies */
export interface VersionInputBaseProps {
    /** Current version value */
    value: string;
    /** Change handler */
    onChange: (value: string) => void;
    /** Blur handler */
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Whether the input has an error */
    error?: boolean;
    /** Helper text to display below input */
    helperText?: string | boolean | null | undefined;
    /** Label for input component, NOT the version label. */
    label?: string;
    /** Name attribute for the input */
    name?: string;
    /** Whether the input is required */
    isRequired?: boolean;
    /** Whether the input is disabled */
    disabled?: boolean;
    /** Existing versions of the object. Used to determine minimum version number. */
    versions: string[];
    /** Additional props from TextInput */
    InputProps?: TextInputProps["InputProps"];
    /** Additional styles */
    sx?: SxType;
}

/** Props for Formik-integrated VersionInput component */
export interface VersionInputFormikProps extends Omit<VersionInputBaseProps, "value" | "onChange" | "error" | "helperText" | "onBlur"> {
    /** Formik field name */
    name?: string;
    /** Optional validation function */
    validate?: (value: unknown) => string | void | Promise<string | void>;
}

/** Legacy VersionInputProps for backward compatibility */
export type VersionInputProps = Omit<TextInputProps, "helperText" | "onBlur" | "onChange" | "value"> & {
    /** Label for input component, NOT the version label. */
    label?: string;
    name?: string;
    /** Existing versions of the object. Used to determine mimum version number. */
    versions: string[];
}

// Radio component types
export type RadioVariant = "primary" | "secondary" | "danger" | "success" | "warning" | "info" | "custom";
export type RadioSize = "sm" | "md" | "lg";

/** Base props for RadioBase component - no Formik dependencies */
export interface RadioBaseProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size" | "type" | "onChange"> {
    /** Visual style variant */
    color?: RadioVariant;
    /** Custom color for the custom variant (hex, rgb, hsl, etc.) */
    customColor?: string;
    /** Size of the radio button */
    size?: RadioSize;
    /** Whether the radio is checked */
    checked?: boolean;
    /** Default checked state for uncontrolled component */
    defaultChecked?: boolean;
    /** Value of the radio button */
    value?: string | number;
    /** Name attribute for grouping radios */
    name?: string;
    /** Whether the radio is disabled */
    disabled?: boolean;
    /** Whether the radio is required */
    required?: boolean;
    /** Change handler */
    onChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    /** Click handler */
    onClick?: (event: React.MouseEvent<HTMLInputElement>) => void;
    /** Focus handler */
    onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Blur handler */
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Additional CSS classes */
    className?: string;
    /** Inline styles */
    style?: React.CSSProperties;
    /** MUI sx prop for custom styling */
    sx?: SxType;
    /** Error state */
    error?: boolean;
    /** Helper text to display below radio */
    helperText?: React.ReactNode;
    /** Label to display next to radio */
    label?: React.ReactNode;
}

/** Props for Formik-integrated Radio component */
export interface RadioFormikProps extends Omit<RadioBaseProps, "checked" | "defaultChecked" | "onChange" | "error" | "helperText" | "name"> {
    /** Formik field name */
    name: string;
    /** Optional validation function */
    validate?: (value: unknown) => string | void | Promise<string | void>;
}

/** Legacy RadioProps for backward compatibility */
export interface RadioProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size" | "type" | "onChange"> {
    /** Visual style variant */
    color?: RadioVariant;
    /** Custom color for the custom variant (hex, rgb, hsl, etc.) */
    customColor?: string;
    /** Size of the radio button */
    size?: RadioSize;
    /** Whether the radio is checked */
    checked?: boolean;
    /** Default checked state for uncontrolled component */
    defaultChecked?: boolean;
    /** Value of the radio button */
    value?: string | number;
    /** Name attribute for grouping radios */
    name?: string;
    /** Whether the radio is disabled */
    disabled?: boolean;
    /** Whether the radio is required */
    required?: boolean;
    /** Change handler */
    onChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    /** Click handler */
    onClick?: (event: React.MouseEvent<HTMLInputElement>) => void;
    /** Focus handler */
    onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Blur handler */
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Additional CSS classes */
    className?: string;
    /** Inline styles */
    style?: React.CSSProperties;
    /** MUI sx prop for custom styling */
    sx?: SxType;
}

// Checkbox component types
export type CheckboxVariant = "primary" | "secondary" | "danger" | "success" | "warning" | "info" | "custom";
export type CheckboxSize = "sm" | "md" | "lg";

/** Base props for CheckboxBase component - no Formik dependencies */
export interface CheckboxBaseProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size" | "type" | "onChange"> {
    /** Visual style variant */
    color?: CheckboxVariant;
    /** Custom color for the custom variant (hex, rgb, hsl, etc.) */
    customColor?: string;
    /** Size of the checkbox */
    size?: CheckboxSize;
    /** Whether the checkbox is checked */
    checked?: boolean;
    /** Default checked state for uncontrolled component */
    defaultChecked?: boolean;
    /** Whether the checkbox is in indeterminate state */
    indeterminate?: boolean;
    /** Whether the checkbox is disabled */
    disabled?: boolean;
    /** Whether the checkbox is required */
    required?: boolean;
    /** Change handler */
    onChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    /** Click handler */
    onClick?: (event: React.MouseEvent<HTMLInputElement>) => void;
    /** Focus handler */
    onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Blur handler */
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Additional CSS classes */
    className?: string;
    /** Inline styles */
    style?: React.CSSProperties;
    /** MUI sx prop for custom styling */
    sx?: SxType;
    /** Error state */
    error?: boolean;
    /** Helper text to display below checkbox */
    helperText?: React.ReactNode;
    /** Label to display next to checkbox */
    label?: React.ReactNode;
}

/** Props for Formik-integrated Checkbox component */
export interface CheckboxFormikProps extends Omit<CheckboxBaseProps, "checked" | "defaultChecked" | "onChange" | "error" | "helperText" | "name"> {
    /** Formik field name */
    name: string;
    /** Optional validation function */
    validate?: (value: unknown) => string | void | Promise<string | void>;
}

/** Legacy CheckboxProps for backward compatibility */
export interface CheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size" | "type" | "onChange"> {
    /** Visual style variant */
    color?: CheckboxVariant;
    /** Custom color for the custom variant (hex, rgb, hsl, etc.) */
    customColor?: string;
    /** Size of the checkbox */
    size?: CheckboxSize;
    /** Whether the checkbox is checked */
    checked?: boolean;
    /** Default checked state for uncontrolled component */
    defaultChecked?: boolean;
    /** Whether the checkbox is in indeterminate state */
    indeterminate?: boolean;
    /** Whether the checkbox is disabled */
    disabled?: boolean;
    /** Whether the checkbox is required */
    required?: boolean;
    /** Change handler */
    onChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;
    /** Click handler */
    onClick?: (event: React.MouseEvent<HTMLInputElement>) => void;
    /** Focus handler */
    onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Blur handler */
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /** Additional CSS classes */
    className?: string;
    /** Inline styles */
    style?: React.CSSProperties;
    /** MUI sx prop for custom styling */
    sx?: SxType;
}

// FormControlLabel component types
export type FormControlLabelPlacement = "end" | "start" | "top" | "bottom";

export interface FormControlLabelProps {
    /** The control element (Radio, Checkbox, Switch) */
    control: React.ReactElement;
    /** The label content */
    label: React.ReactNode;
    /** The placement of the label relative to the control */
    labelPlacement?: FormControlLabelPlacement;
    /** Whether the control is disabled */
    disabled?: boolean;
    /** Whether the control is required */
    required?: boolean;
    /** Additional CSS classes */
    className?: string;
    /** Inline styles */
    style?: React.CSSProperties;
    /** MUI sx prop for custom styling */
    sx?: SxType;
    /** Value to be used in controlled forms */
    value?: unknown;
    /** Change handler */
    onChange?: (event: React.SyntheticEvent, checked: boolean) => void;
}

// FormGroup component types
export interface FormGroupProps {
    /** The content of the form group */
    children: React.ReactNode;
    /** Display group items in a row */
    row?: boolean;
    /** Additional CSS classes */
    className?: string;
    /** Inline styles */
    style?: React.CSSProperties;
    /** MUI sx prop for custom styling */
    sx?: SxType;
}

