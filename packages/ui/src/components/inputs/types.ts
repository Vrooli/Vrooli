import { ApiVersion, CodeLanguage, CodeVersion, JSONVariable, ListObject, NoteVersion, ProjectVersion, ResourceListFor, RoutineVersion, StandardVersion, Tag, TagShape } from "@local/shared";
import { BoxProps, CheckboxProps, TextFieldProps } from "@mui/material";
import { FindObjectTabOption } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { ResourceListProps } from "components/lists/resource/types";
import { FieldProps } from "formik";
import { CSSProperties, RefObject } from "react";
import { SvgComponent, SxType } from "types";

export interface CharLimitIndicatorProps {
    chars: number;
    /** Hides indicator until this number of characters is reached */
    minCharsToShow?: number;
    maxChars: number;
    size?: number;
}

export type CheckboxInputProps = Omit<(CheckboxProps & FieldProps), "form"> & {
    label: string;
};

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

export type CodeInputProps = Omit<CodeInputBaseProps, "codeLanguage" | "content" | "defaultValue" | "format" | "handleCodeLanguageChange" | "handleContentChange" | "variables"> & {
    codeLanguageField?: string;
}

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
    onUpload: (files: any[]) => unknown;
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
    step?: number;
    sx?: SxType;
    tooltip?: string;
    value: number;
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
    limitTo?: FindObjectTabOption[];
    name: string;
    onBlur?: (event: React.FocusEvent<HTMLElement>) => unknown;
    onChange: (newLink: string) => unknown;
    onObjectData?: ({ title, subtitle }: { title: string; subtitle: string }) => unknown;
    placeholder?: string;
    sxs?: { root?: SxType };
    tabIndex?: number;
    value: string;
}

export type LinkInputProps = Omit<LinkInputBaseProps, "onChange" | "value">;

export type RichInputBaseProps = Omit<TextInputProps, "onChange" | "onSubmit"> & {
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
    getTaggableItems?: (query: string) => Promise<ListObject[]>;
    helperText?: string | boolean | null | undefined;
    maxChars?: number;
    maxRows?: number;
    minRows?: number;
    name: string;
    onBlur?: (event: React.FocusEvent<HTMLElement>) => unknown;
    onFocus?: (event: React.FocusEvent<HTMLElement>) => unknown;
    onChange: (newText: string) => unknown;
    /** Allows "Enter" or "Shift+Enter" to submit */
    onSubmit?: (newText: string) => unknown;
    placeholder?: string;
    tabIndex?: number;
    value: string;
    sxs?: {
        topBar?: SxType;
        bottomBar?: SxType;
        root?: SxType;
        inputRoot?: SxType;
        textArea?: CSSProperties;
    };
}

export type RichInputProps = Omit<RichInputBaseProps, "onChange" | "value">

export interface RichInputChildProps extends Omit<RichInputBaseProps, "actionButtons" | "helperText" | "maxChars" | "sxs"> {
    enterWillSubmit?: boolean;
    id: string;
    openAssistantDialog: (selected: string, fullText: string) => unknown;
    onActiveStatesChange: (activeStates: RichInputActiveStates) => unknown;
    redo: () => unknown;
    setHandleAction: (handleAction: (action: RichInputAction, data?: unknown) => unknown) => unknown;
    toggleMarkdown: () => unknown;
    undo: () => unknown;
    sxs?: {
        inputRoot?: SxType;
        textArea?: CSSProperties;
    };
}

export type RichInputMarkdownProps = RichInputChildProps;
export type RichInputLexicalProps = RichInputChildProps;

export enum RichInputAction {
    Assistant = "Assistant",
    Bold = "Bold",
    Code = "Code",
    Header1 = "Header1",
    Header2 = "Header2",
    Header3 = "Header3",
    Header4 = "Header4",
    Header5 = "Header5",
    Header6 = "Header6",
    Italic = "Italic",
    Link = "Link",
    ListBullet = "ListBullet",
    ListCheckbox = "ListCheckbox",
    ListNumber = "ListNumber",
    Mode = "Mode",
    Quote = "Quote",
    Redo = "Redo",
    SetValue = "SetValue",
    Spoiler = "Spoiler",
    Strikethrough = "Strikethrough",
    Table = "Table",
    Underline = "Underline",
    Undo = "Undo",
}
export type RichInputActiveStates = { [x in Exclude<RichInputAction, "Assistant" | "Mode" | "Redo" | "Undo" | "SetValue">]: boolean };

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
    onBlur?: (event: React.FocusEvent<any>) => unknown;
    onChange: (value: string) => unknown;
    setError: (error: string | undefined) => unknown;
    value: string;
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
        updated_at?: string;
    } | null | undefined;
}

export type ResourceListInputProps = Pick<ResourceListProps, "sxs"> & {
    disabled?: boolean;
    horizontal: boolean;
    isCreate: boolean;
    isLoading?: boolean;
    parent: { __typename: ResourceListFor | `${ResourceListFor}`, id: string };
}

export interface SelectorProps<T extends string | number | { [x: string]: any }> {
    addOption?: {
        label: string;
        onSelect: () => unknown;
    };
    autoFocus?: boolean;
    disabled?: boolean;
    fullWidth?: boolean;
    getDisplayIcon?: (option: T) => SvgComponent | JSX.Element | undefined;
    getOptionDescription?: (option: T) => string | null | undefined;
    getOptionIcon?: (option: T) => SvgComponent | JSX.Element | undefined;
    getOptionLabel: (option: T) => string | null | undefined;
    inputAriaLabel?: string;
    isRequired?: boolean,
    label?: string;
    multiple?: false;
    name: string;
    noneOption?: boolean;
    noneText?: string;
    onChange?: (value: T | null) => unknown;
    options: readonly T[];
    sx?: SxType;
    tabIndex?: number;
}

export interface SelectorBaseProps<T extends string | number | { [x: string]: any }> extends Omit<SelectorProps<T>, "onChange" | "sx"> {
    color?: string;
    error?: boolean;
    helperText?: string | boolean | null | undefined;
    onBlur?: (event: React.FocusEvent<any>) => unknown;
    onChange: (value: T) => unknown;
    value: T | null;
    sxs?: {
        fieldset?: SxType;
        root?: SxType;
    };
}

export type ObjectVersionSelectPayloads = {
    ApiVersion: ApiVersion,
    CodeVersion: CodeVersion,
    NoteVersion: NoteVersion,
    ProjectVersion: ProjectVersion,
    RoutineVersion: RoutineVersion,
    StandardVersion: StandardVersion,
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

export type TimezoneSelectorProps = Omit<SelectorProps<string>, "getOptionLabel" | "options">

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

export interface TranslatedRichInputProps {
    actionButtons?: Array<{
        disabled?: boolean;
        Icon: SvgComponent;
        onClick: () => unknown;
        tooltip?: string;
    }>;
    autoFocus?: boolean;
    disabled?: boolean;
    isRequired?: boolean;
    language: string;
    maxChars?: number;
    maxRows?: number;
    minRows?: number;
    name: string;
    placeholder?: string;
    sxs?: {
        topBar?: SxType;
        bottomBar?: SxType;
        root?: SxType;
        inputRoot?: SxType;
        textArea?: CSSProperties;
    };
}

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

export type VersionInputProps = Omit<TextInputProps, "helperText" | "onBlur" | "onChange" | "value"> & {
    autoFocus?: boolean;
    fullWidth?: boolean;
    /** Label for input component, NOT the version label. */
    label?: string;
    name?: string;
    /** Existing versions of the object. Used to determine mimum version number. */
    versions: string[];
}
