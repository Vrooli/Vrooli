import { BuildRunState } from "utils";
import { Routine, Run, Session } from "types";
import { CommentFor } from "graphql/generated/globalTypes";
import { TextFieldProps } from "@mui/material";
import { MarkdownInputProps } from "components/inputs/types";
import { GridSubmitButtonsProps } from "components/buttons/types";

export interface CommentContainerProps {
    language: string;
    objectId: string;
    objectType: CommentFor;
    session: Session;
    sxs?: {
        root: any;
    }
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

export interface BuildBottomContainerProps {
    canSubmitMutate: boolean;
    canCancelMutate: boolean;
    errors: GridSubmitButtonsProps['errors'];
    handleCancel: () => void;
    handleSubmit: () => void;
    handleScaleChange: (scale: number) => void;
    handleRunDelete: (run: Run) => void;
    handleRunAdd: (run: Run) => void;
    hasNext: boolean;
    hasPrevious: boolean;
    isAdding: boolean;
    isEditing: boolean;
    loading: boolean;
    scale: number;
    session: Session;
    sliderColor: string;
    routine: Routine | null;
    runState: BuildRunState
    zIndex: number;
}

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
    onOpenChange?: (isOpen: boolean) => void;
    showOnNoText?: boolean;
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