import { RoutineVersion, RunRoutine } from "@local/shared";
import { DecisionStep, EndStep, RoutineListStep, RunnableProjectVersion, RunnableRoutineVersion } from "utils/runUtils";
import { ViewProps } from "views/types";

export type DecisionViewProps = Omit<ViewProps, "display" | "isOpen"> & {
    /** The decision step data */
    data: DecisionStep;
    /** Callback when a decision is selected */
    handleDecisionSelect: (step: DecisionStep | EndStep | RoutineListStep) => unknown;
}

export type RunViewProps = ViewProps & {
    onClose?: () => unknown;
    runnableObject: RunnableRoutineVersion | RunnableProjectVersion;
}

export type SubroutineViewProps = Omit<ViewProps, "display" | "isOpen"> & {
    loading: boolean;
    handleUserInputsUpdate: (inputs: { [inputId: string]: string }) => unknown;
    handleSaveProgress: () => unknown;
    /**
     * Owner of overall routine, not subroutine
     */
    owner: RoutineVersion["root"]["owner"] | null | undefined;
    routineVersion: RunnableRoutineVersion;
    run: RunRoutine | null | undefined;
}
