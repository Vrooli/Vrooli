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

export interface DialogActionsContainerProps {
    actions: DialogActionItem[];
    /**
     * If true, the actions will be fixed to the bottom of the window
     */
    fixed?: boolean;
    onResize?: ({ height: number, width: number }) => any;
}

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