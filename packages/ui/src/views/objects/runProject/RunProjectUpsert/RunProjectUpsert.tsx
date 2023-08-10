import { endpointGetRunProject, endpointPostRunProject, endpointPutRunProject, RunProject, RunProjectCreateInput, RunProjectUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { RunProjectForm, runProjectInitialValues, transformRunProjectValues, validateRunProjectValues } from "forms/RunProjectForm/RunProjectForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { RunProjectShape } from "utils/shape/models/runProject";
import { RunProjectUpsertProps } from "../types";

export const RunProjectUpsert = ({
    isCreate,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: RunProjectUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { display, isLoading: isReadLoading, object: existing } = useObjectFromUrl<RunProject, RunProjectShape>({
        ...endpointGetRunProject,
        objectType: "RunProject",
        overrideObject,
        transform: (existing) => runProjectInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<RunProject, RunProjectCreateInput, RunProjectUpdateInput>({
        display,
        endpointCreate: endpointPostRunProject,
        endpointUpdate: endpointPutRunProject,
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
                    fetchLazyWrapper<RunProjectCreateInput | RunProjectUpdateInput, RunProject>({
                        fetch,
                        inputs: transformRunProjectValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateRunProjectValues(values, existing)}
            >
                {(formik) => <RunProjectForm
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
