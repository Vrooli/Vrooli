import { endpointGetRunRoutine, endpointPostRunRoutine, endpointPutRunRoutine, RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RunRoutineForm, runRoutineInitialValues, transformRunRoutineValues, validateRunRoutineValues } from "forms/RunRoutineForm/RunRoutineForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { RunRoutineShape } from "utils/shape/models/runRoutine";
import { RunRoutineUpsertProps } from "../types";

export const RunRoutineUpsert = ({
    isCreate,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: RunRoutineUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { display, isLoading: isReadLoading, object: existing } = useObjectFromUrl<RunRoutine, RunRoutineShape>({
        ...endpointGetRunRoutine,
        objectType: "RunRoutine",
        overrideObject,
        transform: (existing) => runRoutineInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput>({
        display,
        endpointCreate: endpointPostRunRoutine,
        endpointUpdate: endpointPutRunRoutine,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateRun" : "UpdateRun")}
                zIndex={zIndex}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<RunRoutineCreateInput | RunRoutineUpdateInput, RunRoutine>({
                        fetch,
                        inputs: transformRunRoutineValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateRunRoutineValues(values, existing)}
            >
                {(formik) => <RunRoutineForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    );
};
