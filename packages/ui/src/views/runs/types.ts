import { ProjectVersion, RoutineVersion, RunRoutine } from "@local/shared";
import { DecisionStep, EndStep, RoutineListStep } from "types";
import { ViewProps } from "views/types";

export type RunnableRoutineVersion = Pick<RoutineVersion, "__typename" | "id" | "nodeLinks" | "nodes" | "routineType" | "translations" | "you">
export type RunnableProjectVersion = Pick<ProjectVersion, "__typename" | "id" | "directories" | "translations" | "you">

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
    routineVersion: RoutineVersion | null | undefined;
    run: RunRoutine | null | undefined;
}
