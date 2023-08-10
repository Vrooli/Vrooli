import { endpointGetRunRoutine, endpointPostRunRoutine, endpointPutRunRoutine, RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RunRoutineForm, runRoutineInitialValues, transformRunRoutineValues, validateRunRoutineValues } from "forms/RunRoutineForm/RunRoutineForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { RunRoutineShape } from "utils/shape/models/runRoutine";
import { RunRoutineUpsertProps } from "../types";

export const RunRoutineUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex,
}: RunRoutineUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<RunRoutine, RunRoutineShape>({
        ...endpointGetRunRoutine,
        objectType: "RunRoutine",
        upsertTransform: (existing) => runRoutineInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const { handleCancel, handleCompleted } = useUpsertActions<RunRoutine>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<RunRoutineCreateInput, RunRoutine>(endpointPostRunRoutine);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<RunRoutineUpdateInput, RunRoutine>(endpointPutRunRoutine);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<RunRoutineCreateInput | RunRoutineUpdateInput, RunRoutine>;

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
