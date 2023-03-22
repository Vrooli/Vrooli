import { ApiVersion, ApiVersionCreateInput } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { apiVersionValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { apiVersionCreate } from "api/generated/endpoints/apiVersion_create";
import { useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { ApiForm } from "forms/ApiForm/ApiForm";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { useContext, useRef } from "react";
import { getUserLanguages } from "utils/display/translationTools";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeApiVersion } from "utils/shape/models/apiVersion";
import { ApiCreateProps } from "../types";

export const ApiCreate = ({
    display = 'page',
    zIndex = 200,
}: ApiCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const { onCancel, onCreated } = useCreateActions<ApiVersion>();
    const [mutation, { loading: isLoading }] = useCustomMutation<ApiVersion, ApiVersionCreateInput>(apiVersionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateApi',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    __typename: 'ApiVersion' as const,
                    id: uuid(),
                    translations: [{
                        id: uuid(),
                        language: getUserLanguages(session)[0],
                        details: '',
                        name: '',
                        summary: '',
                    }]
                    //TODO
                }}
                onSubmit={(values, helpers) => {
                    mutationWrapper<ApiVersion, ApiVersionCreateInput>({
                        mutation,
                        input: shapeApiVersion.create(values),
                        onSuccess: (data) => { onCreated(data) },
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
                    onCancel={onCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}