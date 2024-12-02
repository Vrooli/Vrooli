import { DecisionStep, EndStep, RoutineListStep, RunnableRoutineVersion } from "@local/shared";
import { FormikProps } from "formik";
import { RefObject } from "react";
import { ViewProps } from "types";

export type DecisionViewProps = Omit<ViewProps, "display" | "isOpen"> & {
    /** The decision step data */
    data: DecisionStep;
    /** Callback when a decision is selected */
    handleDecisionSelect: (step: DecisionStep | EndStep | RoutineListStep) => unknown;
}

export type RunViewProps = ViewProps & {
    onClose?: () => unknown;
}

export type SubroutineViewProps = {
    formikRef: RefObject<FormikProps<object>>;
    handleGenerateOutputs: () => unknown;
    isGeneratingOutputs: boolean;
    isLoading: boolean;
    routineVersion: RunnableRoutineVersion;
}
