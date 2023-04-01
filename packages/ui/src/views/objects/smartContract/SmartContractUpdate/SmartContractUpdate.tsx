import { FindVersionInput, SmartContractVersion, SmartContractVersionUpdateInput } from "@shared/consts";
import { mutationWrapper } from "api";
import { projectVersionUpdate } from "api/generated/endpoints/projectVersion_update";
import { smartContractVersionFindOne } from "api/generated/endpoints/smartContractVersion_findOne";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { SmartContractForm, smartContractInitialValues, validateSmartContractValues } from "forms/SmartContractForm/SmartContractForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeSmartContractVersion } from "utils/shape/models/smartContractVersion";
import { SmartContractUpdateProps } from "../types";

export const SmartContractUpdate = ({
    display = 'page',
    onCancel,
    onUpdated,
    zIndex = 200,
}: SmartContractUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<SmartContractVersion, FindVersionInput>(smartContractVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => smartContractInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleUpdated } = useUpdateActions<SmartContractVersion>(display, onCancel, onUpdated);
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<SmartContractVersion, SmartContractVersionUpdateInput>(projectVersionUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'UpdateSmartContract',
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
                    mutationWrapper<SmartContractVersion, SmartContractVersionUpdateInput>({
                        mutation,
                        input: shapeSmartContractVersion.update(existing, values),
                        onSuccess: (data) => { handleUpdated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateSmartContractValues(values, false)}
            >
                {(formik) => <SmartContractForm
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