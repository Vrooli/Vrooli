import { FindVersionInput, StandardVersion, StandardVersionUpdateInput } from "@shared/consts";
import { standardVersionFindOne } from "api/generated/endpoints/standardVersion_findOne";
import { standardVersionUpdate } from "api/generated/endpoints/standardVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { StandardForm } from "forms/StandardForm/StandardForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeStandardVersion } from "utils/shape/models/standardVersion";
import { standardInitialValues, validateStandardValues } from "..";
import { StandardUpdateProps } from "../types";

export const StandardUpdate = ({
    display = 'page',
    onCancel,
    onUpdated,
    zIndex = 200,
}: StandardUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<StandardVersion, FindVersionInput>(standardVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => standardInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleUpdated } = useUpdateActions<StandardVersion>(display, onCancel, onUpdated);
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<StandardVersion, StandardVersionUpdateInput>(standardVersionUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'UpdateStandard',
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
                    mutationWrapper<StandardVersion, StandardVersionUpdateInput>({
                        mutation,
                        input: shapeStandardVersion.update(existing, values),
                        onSuccess: (data) => { handleUpdated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateStandardValues(values, false)}
            >
                {(formik) => <StandardForm
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