import { type DeferredDecisionData, type ResolvedDecisionDataChooseMultiple, type ResolvedDecisionDataChooseOne, type RoutineVersion } from "@vrooli/shared";
import { type FormikProps } from "formik";
import { type RefObject } from "react";
import { type ViewProps } from "../../types.js";

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
