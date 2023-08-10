import { ApiVersion, ApiVersionCreateInput, ApiVersionUpdateInput, endpointGetApiVersion, endpointPostApiVersion, endpointPutApiVersion } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { ApiForm, apiInitialValues, transformApiValues, validateApiValues } from "forms/ApiForm/ApiForm";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ApiVersionShape } from "utils/shape/models/apiVersion";
import { ApiUpsertProps } from "../types";

export const ApiUpsert = ({
    isCreate,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: ApiUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { display, isLoading: isReadLoading, object: existing } = useObjectFromUrl<ApiVersion, ApiVersionShape>({
        ...endpointGetApiVersion,
        objectType: "ApiVersion",
        overrideObject,
        transform: (data) => apiInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
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

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateApi" : "UpdateApi")}
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
                    fetchLazyWrapper<ApiVersionCreateInput | ApiVersionUpdateInput, ApiVersion>({
                        fetch,
                        inputs: transformApiValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateApiValues(values, existing)}
            >
                {(formik) => <ApiForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={[]}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    );
};
