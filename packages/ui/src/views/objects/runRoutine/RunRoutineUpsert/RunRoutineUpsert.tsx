import { FindByIdInput, RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput } from "@local/shared";
import { runRoutineCreate } from "api/generated/endpoints/runRoutine_create";
import { runRoutineFindOne } from "api/generated/endpoints/runRoutine_findOne";
import { runRoutineUpdate } from "api/generated/endpoints/runRoutine_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RunRoutineForm, runRoutineInitialValues, transformRunRoutineValues, validateRunRoutineValues } from "forms/RunRoutineForm/RunRoutineForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { RunRoutineUpsertProps } from "../types";

export const RunRoutineUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: RunRoutineUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<RunRoutine, FindByIdInput>(runRoutineFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => runRoutineInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<RunRoutine>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<RunRoutine, RunRoutineCreateInput>(runRoutineCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<RunRoutine, RunRoutineUpdateInput>(runRoutineUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateRun" : "UpdateRun",
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper<RunRoutine, RunRoutineCreateInput | RunRoutineUpdateInput>({
                        mutation,
                        input: transformRunRoutineValues(values, existing),
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