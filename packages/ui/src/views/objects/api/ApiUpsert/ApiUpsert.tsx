import { ApiVersion, ApiVersionCreateInput, ApiVersionUpdateInput, endpointGetApiVersion, endpointPostApiVersion, endpointPutApiVersion } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { ApiForm, apiInitialValues, transformApiValues, validateApiValues } from "forms/ApiForm/ApiForm";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { ApiVersionShape } from "utils/shape/models/apiVersion";
import { ApiUpsertProps } from "../types";

export const ApiUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: ApiUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<ApiVersion, ApiVersionShape>({
        ...endpointGetApiVersion,
        objectType: "ApiVersion",
        overrideObject,
        transform: (data) => apiInitialValues(session, data),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<ApiVersion, ApiVersionCreateInput, ApiVersionUpdateInput>({
        display,
        endpointCreate: endpointPostApiVersion,
        endpointUpdate: endpointPutApiVersion,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="api-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateApi" : "UpdateApi")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<ApiVersionCreateInput | ApiVersionUpdateInput, ApiVersion>({
                        fetch,
                        inputs: transformApiValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateApiValues(values, existing, isCreate)}
            >
                {(formik) => <ApiForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    versions={[]}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
