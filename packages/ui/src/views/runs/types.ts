import { ProjectVersion, RoutineVersion, RunRoutine } from "@local/shared";
import { DecisionStep, EndStep, RoutineListStep } from "types";
import { ViewProps } from "views/types";

export type RunnableRoutineVersion = Pick<RoutineVersion, "__typename" | "id" | "complexity" | "configCallData" | "configFormInput" | "configFormOutput" | "nodeLinks" | "nodes" | "root" | "routineType" | "translations" | "versionLabel" | "you">
export type RunnableProjectVersion = Pick<ProjectVersion, "__typename" | "id" | "directories" | "root" | "translations" | "versionLabel" | "you">

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
