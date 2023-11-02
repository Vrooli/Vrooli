import { endpointGetRunProject, endpointPostRunProject, endpointPutRunProject, RunProject, RunProjectCreateInput, RunProjectUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { RunProjectForm, runProjectInitialValues, transformRunProjectValues, validateRunProjectValues } from "forms/RunProjectForm/RunProjectForm";
import { useFormDialog } from "hooks/useConfirmBeforeLeave";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { RunProjectShape } from "utils/shape/models/runProject";
import { RunProjectUpsertProps } from "../types";

export const RunProjectUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: RunProjectUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<RunProject, RunProjectShape>({
        ...endpointGetRunProject,
        objectType: "RunProject",
        overrideObject,
        transform: (existing) => runProjectInitialValues(session, existing),
    });

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
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="run-project-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateRun" : "UpdateRun")}
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
                        inputs: transformRunProjectValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateRunProjectValues(values, existing, isCreate)}
            >
                {(formik) => <RunProjectForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
