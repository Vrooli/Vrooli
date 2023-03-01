import { Node, ProjectVersion, RoutineVersion, RunRoutine, Session } from "@shared/consts";
import { DecisionStep } from "types";
import { ViewProps } from "views/objects/types";

export interface DecisionViewProps {
    data: DecisionStep;
    handleDecisionSelect: (node: Node) => void;
    nodes: Node[];
    session: Session | undefined;
    zIndex: number;
}

export interface RunViewProps extends ViewProps<RoutineVersion> {
    handleClose: () => void;
    runnableObject: ProjectVersion | RoutineVersion;
}

export interface SubroutineViewProps {
    loading: boolean;
    handleUserInputsUpdate: (inputs: { [inputId: string]: string }) => void;
    handleSaveProgress: () => void;
    /**
     * Owner of overall routine, not subroutine
     */
    owner: RoutineVersion['root']['owner'] | null | undefined;
    routineVersion: RoutineVersion | null | undefined;
    run: RunRoutine | null | undefined;
    session: Session | undefined;
    zIndex: number;
}
