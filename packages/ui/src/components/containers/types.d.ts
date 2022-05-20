import { BuildRunState } from "utils";
import { Node, Routine, Session } from "types";
import { BuildStatusObject } from "components/graphs/NodeGraph/types";

export interface TitleContainerProps {
    children: JSX.Element | JSX.Element[];
    helpText?: string;
    id?: string;
    loading?: boolean;
    onClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
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
}

export interface BuildInfoContainerProps {
    canEdit: boolean;
    handleAddLink: () => void;
    handleLanguageUpdate: (language: string) => void;
    handleNodeDelete: (nodeId: string) => void;
    handleRoutineAction: (action: BaseObjectAction) => void;
    handleRoutineUpdate: (changedRoutine: Routine) => void;
    handleStartEdit: () => void;
    handleTitleUpdate: (newTitle: string) => void;
    isEditing: boolean;
    language: string;
    loading: boolean;
    nodesOffGraph: Node[];
    routine: Routine | null;
    session: Session;
    status: BuildStatusObject;
}