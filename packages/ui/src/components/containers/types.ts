import { TextFieldProps } from "@mui/material";
import { CommentFor } from "@shared/consts";
import { CommonKey } from "@shared/translations";
import { MarkdownInputProps, TranslatedMarkdownInputProps, TranslatedTextFieldProps } from "components/inputs/types";

export interface CommentContainerProps {
    forceAddCommentOpen?: boolean;
    isOpen?: boolean;
    language: string;
    objectId: string;
    objectType: CommentFor;
    onAddCommentClose?: () => void;
    zIndex: number;
}

export interface TitleContainerProps {
    children: JSX.Element | JSX.Element[];
    helpKey?: CommonKey;
    helpVariables?: { [x: string]: string | number };
    titleKey: CommonKey;
    titleVariables?: { [x: string]: string | number };
    id?: string;
    loading?: boolean;
    onClick?: (event: React.MouseEvent) => void;
    options?: [(CommonKey | { key: CommonKey, variables: Record<string, any> }), (e?: any) => void][];
    sx?: object;
    tooltip?: string;
}

export interface ListContainerProps {
    children: JSX.Element | JSX.Element[];
    emptyText?: string;
    isEmpty?: boolean;
    sx?: { [x: string]: any };
}

export interface ListTitleContainerProps extends TitleContainerProps {
    emptyText?: string;
    isEmpty: boolean;
}

// label, Icon, disabled, isSubmit, onClick
export type DialogActionItem = [string, any, boolean, boolean, () => void,]

export interface ContentCollapseProps {
    helpText?: string;
    id?: string;
    isOpen?: boolean;
    onOpenChange?: (isOpen: boolean) => void;
    sxs?: {
        titleContainer?: { [x: string]: any };
        root?: { [x: string]: any };
    }
    title?: string | null;
    titleComponent?: "h1" | "h2" | "h3" | "h4" | "h5" | "h6" | "p" | "span" | "legend";
    titleKey?: CommonKey;
    titleVariables?: { [x: string]: string | number };
    children?: React.ReactNode;
}

export interface TextCollapseProps {
    helpText?: string;
    isOpen?: boolean;
    loading?: boolean;
    loadingLines?: number;
    onOpenChange?: (isOpen: boolean) => void;
    title?: string | null;
    text?: string | null;
}

export type EditTextComponent = 'Markdown' | 'TranslatedMarkdown' | 'TranslatedTextField' | 'TextField';

interface BaseEditableTextProps<T extends EditTextComponent> {
    component: T;
    isEditing: boolean;
    name: string;
    showOnNoText?: boolean;
    variant?: 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6' | 'subtitle1' | 'subtitle2' | 'body1' | 'body2';
}

interface BaseEditableTextCollapseProps<T extends EditTextComponent> extends BaseEditableTextProps<T> {
    helpText?: string;
    isOpen?: boolean;
    onOpenChange?: (isOpen: boolean) => void;
    title?: string | null;
}

export type PropsByComponentType = {
    Markdown: Omit<MarkdownInputProps, 'name'>;
    TranslatedMarkdown: Omit<TranslatedMarkdownInputProps, 'name'>;
    TranslatedTextField: Omit<TranslatedTextFieldProps, 'name'>;
    TextField: Omit<TextFieldProps, 'error' | 'helpText' | 'name' | 'onBlur' | 'onChange' | 'value'>;
};

export type EditableTextProps<T extends EditTextComponent> = BaseEditableTextProps<T> & {
    props?: PropsByComponentType[T];
}
export type EditableTextCollapseProps<T extends EditTextComponent> = BaseEditableTextCollapseProps<T> & {
    props?: PropsByComponentType[T];
}

export interface PageContainerProps {
    children: boolean | null | undefined | JSX.Element | (boolean | null | undefined | JSX.Element)[];
    sx?: { [x: string]: any };
}