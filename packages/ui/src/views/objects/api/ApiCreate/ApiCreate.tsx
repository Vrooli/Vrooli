import { ApiVersion, ApiVersionCreateInput } from "@shared/consts";
import { apiVersionValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { apiVersionCreate } from "api/generated/endpoints/apiVersion_create";
import { useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { ApiForm } from "forms/ApiForm/ApiForm";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeApiVersion } from "utils/shape/models/apiVersion";
import { apiInitialValues } from "..";
import { ApiCreateProps } from "../types";

export const ApiCreate = ({
    display = 'page',
    onCancel,
    onCreated,
    zIndex = 200,
}: ApiCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => apiInitialValues(session), [session]);
    const { handleCancel, handleCreated } = useCreateActions<ApiVersion>(display, onCancel, onCreated);
    const [mutation, { loading: isLoading }] = useCustomMutation<ApiVersion, ApiVersionCreateInput>(apiVersionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'CreateApi',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    mutationWrapper<ApiVersion, ApiVersionCreateInput>({
                        mutation,
                        input: shapeApiVersion.create(values),
                        onSuccess: (data) => { handleCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={apiVersionValidation.create({})}
            >
                {(formik) => <ApiForm
                    display={display}
                    isCreate={true}
                    isLoading={isLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={[]}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}