import { TextFieldProps } from "@mui/material";
import { CommentFor, Session } from "@shared/consts";
import { MarkdownInputProps } from "components/inputs/types";

export interface CommentContainerProps {
    forceAddCommentOpen?: boolean;
    isOpen?: boolean;
    language: string;
    objectId: string;
    objectType: CommentFor;
    onAddCommentClose?: () => void;
    session: Session;
    zIndex: number;
}

export interface TitleContainerProps {
    children: JSX.Element | JSX.Element[];
    helpText?: string;
    id?: string;
    loading?: boolean;
    onClick?: (event: React.MouseEvent) => void;
    options?: [string, (e?: any) => void][];
    sx?: object;
    title?: string;
    tooltip?: string;
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

export interface EditableTextCollapseProps {
    helpText?: string;
    isEditing: boolean;
    isOpen?: boolean;
    onOpenChange?: (isOpen: boolean) => void;
    /**
     * Props for TextField
     */
    propsTextField?: TextFieldProps;
    /**
     * Props for MarkdownInput. If not set, assumes TextField is used.
     */
    propsMarkdownInput?: MarkdownInputProps;
    showOnNoText?: boolean;
    title?: string | null;
    text?: string | null;
}

export interface PageContainerProps {
    children: boolean | null | undefined | JSX.Element | (boolean | null | undefined | JSX.Element)[];
    sx?: { [x: string]: any };
}