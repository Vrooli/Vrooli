import { BuildRunState } from "utils";
import { Routine, Run, Session } from "types";
import { CommentFor } from "graphql/generated/globalTypes";

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
    handleCancelAdd: () => void;
    handleCancelUpdate: () => void;
    handleAdd: () => void;
    handleUpdate: () => void;
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