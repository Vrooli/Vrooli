/* eslint-disable no-magic-numbers */
import { IconInfo } from "../../../icons/Icons.js";

export const advancedInputTextareaClassName = "advanced-input-field";

/**
 * All available actions to entering, editing, and styling text in the advanced input.
 */
export enum AdvancedInputAction {
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

/**
 * The AdvancedInput actions related to styling.
 */
export type AdvancedInputStylingAction = Extract<AdvancedInputAction, "Bold" | "Code" | "Header1" | "Header2" | "Header3" | "Header4" | "Header5" | "Header6" | "Italic" | "Link" | "ListBullet" | "ListCheckbox" | "ListNumber" | "Quote" | "Spoiler" | "Strikethrough" | "Underline">;

/**
 * The active states of the advanced input, for styling purposes.
 */
export type AdvancedInputActiveStates = { [x in Exclude<AdvancedInputAction, "Mode" | "Redo" | "Undo" | "SetValue">]: boolean };

export type ExternalApp = {
    id: string;
    name: string;
    iconInfo: IconInfo;
    connected: boolean;
}

export enum ToolState {
    /** Tool not provided to LLM */
    Disabled = "disabled",
    /** Tool provided to LLM with other enabled tools */
    Enabled = "enabled",
    /** LLM instructed to use this tool only */
    Exclusive = "exclusive",
}
export type Tool = {
    displayName: string;
    iconInfo: IconInfo;
    type: string;
    name: string;
    state: ToolState;
    arguments: Record<string, any>;
}

//TODO should migrate to TaskContextInfo, and update TaskContextInfo to include things like type
export type ContextItem = {
    id: string;
    type: "file" | "image" | "text";
    label: string;
    src?: string;
    file?: File;
}

export type AdvancedInputBaseProps = {
    contextData: ContextItem[];
    disabled?: boolean;
    error?: boolean;
    helperText?: string;
    maxChars?: number;
    name: string;
    placeholder?: string;
    tools: Tool[];
    value: string;
    onBlur?: (event: any) => unknown;
    onChange: (value: string) => unknown;
    onFocus?: (event: any) => unknown;
    onToolsChange?: (updatedTools: Tool[]) => unknown;
    onContextDataChange?: (updatedContext: ContextItem[]) => unknown;
    onSubmit?: (value: string) => unknown;
    tabIndex?: number;
}

export interface AdvancedInputProps extends Omit<AdvancedInputBaseProps, "value" | "onChange" | "onBlur" | "error" | "helperText"> {
    name: string;
}

export interface TranslatedAdvancedInputProps extends Omit<AdvancedInputBaseProps, "value" | "onChange" | "onBlur" | "error" | "helperText"> {
    language: string;
    name: string;
}

interface AdvancedInputChildProps extends Pick<AdvancedInputBaseProps, "disabled" | "name" | "onBlur" | "onChange" | "onFocus" | "onSubmit" | "placeholder" | "tabIndex" | "value"> {
    enterWillSubmit: boolean;
    id: string;
    maxRows: number;
    minRows: number;
    onActiveStatesChange: (activeStates: AdvancedInputActiveStates) => unknown;
    redo: () => unknown;
    setHandleAction: (handleAction: (action: AdvancedInputAction, data?: unknown) => unknown) => unknown;
    toggleMarkdown: () => unknown;
    undo: () => unknown;
}
export type AdvancedInputMarkdownProps = AdvancedInputChildProps;
export type AdvancedInputLexicalProps = AdvancedInputChildProps;
