import { ApiVersion, ApiVersionUpdateInput, FindByIdInput } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { apiVersionValidation } from '@shared/validation';
import { apiVersionFindOne } from "api/generated/endpoints/apiVersion_findOne";
import { apiVersionUpdate } from "api/generated/endpoints/apiVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from 'formik';
import { ApiForm } from "forms/ApiForm/ApiForm";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { getUserLanguages } from "utils/display/translationTools";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ApiUpdateProps } from "../types";

export const ApiUpdate = ({
    display = 'page',
    zIndex = 200,
}: ApiUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: apiVersion, loading: isReadLoading }] = useCustomLazyQuery<ApiVersion, FindByIdInput>(apiVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const { onCancel, onUpdated } = useUpdateActions<ApiVersion>();
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<ApiVersion, ApiVersionUpdateInput>(apiVersionUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'UpdateApi',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    __typename: 'Api' as const,
                    id: apiVersion?.id ?? uuid(),
                    translations: apiVersion?.translations ?? [{
                        id: uuid(),
                        language: getUserLanguages(session)[0],
                        details: '',
                        name: '',
                        summary: '',
                    }]
                    //TODO
                }}
                onSubmit={(values, helpers) => {
                    if (!apiVersion) {
                        PubSub.get().publishSnack({ messageKey: 'CouldNotReadApi', severity: 'Error' });
                        return;
                    }
                    // mutationWrapper<ApiVersion, ApiVersionUpdateInput>({
                    //     mutation,
                    //     input: shapeApiVersion.update(apiVersion, {
                    //         id: apiVersion.id,
                    //         isOpenToNewMembers: values.isOpenToNewMembers,
                    //         isPrivate: relationships.isPrivate,
                    //         resourceList: resourceList,
                    //         tags: tags,
                    //         translations: values.translationsUpdate,
                    //     }),
                    //     onSuccess: (data) => { onUpdated(data) },
                    //     onError: () => { formik.setSubmitting(false) },
                    // })
                }}
                validationSchema={apiVersionValidation.update({})}
            >
                {(formik) => <ApiForm
                    display={display}
                    isCreate={false}
                    isLoading={isReadLoading || isUpdateLoading}
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