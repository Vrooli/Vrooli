import { DecisionStep, EndStep, RoutineListStep, RunnableProjectVersion, RunnableRoutineVersion } from "@local/shared";
import { FormikProps } from "formik";
import { RefObject } from "react";
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
    inputFormikRef: RefObject<FormikProps<object>>;
    routineVersion: RunnableRoutineVersion;
}
