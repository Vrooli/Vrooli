import { ProjectVersion, RoutineVersion, RunRoutine } from "@local/shared";
import { DecisionStep, EndStep, PartialWithType, RoutineListStep, RoutineStep } from "types";
import { ViewProps } from "views/types";

export interface DecisionViewProps extends ViewProps {
    data: DecisionStep;
    handleDecisionSelect: (step: RoutineStep | EndStep) => unknown;
    routineList: RoutineListStep;
}

export interface RunViewProps extends ViewProps {
    onClose?: () => unknown;
    runnableObject: PartialWithType<ProjectVersion | RoutineVersion>;
}

export interface SubroutineViewProps extends ViewProps {
    loading: boolean;
    handleUserInputsUpdate: (inputs: { [inputId: string]: string }) => unknown;
    handleSaveProgress: () => unknown;
    /**
     * Owner of overall routine, not subroutine
     */
    owner: RoutineVersion["root"]["owner"] | null | undefined;
    routineVersion: RoutineVersion | null | undefined;
    run: RunRoutine | null | undefined;
}
