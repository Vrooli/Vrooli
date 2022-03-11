import { BuildRunState, BuildStatus } from "utils";
import { Routine, Session } from "types";
import { BuildStatusObject } from "components/graphs/NodeGraph/types";

export interface TitleContainerProps {
    id?: string;
    title?: string;
    onClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
    loading?: boolean;
    tooltip?: string;
    helpText?: string;
    options?: [string, (e?: any) => void][];
    sx?: object;
    children: JSX.Element | JSX.Element[];
}

// label, Icon, disabled, isSubmit, onClick
export type DialogActionItem = [string, any, boolean, boolean, () => void,]

export interface DialogActionsContainerProps {
    fixed?: boolean; // if true, the actions will be fixed to the bottom of the window
    actions: DialogActionItem[];
    onResize?: ({ height: number, width: number }) => any;
}

export interface BuildBottomContainerProps {
    canUpdate: boolean;
    canCancelUpdate: boolean;
    handleCancelRoutineUpdate: () => void;
    handleRoutineUpdate: () => void;
    handleScaleChange: (scale: number) => void;
    hasPrevious: boolean;
    hasNext: boolean;
    isEditing: boolean;
    loading: boolean;
    scale: number;
    sliderColor: string;
    runState: BuildRunState
}

export interface BuildInfoContainerProps {
    canEdit: boolean;
    handleRoutineUpdate: (changedRoutine: Routine) => void;
    handleStartEdit: () => void;
    handleTitleUpdate: (newTitle: string) => void;
    isEditing: boolean;
    language: string; // Language to display/edit
    routine: Routine | null;
    session: Session;
    status: BuildStatusObject;
}