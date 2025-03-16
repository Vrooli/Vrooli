import { DeferredDecisionData, ResolvedDecisionDataChooseMultiple, ResolvedDecisionDataChooseOne, RoutineVersion } from "@local/shared";
import { FormikProps } from "formik";
import { RefObject } from "react";
import { ViewProps } from "types";

export type DecisionViewProps = Omit<ViewProps, "display" | "isOpen"> & {
    /** The decision to make */
    data: DeferredDecisionData;
    /** Callback when a decision is selected */
    handleDecisionSelect: (data: ResolvedDecisionDataChooseOne | ResolvedDecisionDataChooseMultiple) => unknown;
}

export type RunViewProps = ViewProps & {
    onClose?: () => unknown;
}

export type SubroutineViewProps = {
    formikRef: RefObject<FormikProps<object>>;
    handleGenerateOutputs: () => unknown;
    isGeneratingOutputs: boolean;
    isLoading: boolean;
    routineVersion: RoutineVersion;
}
