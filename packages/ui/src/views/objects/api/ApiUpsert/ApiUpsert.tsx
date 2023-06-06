import { ApiVersion, ApiVersionCreateInput, ApiVersionUpdateInput, endpointGetApiVersion, endpointPostApiVersion, endpointPutApiVersion, FindVersionInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { ApiForm, apiInitialValues, transformApiValues, validateApiValues } from "forms/ApiForm/ApiForm";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ApiUpsertProps } from "../types";

export const ApiUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: ApiUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindVersionInput, ApiVersion>(endpointGetApiVersion);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => apiInitialValues(session), [session]);
    const { handleCancel, handleCompleted } = useUpsertActions<ApiVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<ApiVersionCreateInput, ApiVersion>(endpointPostApiVersion);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<ApiVersionUpdateInput, ApiVersion>(endpointPutApiVersion);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<ApiVersionCreateInput | ApiVersionUpdateInput, ApiVersion>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateApi" : "UpdateApi",
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
