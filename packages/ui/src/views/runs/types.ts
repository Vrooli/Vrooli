import { ProjectVersion, RoutineVersion, RunRoutine } from "@local/shared";
import { DecisionStep, EndStep, RoutineListStep, RoutineStep } from "types";
import { ViewProps } from "views/types";

export interface DecisionViewProps extends ViewProps {
    data: DecisionStep;
    handleDecisionSelect: (step: RoutineStep | EndStep) => void;
    routineList: RoutineListStep;
    zIndex: number;
}

export interface RunViewProps extends ViewProps {
    onClose?: () => void;
    runnableObject: ProjectVersion | RoutineVersion;
}

export interface SubroutineViewProps extends ViewProps {
    loading: boolean;
    handleUserInputsUpdate: (inputs: { [inputId: string]: string }) => void;
    handleSaveProgress: () => void;
    /**
     * Owner of overall routine, not subroutine
     */
    owner: RoutineVersion["root"]["owner"] | null | undefined;
    routineVersion: RoutineVersion | null | undefined;
    run: RunRoutine | null | undefined;
    zIndex: number;
}
