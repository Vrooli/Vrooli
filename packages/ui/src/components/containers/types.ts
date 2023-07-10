import { CommentFor, CommonKey, SvgComponent, SvgProps } from "@local/shared";
import { TextFieldProps } from "@mui/material";
import { MarkdownInputProps, TranslatedMarkdownInputProps, TranslatedTextFieldProps } from "components/inputs/types";
import { ReactNode } from "react";
import { SxType } from "types";

export interface CommentContainerProps {
    forceAddCommentOpen?: boolean;
    isOpen?: boolean;
    language: string;
    objectId: string;
    objectType: CommentFor;
    onAddCommentClose?: () => unknown;
    zIndex: number;
}

export interface TitleContainerProps {
    children: ReactNode;
    help?: string;
    /** Icon displayed to the left of the title */
    Icon?: SvgComponent;
    title: string;
    id?: string;
    loading?: boolean;
    onClick?: (event: React.MouseEvent) => unknown;
    options?: {
        /** Adds icon for option to the right of the title */
        Icon?: SvgComponent;
        label: string;
        onClick: (e?: any) => unknown;
    }[];
    sx?: SxType;
    zIndex: number;
}

export interface ListContainerProps {
    children: ReactNode;
    emptyText?: string;
    isEmpty?: boolean;
    sx?: SxType;
}

export interface ListTitleContainerProps extends TitleContainerProps {
    emptyText?: string;
    isEmpty: boolean;
}

/** Array of label, Icon, disabled, isSubmit, onClick */
export type DialogActionItem = [string, SvgComponent, boolean, boolean, () => unknown]

export interface ContentCollapseProps {
    children?: React.ReactNode;
    helpText?: string;
    id?: string;
    isOpen?: boolean;
    onOpenChange?: (isOpen: boolean) => unknown;
    sxs?: {
        titleContainer?: SxType;
        root?: SxType;
        helpButton?: SvgProps;
    }
    title?: string | null;
    titleComponent?: "h1" | "h2" | "h3" | "h4" | "h5" | "h6" | "p" | "span" | "legend";
    titleKey?: CommonKey;
    titleVariables?: { [x: string]: string | number };
    zIndex: number;
}

export interface TextCollapseProps {
    helpText?: string;
    isOpen?: boolean;
    loading?: boolean;
    loadingLines?: number;
    onOpenChange?: (isOpen: boolean) => unknown;
    title?: string | null;
    text?: string | null;
    zIndex: number;
}

export type EditTextComponent = "Markdown" | "TranslatedMarkdown" | "TranslatedTextField" | "TextField";

interface BaseEditableTextProps<T extends EditTextComponent> {
    component: T;
    isEditing: boolean;
    name: string;
    showOnNoText?: boolean;
    variant?: "h1" | "h2" | "h3" | "h4" | "h5" | "h6" | "subtitle1" | "subtitle2" | "body1" | "body2";
    zIndex: number;
}

interface BaseEditableTextCollapseProps<T extends EditTextComponent> extends BaseEditableTextProps<T> {
    helpText?: string;
    isOpen?: boolean;
    onOpenChange?: (isOpen: boolean) => unknown;
    title?: string | null;
}

export type PropsByComponentType = {
    Markdown: Omit<MarkdownInputProps, "name" | "zIndex">;
    TranslatedMarkdown: Omit<TranslatedMarkdownInputProps, "name" | "zIndex">;
    TranslatedTextField: Omit<TranslatedTextFieldProps, "name">;
    TextField: Omit<TextFieldProps, "error" | "helpText" | "name" | "onBlur" | "onChange" | "value">;
};

export type EditableTextProps<T extends EditTextComponent> = BaseEditableTextProps<T> & {
    props?: PropsByComponentType[T];
}
export type EditableTextCollapseProps<T extends EditTextComponent> = BaseEditableTextCollapseProps<T> & {
    props?: PropsByComponentType[T];
}

export interface PageContainerProps {
    children: ReactNode;
    sx?: SxType;
}
