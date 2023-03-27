import { ApiVersion, ApiVersionUpdateInput, FindVersionInput } from "@shared/consts";
import { apiVersionValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { apiVersionFindOne } from "api/generated/endpoints/apiVersion_findOne";
import { apiVersionUpdate } from "api/generated/endpoints/apiVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from 'formik';
import { ApiForm } from "forms/ApiForm/ApiForm";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeApiVersion } from "utils/shape/models/apiVersion";
import { apiInitialValues } from "..";
import { ApiUpdateProps } from "../types";

export const ApiUpdate = ({
    display = 'page',
    onCancel,
    onUpdated,
    zIndex = 200,
}: ApiUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<ApiVersion, FindVersionInput>(apiVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => apiInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleUpdated } = useUpdateActions<ApiVersion>(display, onCancel, onUpdated);
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<ApiVersion, ApiVersionUpdateInput>(apiVersionUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'UpdateApi',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!existing) {
                        PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                        return;
                    }
                    mutationWrapper<ApiVersion, ApiVersionUpdateInput>({
                        mutation,
                        input: shapeApiVersion.update(existing, values),
                        onSuccess: (data) => { handleUpdated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={apiVersionValidation.update({})}
            >
                {(formik) => <ApiForm
                    display={display}
                    isCreate={false}
                    isLoading={isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={existing?.root?.versions?.map(v => v.versionLabel) ?? []}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}