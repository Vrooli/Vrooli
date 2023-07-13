import { Comment, CommentFor, StandardVersion, SvgComponent, Tag } from "@local/shared";
import { BoxProps, CheckboxProps, TextFieldProps } from "@mui/material";
import { FieldProps } from "formik";
import { JSONVariable } from "forms/types";
import { SxType } from "types";
import { ListObjectType } from "utils/display/listTools";
import { TagShape } from "utils/shape/models/tag";
import { StandardLanguage } from "./CodeInputBase/CodeInputBase";

export interface CharLimitIndicatorProps {
    chars: number;
    maxChars: number;
    size?: number;
}

export type CheckboxInputProps = Omit<(CheckboxProps & FieldProps), "form"> & {
    label: string;
};

export interface CommentUpsertInputProps {
    comment: Comment | undefined;
    language: string;
    objectId: string;
    objectType: CommentFor;
    onCancel: () => unknown;
    onCompleted: (comment: Comment) => unknown;
    parent: Comment | null;
    zIndex: number;
}

export interface CodeInputBaseProps {
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
    limitTo?: StandardLanguage[];
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
    zIndex: number;
}

export type CodeInputProps = Omit<CodeInputBaseProps, "defaultValue" | "format" | "variables">

export interface DateInputProps {
    fullWidth?: boolean;
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
    onUpload: (files: any[]) => unknown;
    showThumbs?: boolean;
    uploadText?: string;
}

export interface LanguageInputProps {
    disabled?: boolean;
    /**
     * Currently-selected language, if using this component to add/edit an object
     * with translations. Not needed if using this component to select languages 
     * for an advanced search, for example.
     */
    currentLanguage: string;
    handleAdd: (language: string) => unknown;
    handleDelete: (language: string) => unknown;
    handleCurrent: (language: string) => unknown;
    /**
     * All languages that currently have translations for the object being edited.
     */
    languages: string[];
    zIndex: number;
}

export interface LinkInputProps {
    label?: string;
    name?: string;
    onObjectData?: ({ title, subtitle }: { title: string; subtitle: string }) => unknown;
    zIndex: number;
}

export type MarkdownInputBaseProps = Omit<TextFieldProps, "onChange"> & {
    actionButtons?: Array<{
        disabled?: boolean;
        Icon: SvgComponent;
        onClick: () => unknown;
        tooltip?: string;
    }>;
    autoFocus?: boolean;
    disabled?: boolean;
    disableAssistant?: boolean;
    error?: boolean;
    /**
     * Callback to provide data for "@" tagging dropdown. 
     * If not provided, the dropdown will not appear.
     */
    getTaggableItems?: (query: string) => Promise<ListObjectType[]>;
    helperText?: string | boolean | null | undefined;
    maxChars?: number;
    maxRows?: number;
    minRows?: number;
    name: string;
    onBlur?: (event: React.FocusEvent<HTMLTextAreaElement>) => unknown;
    onChange: (newText: string) => unknown;
    placeholder?: string;
    tabIndex?: number;
    value: string;
    sxs?: {
        bar?: SxType;
        root?: SxType;
        textArea?: Record<string, unknown>;
    };
    zIndex: number;
}

export type MarkdownInputProps = Omit<MarkdownInputBaseProps, "onChange" | "value">

export type PasswordTextFieldProps = TextFieldProps & {
    autoComplete?: string;
    autoFocus?: boolean;
    fullWidth?: boolean;
    label?: string;
    name: string;
}

export type PreviewSwitchProps = Omit<BoxProps, "onChange"> & {
    disabled?: boolean;
    isPreviewOn: boolean;
    onChange: (isPreviewOn: boolean) => unknown;
}

export interface ProfilePictureInputProps {
    name?: string;
    onChange: (file: File | null) => unknown;
    profile?: {
        __typename: "Organization" | "User";
        isBot?: boolean;
    } | null | undefined;
    zIndex: number;
}

export interface IntegerInputProps extends BoxProps {
    autoFocus?: boolean;
    disabled?: boolean;
    fullWidth?: boolean;
    key?: string;
    initial?: number;
    label?: string;
    max?: number;
    min?: number;
    name: string;
    offset?: number;
    step?: number;
    tooltip?: string;
}

export interface ResourceListHorizontalInputProps {
    disabled?: boolean;
    isCreate: boolean;
    isLoading?: boolean;
    zIndex: number;
}

export interface SelectorProps<T extends string | number | { [x: string]: any }> {
    addOption?: {
        label: string;
        onSelect: () => unknown;
    };
    autoFocus?: boolean;
    disabled?: boolean;
    fullWidth?: boolean;
    getOptionDescription?: (option: T) => string | null | undefined;
    getOptionIcon?: (option: T) => SvgComponent;
    getOptionLabel: (option: T) => string;
    inputAriaLabel?: string;
    label?: string;
    multiple?: false;
    name: string;
    noneOption?: boolean;
    onChange?: (value: T | null) => unknown;
    options: T[];
    required?: boolean;
    sx?: SxType;
    tabIndex?: number;
}

export interface SelectorBaseProps<T extends string | number | { [x: string]: any }> extends Omit<SelectorProps<T>, "onChange"> {
    error?: boolean;
    helperText?: string | boolean | null | undefined;
    onBlur?: (event: React.FocusEvent<any>) => unknown;
    onChange: (value: T) => unknown;
    value: T | null;
}

export type StandardVersionSelectSwitchProps = {
    /** Indicates if the standard is allowed to be updated */
    canUpdateStandardVersion: boolean;
    selected: {
        translations: StandardVersion["translations"];
    } | null;
    onChange: (value: StandardVersion | null) => unknown;
    disabled?: boolean;
    zIndex: number;
}

export interface TagSelectorProps {
    disabled?: boolean;
    name: string;
    placeholder?: string;
    zIndex: number;
}

export interface TagSelectorBaseProps {
    disabled?: boolean;
    handleTagsUpdate: (tags: (TagShape | Tag)[]) => unknown;
    placeholder?: string;
    tags: (TagShape | Tag)[];
    zIndex: number
}

export interface TimezoneSelectorProps extends Omit<SelectorProps<string>, "getOptionLabel" | "options"> {
    zIndex: number;
}

export interface ToggleSwitchProps {
    checked: boolean;
    name?: string;
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => unknown;
    OffIcon?: SvgComponent;
    OnIcon?: SvgComponent;
    label?: string;
    tooltip?: string;
    disabled?: boolean;
    sx?: SxType;
}

export interface TranslatedMarkdownInputProps {
    actionButtons?: Array<{
        disabled?: boolean;
        Icon: SvgComponent;
        onClick: () => unknown;
        tooltip?: string;
    }>;
    disabled?: boolean;
    language: string;
    maxChars?: number;
    maxRows?: number;
    minRows?: number;
    name: string;
    placeholder?: string;
    sxs?: {
        bar?: SxType;
        root?: SxType;
        textArea?: Record<string, unknown>;
    };
    zIndex: number;
}

export interface TranslatedTextFieldProps {
    fullWidth?: boolean;
    label?: string;
    language: string;
    maxRows?: number;
    minRows?: number;
    multiline?: boolean;
    name: string;
    placeholder?: string;
}

export type VersionInputProps = Omit<TextFieldProps, "helperText" | "onBlur" | "onChange" | "value"> & {
    autoFocus?: boolean;
    fullWidth?: boolean;
    /**
     * Label for input component, NOT the version label.
     */
    label?: string;
    name?: string;
    /**
     * Existing versions of the object. Used to determine mimum version number.
     */
    versions: string[];
}
