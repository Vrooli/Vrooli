import { endpointGetRunRoutine, endpointPostRunRoutine, endpointPutRunRoutine, RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RunRoutineForm, runRoutineInitialValues, transformRunRoutineValues, validateRunRoutineValues } from "forms/RunRoutineForm/RunRoutineForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { RunRoutineShape } from "utils/shape/models/runRoutine";
import { RunRoutineUpsertProps } from "../types";

export const RunRoutineUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: RunRoutineUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<RunRoutine, RunRoutineShape>({
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
        <MaybeLargeDialog
            display={display}
            id="run-routine-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
            zIndex={zIndex}
        >
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
                        inputs: transformRunRoutineValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateRunRoutineValues(values, existing, isCreate)}
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
        </MaybeLargeDialog>
    );
};
