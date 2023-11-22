import { ProjectVersion, RoutineVersion, RunRoutine } from "@local/shared";
import { DecisionStep, EndStep, PartialWithType, RoutineListStep, RoutineStep } from "types";
import { ViewProps } from "views/types";

export type DecisionViewProps = Omit<ViewProps, "display" | "isOpen"> & {
    data: DecisionStep;
    handleDecisionSelect: (step: RoutineStep | EndStep) => unknown;
    routineList: RoutineListStep;
}

export type RunViewProps = ViewProps & {
    onClose?: () => unknown;
    runnableObject: PartialWithType<ProjectVersion | RoutineVersion>;
}

export type SubroutineViewProps = Omit<ViewProps, "display" | "isOpen"> & {
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
