import { Node, ProjectVersion, RoutineVersion, RunRoutine, Session } from "@shared/consts";
import { DecisionStep } from "types";
import { ViewProps } from "views/objects/types";
import { BaseViewProps } from "views/types";

export interface DecisionViewProps extends BaseViewProps {
    data: DecisionStep;
    handleDecisionSelect: (node: Node) => void;
    nodes: Node[];
    zIndex: number;
}

export interface RunViewProps extends ViewProps<RoutineVersion> {
    handleClose: () => void;
    runnableObject: ProjectVersion | RoutineVersion;
}

export interface SubroutineViewProps extends BaseViewProps {
    loading: boolean;
    handleUserInputsUpdate: (inputs: { [inputId: string]: string }) => void;
    handleSaveProgress: () => void;
    /**
     * Owner of overall routine, not subroutine
     */
    owner: RoutineVersion['root']['owner'] | null | undefined;
    routineVersion: RoutineVersion | null | undefined;
    run: RunRoutine | null | undefined;
    zIndex: number;
}
