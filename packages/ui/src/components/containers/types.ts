/* c8 ignore start */
import { CommentFor, TranslationKeyCommon } from "@local/shared";
import { TypographyProps } from "@mui/material";
import { ReactNode } from "react";
import { IconInfo } from "../../icons/Icons.js";
import { SxType } from "../../types.js";
import { RichInputProps, TextInputProps, TranslatedRichInputProps, TranslatedTextInputProps } from "../inputs/types.js";

export interface CommentContainerProps {
    forceAddCommentOpen?: boolean;
    isOpen?: boolean;
    language: string;
    objectId: string;
    objectType: CommentFor;
    onAddCommentClose?: () => unknown;
}

export interface TitleContainerProps {
    children: ReactNode;
    help?: string;
    /** Icon displayed to the left of the title */
    iconInfo?: IconInfo | null | undefined;
    title: string;
    id?: string;
    loading?: boolean;
    options?: {
        /** Adds icon for option to the right of the title */
        iconInfo?: IconInfo | null | undefined;
        label: string;
        onClick: (event?: any) => unknown;
    }[];
    sx?: SxType;
}

export interface ListContainerProps {
    borderRadius?: number;
    children: ReactNode;
    emptyText?: string;
    id?: string;
    isEmpty?: boolean;
    sx?: SxType;
}

/** Array of label, Icon, disabled, isSubmit, onClick */
export type DialogActionItem = [string, IconInfo | null | undefined, boolean, boolean, () => unknown]

export interface ContentCollapseProps {
    children?: React.ReactNode;
    /** True if the container is not collapsible. Defaults to false. */
    disableCollapse?: boolean;
    helpText?: string;
    id?: string;
    isOpen?: boolean;
    onOpenChange?: (isOpen: boolean) => unknown;
    sxs?: {
        titleContainer?: SxType;
        root?: SxType;
    }
    title?: string | null;
    titleComponent?: TypographyProps["component"];
    titleVariant?: TypographyProps["variant"];
    titleKey?: TranslationKeyCommon;
    titleVariables?: { [x: string]: string | number };
    toTheRight?: JSX.Element;
}

export interface TextCollapseProps {
    helpText?: string;
    isOpen?: boolean;
    loading?: boolean;
    loadingLines?: number;
    onOpenChange?: (isOpen: boolean) => unknown;
    title?: string | null;
    text?: string | null;
}

export type EditTextComponent = "Markdown" | "TranslatedMarkdown" | "TranslatedTextInput" | "TextInput";

interface BaseEditableTextProps<T extends EditTextComponent> {
    component: T;
    isEditing: boolean;
    name: string;
    showOnNoText?: boolean;
    variant?: "h1" | "h2" | "h3" | "h4" | "h5" | "h6" | "subtitle1" | "subtitle2" | "body1" | "body2";
}

interface BaseEditableTextCollapseProps<T extends EditTextComponent> extends BaseEditableTextProps<T> {
    helpText?: string;
    isOpen?: boolean;
    onOpenChange?: (isOpen: boolean) => unknown;
    title?: string | null;
}

export type PropsByComponentType = {
    Markdown: Omit<RichInputProps, "name" | "zIndex">;
    TranslatedMarkdown: Omit<TranslatedRichInputProps, "name" | "zIndex">;
    TranslatedTextInput: Omit<TranslatedTextInputProps, "name">;
    TextInput: Omit<TextInputProps, "error" | "helpText" | "name" | "onBlur" | "onChange" | "value">;
};

export type EditableTextProps<T extends EditTextComponent> = BaseEditableTextProps<T> & {
    props?: PropsByComponentType[T];
}
export type EditableTextCollapseProps<T extends EditTextComponent> = BaseEditableTextCollapseProps<T> & {
    props?: PropsByComponentType[T];
}
